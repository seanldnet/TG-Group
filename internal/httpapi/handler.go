package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/seanldnet/tg-group/internal/auth"
	"github.com/seanldnet/tg-group/internal/bots"
)

type Handler struct {
	logger *slog.Logger
	auth *auth.Service
	bots *bots.Service
}

func NewHandler(logger *slog.Logger, authService *auth.Service, botService *bots.Service) http.Handler {
	if logger == nil { logger = slog.Default() }
	h := &Handler{logger:logger,auth:authService,bots:botService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz",h.health)
	mux.HandleFunc("POST /api/v1/auth/register",h.register)
	mux.HandleFunc("POST /api/v1/auth/login",h.login)
	mux.HandleFunc("POST /api/v1/auth/logout",h.logout)
	mux.HandleFunc("GET /api/v1/auth/me",h.me)
	mux.HandleFunc("GET /api/v1/bots",h.listBots)
	mux.HandleFunc("GET /api/v1/groups",h.listGroups)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/events",h.listGroupEvents)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/members",h.listGroupMembers)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/permissions",h.groupBotPermissions)
	mux.HandleFunc("POST /api/v1/groups/{groupID}/members/{userID}/action",h.moderateGroupMember)
	mux.HandleFunc("POST /api/v1/bots",h.addBot)
	mux.HandleFunc("PATCH /api/v1/bots/{botID}/status",h.setBotStatus)
	mux.HandleFunc("PUT /api/v1/bots/{botID}/token",h.rotateBotToken)
	mux.HandleFunc("DELETE /api/v1/bots/{botID}",h.removeBot)
	return mux
}

type healthResponse struct {
	Status string `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
type credentialsRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}
type registerRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
	DisplayName string `json:"display_name"`
}

func (h *Handler) health(w http.ResponseWriter,r *http.Request) {
	w.Header().Set("Content-Type","application/json; charset=utf-8")
	w.Header().Set("Cache-Control","no-store")
	w.WriteHeader(http.StatusOK)
	if err:=json.NewEncoder(w).Encode(healthResponse{Status:"ok",Timestamp:time.Now().UTC()});err!=nil {
		h.logger.Error("write health response","error",err)
	}
}

func (h *Handler) register(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	var input registerRequest
	if !decodeJSON(w,r,&input) { return }
	session,err:=h.auth.Register(r.Context(),input.Email,input.Password,input.DisplayName,r.UserAgent(),r.RemoteAddr)
	if err!=nil { h.writeAuthError(w,err);return }
	writeJSON(w,http.StatusCreated,session)
}

func (h *Handler) login(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	var input credentialsRequest
	if !decodeJSON(w,r,&input) { return }
	session,err:=h.auth.Login(r.Context(),input.Email,input.Password,r.UserAgent(),r.RemoteAddr)
	if err!=nil { h.writeAuthError(w,err);return }
	writeJSON(w,http.StatusOK,session)
}

func (h *Handler) logout(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	token:=bearerToken(r)
	if token=="" { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	if err:=h.auth.Logout(r.Context(),token);err!=nil { h.logger.Error("logout failed","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return }
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	writeJSON(w,http.StatusOK,user)
}

func (h *Handler) requireAuthStore(w http.ResponseWriter) bool {
	if h.auth==nil { writeError(w,http.StatusServiceUnavailable,"authentication service unavailable: configure DATABASE_URL");return false }
	return true
}

func (h *Handler) writeAuthError(w http.ResponseWriter,err error) {
	switch {
	case errors.Is(err,auth.ErrInvalidInput): writeError(w,http.StatusBadRequest,"email, display name or password is invalid; password must be 10-72 bytes")
	case errors.Is(err,auth.ErrEmailTaken): writeError(w,http.StatusConflict,"email already registered")
	case errors.Is(err,auth.ErrInvalidCredentials): writeError(w,http.StatusUnauthorized,"invalid email or password")
	case errors.Is(err,auth.ErrAccountInactive): writeError(w,http.StatusForbidden,"account is not active")
	default: h.logger.Error("authentication request failed","error",err);writeError(w,http.StatusInternalServerError,"internal server error")
	}
}

func decodeJSON(w http.ResponseWriter,r *http.Request,dst any) bool {
	r.Body=http.MaxBytesReader(w,r.Body,1<<20)
	decoder:=json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err:=decoder.Decode(dst);err!=nil { writeError(w,http.StatusBadRequest,"invalid JSON request");return false }
	if err:=decoder.Decode(&struct{}{});err!=io.EOF { writeError(w,http.StatusBadRequest,"request must contain one JSON object");return false }
	return true
}

func bearerToken(r *http.Request) string {
	header:=r.Header.Get("Authorization")
	parts:=strings.SplitN(header," ",2)
	if len(parts)!=2 || !strings.EqualFold(parts[0],"Bearer") { return "" }
	return strings.TrimSpace(parts[1])
}

func writeJSON(w http.ResponseWriter,status int,value any) {
	w.Header().Set("Content-Type","application/json; charset=utf-8")
	w.Header().Set("Cache-Control","no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter,status int,message string) {
	writeJSON(w,status,map[string]string{"error":message})
}

type addBotRequest struct { Token string `json:"token"` }
type botStatusRequest struct { Status string `json:"status"` }

func (h *Handler) botServicesReady(w http.ResponseWriter) bool {
	if h.auth == nil || h.bots == nil {
		writeError(w,http.StatusServiceUnavailable,"Bot management unavailable: configure DATABASE_URL and BOT_TOKEN_ENCRYPTION_KEY")
		return false
	}
	return true
}

func (h *Handler) currentUser(w http.ResponseWriter,r *http.Request)(auth.User,bool){
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return auth.User{},false }
	return user,true
}

func (h *Handler) listGroups(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	result,err:=h.bots.ListGroups(r.Context(),user.TenantID)
	if err!=nil{h.logger.Error("list Telegram groups","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":result})
}

func (h *Handler) listGroupEvents(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	limit:=50
	if raw:=r.URL.Query().Get("limit");raw!=""{
		value,err:=strconv.Atoi(raw)
		if err!=nil || value<1 || value>100{writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return}
		limit=value
	}
	result,err:=h.bots.ListGroupEvents(r.Context(),user.TenantID,r.PathValue("groupID"),limit)
	if err!=nil{h.logger.Error("list Telegram group message events","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":result})
}


func (h *Handler) listGroupMembers(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	limit:=100
	if raw:=r.URL.Query().Get("limit");raw!=""{
		value,err:=strconv.Atoi(raw)
		if err!=nil || value<1 || value>200{writeError(w,http.StatusBadRequest,"limit must be between 1 and 200");return}
		limit=value
	}
	result,err:=h.bots.ListGroupMembers(r.Context(),user.TenantID,r.PathValue("groupID"),limit)
	if err!=nil{h.logger.Error("list Telegram group members","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":result})
}


type memberActionRequest struct {
 Action string `json:"action"`
 DurationMinutes int `json:"duration_minutes"`
}
func (h *Handler) groupBotPermissions(w http.ResponseWriter,r *http.Request){
 if !h.botServicesReady(w){return}
 user,ok:=h.currentUser(w,r);if !ok{return}
 result,err:=h.bots.CheckGroupBotPermissions(r.Context(),user.TenantID,r.PathValue("groupID"))
 if err!=nil{h.writeBotError(w,err);return}
 writeJSON(w,http.StatusOK,result)
}
func (h *Handler) moderateGroupMember(w http.ResponseWriter,r *http.Request){
 if !h.botServicesReady(w){return}
 user,ok:=h.currentUser(w,r);if !ok{return}
 targetID,err:=strconv.ParseInt(r.PathValue("userID"),10,64)
 if err!=nil || targetID<=0{writeError(w,http.StatusBadRequest,"userID must be a positive Telegram user ID");return}
 var input memberActionRequest
 if !decodeJSON(w,r,&input){return}
 if err:=h.bots.ModerateGroupMember(r.Context(),user.TenantID,user.ID,r.PathValue("groupID"),targetID,input.Action,input.DurationMinutes);err!=nil{h.writeBotError(w,err);return}
 writeJSON(w,http.StatusOK,map[string]any{"ok":true,"action":input.Action,"telegram_user_id":targetID})
}

func (h *Handler) listBots(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	result,err:=h.bots.List(r.Context(),user.TenantID)
	if err!=nil{h.logger.Error("list Telegram bots","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":result})
}

func (h *Handler) addBot(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	var input addBotRequest
	if !decodeJSON(w,r,&input){return}
	result,err:=h.bots.Add(r.Context(),user.TenantID,user.ID,input.Token)
	if err!=nil{h.writeBotError(w,err);return}
	writeJSON(w,http.StatusCreated,result)
}

func (h *Handler) setBotStatus(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	var input botStatusRequest
	if !decodeJSON(w,r,&input){return}
	result,err:=h.bots.SetStatus(r.Context(),user.TenantID,user.ID,r.PathValue("botID"),input.Status)
	if err!=nil{h.writeBotError(w,err);return}
	writeJSON(w,http.StatusOK,result)
}

func (h *Handler) rotateBotToken(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	var input addBotRequest
	if !decodeJSON(w,r,&input){return}
	result,err:=h.bots.RotateToken(r.Context(),user.TenantID,user.ID,r.PathValue("botID"),input.Token)
	if err!=nil{h.writeBotError(w,err);return}
	writeJSON(w,http.StatusOK,result)
}

func (h *Handler) removeBot(w http.ResponseWriter,r *http.Request){
	if !h.botServicesReady(w){return}
	user,ok:=h.currentUser(w,r);if !ok{return}
	if err:=h.bots.Remove(r.Context(),user.TenantID,user.ID,r.PathValue("botID"));err!=nil{h.writeBotError(w,err);return}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeBotError(w http.ResponseWriter,err error){
	switch{
	case errors.Is(err,bots.ErrInvalidToken):writeError(w,http.StatusUnprocessableEntity,"Telegram rejected this bot token")
	case errors.Is(err,bots.ErrTokenTaken):writeError(w,http.StatusConflict,"this Telegram bot is already registered")
	case errors.Is(err,bots.ErrNotFound):writeError(w,http.StatusNotFound,"Bot not found")
	case errors.Is(err,bots.ErrInvalidStatus):writeError(w,http.StatusBadRequest,"status must be active or disabled")
	case errors.Is(err,bots.ErrIdentityChanged):writeError(w,http.StatusConflict,"replacement token must belong to the same Telegram bot")
	case errors.Is(err,bots.ErrTelegramUnavailable):writeError(w,http.StatusBadGateway,"Telegram service is temporarily unavailable")
	case errors.Is(err,bots.ErrInsufficientBotRights):writeError(w,http.StatusForbidden,"Bot must be an administrator with permission to restrict members")
	case errors.Is(err,bots.ErrTargetIsAdministrator):writeError(w,http.StatusForbidden,"cannot moderate a group administrator or creator")
	case errors.Is(err,bots.ErrInvalidMemberAction):writeError(w,http.StatusBadRequest,"invalid member action or duration")
	case errors.Is(err,bots.ErrTelegramOperation):writeError(w,http.StatusUnprocessableEntity,err.Error())
	case errors.Is(err,bots.ErrGroupPermissionsUnavailable):writeError(w,http.StatusConflict,"group default permissions are unavailable; unmute was not applied")
	default:h.logger.Error("Telegram bot operation failed","error",err);writeError(w,http.StatusInternalServerError,"internal server error")
	}
}

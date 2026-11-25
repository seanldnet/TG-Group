package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/seanldnet/tg-group/internal/auth"
)

type Handler struct {
	logger *slog.Logger
	auth *auth.Service
}

func NewHandler(logger *slog.Logger, authService *auth.Service) http.Handler {
	if logger == nil { logger = slog.Default() }
	h := &Handler{logger:logger,auth:authService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz",h.health)
	mux.HandleFunc("POST /api/v1/auth/register",h.register)
	mux.HandleFunc("POST /api/v1/auth/login",h.login)
	mux.HandleFunc("POST /api/v1/auth/logout",h.logout)
	mux.HandleFunc("GET /api/v1/auth/me",h.me)
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

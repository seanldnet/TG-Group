package httpapi

import (
	"encoding/json"
	"crypto/sha256"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/seanldnet/tg-group/internal/auth"
	"github.com/seanldnet/tg-group/internal/bots"
	"github.com/seanldnet/tg-group/internal/platformsettings"
	"github.com/seanldnet/tg-group/internal/telegramauthbot"
	"github.com/seanldnet/tg-group/internal/payments"
)

type Handler struct {
	logger *slog.Logger
	auth *auth.Service
	bots *bots.Service
	captcha *CaptchaVerifier
	telegramAuthBotUsername string
	platformSettings *platformsettings.Service
	telegramAuthBotManager *telegramauthbot.Manager
	paymentChannels *payments.ConfigStore
	recharge *payments.RechargeService
	paymentPublicBaseURL string
}

func NewHandler(logger *slog.Logger, authService *auth.Service, botService *bots.Service, captcha *CaptchaVerifier) http.Handler {
	return NewHandlerWithTelegramAuthBot(logger, authService, botService, captcha, "")
}

func NewHandlerWithTelegramAuthBot(logger *slog.Logger, authService *auth.Service, botService *bots.Service, captcha *CaptchaVerifier, telegramBotUsername string) http.Handler {
	return NewHandlerWithPlatformBot(logger,authService,botService,captcha,telegramBotUsername,nil,nil)
}

func NewHandlerWithPlatformBot(logger *slog.Logger, authService *auth.Service, botService *bots.Service, captcha *CaptchaVerifier, telegramBotUsername string, settings *platformsettings.Service, manager *telegramauthbot.Manager) http.Handler {
	return NewHandlerWithPayments(logger,authService,botService,captcha,telegramBotUsername,settings,manager,nil,nil,"")
}

func NewHandlerWithPayments(logger *slog.Logger, authService *auth.Service, botService *bots.Service, captcha *CaptchaVerifier, telegramBotUsername string, settings *platformsettings.Service, manager *telegramauthbot.Manager, paymentChannels *payments.ConfigStore, recharge *payments.RechargeService, paymentPublicBaseURL string) http.Handler {
	if logger == nil { logger = slog.Default() }
	h := &Handler{logger:logger,auth:authService,bots:botService,captcha:captcha,telegramAuthBotUsername:telegramBotUsername,platformSettings:settings,telegramAuthBotManager:manager,paymentChannels:paymentChannels,recharge:recharge,paymentPublicBaseURL:strings.TrimRight(strings.TrimSpace(paymentPublicBaseURL),"/")}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz",h.health)
	mux.HandleFunc("GET /api/v1/security/captcha",h.captchaConfig)
	mux.HandleFunc("GET /api/v1/admin/platform/telegram-auth-bot",h.platformTelegramBotStatus)
	mux.HandleFunc("PUT /api/v1/admin/platform/telegram-auth-bot",h.updatePlatformTelegramBot)
	mux.HandleFunc("DELETE /api/v1/admin/platform/telegram-auth-bot",h.clearPlatformTelegramBot)
	mux.HandleFunc("GET /api/v1/admin/users",h.adminUsers)
	mux.HandleFunc("PATCH /api/v1/admin/users/{userID}/status",h.updateAdminUserStatus)
	mux.HandleFunc("POST /api/v1/auth/register",h.register)
	mux.HandleFunc("POST /api/v1/auth/login",h.login)
	mux.HandleFunc("POST /api/v1/auth/refresh",h.refresh)
	mux.HandleFunc("POST /api/v1/auth/password/change",h.changePassword)
	mux.HandleFunc("POST /api/v1/auth/logout",h.logout)
	mux.HandleFunc("GET /api/v1/auth/me",h.me)
	mux.HandleFunc("GET /api/v1/account/agent",h.agentAccount)
	mux.HandleFunc("GET /api/v1/account/wallet/ledger",h.walletLedger)
	mux.HandleFunc("POST /api/v1/account/agent/upgrade",h.upgradeAgent)
	mux.HandleFunc("GET /api/v1/account/agent/orders",h.agentUpgradeOrders)
	mux.HandleFunc("GET /api/v1/admin/agent-levels",h.adminAgentLevels)
	mux.HandleFunc("GET /api/v1/admin/payment-channels",h.adminPaymentChannels)
	mux.HandleFunc("PUT /api/v1/admin/payment-channels/{channelCode}",h.updateAdminPaymentChannel)
	mux.HandleFunc("POST /api/v1/account/wallet/recharge",h.createWalletRecharge)
	mux.HandleFunc("GET /api/v1/account/wallet/recharges",h.listWalletRecharges)
	mux.HandleFunc("POST /api/v1/payments/epusdt/gmpay/notify",h.gmpayNotify)
	mux.HandleFunc("POST /api/v1/admin/agent-levels",h.createAdminAgentLevel)
	mux.HandleFunc("PATCH /api/v1/admin/agent-levels/{levelID}",h.updateAdminAgentLevel)
	mux.HandleFunc("PATCH /api/v1/auth/profile",h.updateProfile)
	mux.HandleFunc("GET /api/v1/auth/security-events",h.securityEvents)
	mux.HandleFunc("GET /api/v1/auth/sessions",h.listAuthSessions)
	mux.HandleFunc("DELETE /api/v1/auth/sessions/{sessionID}",h.revokeAuthSession)
	mux.HandleFunc("GET /api/v1/auth/telegram",h.telegramIdentity)
	mux.HandleFunc("POST /api/v1/auth/telegram/link",h.createTelegramLink)
	mux.HandleFunc("DELETE /api/v1/auth/telegram",h.unlinkTelegramIdentity)
	mux.HandleFunc("GET /api/v1/bots",h.listBots)
	mux.HandleFunc("GET /api/v1/groups",h.listGroups)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/events",h.listGroupEvents)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/members",h.listGroupMembers)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/permissions",h.groupBotPermissions)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/rules",h.listGroupRules)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/rule-events",h.listGroupRuleEvents)
	mux.HandleFunc("POST /api/v1/groups/{groupID}/rules",h.createGroupRule)
	mux.HandleFunc("PUT /api/v1/groups/{groupID}/rules/{ruleID}",h.updateGroupRule)
	mux.HandleFunc("DELETE /api/v1/groups/{groupID}/rules/{ruleID}",h.deleteGroupRule)
	mux.HandleFunc("POST /api/v1/groups/{groupID}/members/{userID}/action",h.moderateGroupMember)
	mux.HandleFunc("POST /api/v1/groups/{groupID}/members/actions/bulk",h.bulkModerateGroupMembers)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/member-lists",h.listGroupMemberLists)
	mux.HandleFunc("POST /api/v1/groups/{groupID}/member-lists",h.upsertGroupMemberList)
	mux.HandleFunc("DELETE /api/v1/groups/{groupID}/member-lists/{userID}",h.deleteGroupMemberList)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/members/{userID}/warnings",h.listGroupMemberWarnings)
	mux.HandleFunc("POST /api/v1/groups/{groupID}/members/{userID}/warnings",h.addGroupMemberWarning)
	mux.HandleFunc("DELETE /api/v1/groups/{groupID}/members/{userID}/warnings",h.clearGroupMemberWarnings)
	mux.HandleFunc("GET /api/v1/groups/{groupID}/members/{userID}/warning-escalations",h.listWarningEscalations)
	mux.HandleFunc("POST /api/v1/groups/{groupID}/members/{userID}/warning-escalations/{executionID}/retry",h.retryWarningEscalation)
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
	CaptchaToken string `json:"captcha_token"`
}
type refreshRequest struct { RefreshToken string `json:"refresh_token"` }
type updateProfileRequest struct { DisplayName string `json:"display_name"` }
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword string `json:"new_password"`
}
type registerRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
	DisplayName string `json:"display_name"`
	CaptchaToken string `json:"captcha_token"`
}

func (h *Handler) captchaConfig(w http.ResponseWriter,r *http.Request) {
	if h.captcha == nil { writeJSON(w,http.StatusOK,map[string]any{"enabled":false});return }
	writeJSON(w,http.StatusOK,h.captcha.PublicConfig())
}
func (h *Handler) verifyCaptcha(r *http.Request,token string) error {
	if h.captcha == nil { return nil }
	return h.captcha.Verify(r.Context(),token)
}
func (h *Handler) writeCaptchaError(w http.ResponseWriter,err error) {
	if errors.Is(err,ErrCaptchaRejected) {
		writeError(w,http.StatusBadRequest,"人机验证未通过，请重新完成验证")
		return
	}
	h.logger.Error("CAP verification unavailable","error",err)
	writeError(w,http.StatusServiceUnavailable,"人机验证服务暂时不可用，请稍后重试")
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
	if err:=h.verifyCaptcha(r,input.CaptchaToken);err!=nil { h.writeCaptchaError(w,err);return }
	session,err:=h.auth.Register(r.Context(),input.Email,input.Password,input.DisplayName,r.UserAgent(),r.RemoteAddr)
	if err!=nil { h.writeAuthError(w,err);return }
	writeJSON(w,http.StatusCreated,session)
}

func (h *Handler) login(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	var input credentialsRequest
	if !decodeJSON(w,r,&input) { return }
	if err:=h.verifyCaptcha(r,input.CaptchaToken);err!=nil { h.writeCaptchaError(w,err);return }
	session,err:=h.auth.Login(r.Context(),input.Email,input.Password,r.UserAgent(),r.RemoteAddr)
	if err!=nil { h.writeAuthError(w,err);return }
	writeJSON(w,http.StatusOK,session)
}

func (h *Handler) refresh(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	var input refreshRequest
	if !decodeJSON(w,r,&input) { return }
	session,err:=h.auth.Refresh(r.Context(),input.RefreshToken,r.UserAgent(),r.RemoteAddr)
	if err!=nil {
		if errors.Is(err,auth.ErrRefreshUnauthorized) || errors.Is(err,auth.ErrRefreshReuse) {
			writeError(w,http.StatusUnauthorized,"登录会话已失效，请重新登录")
			return
		}
		h.logger.Error("refresh session failed","error",err)
		writeError(w,http.StatusInternalServerError,"internal server error")
		return
	}
	writeJSON(w,http.StatusOK,session)
}

func (h *Handler) changePassword(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	token := bearerToken(r)
	if token == "" { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	var input changePasswordRequest
	if !decodeJSON(w,r,&input) { return }
	session,err := h.auth.ChangePassword(r.Context(),token,input.CurrentPassword,input.NewPassword,r.UserAgent(),r.RemoteAddr)
	if err != nil {
		switch {
		case errors.Is(err,auth.ErrInvalidInput):
			writeError(w,http.StatusBadRequest,"新密码必须为 10-72 字节，且不能与当前密码相同")
		case errors.Is(err,auth.ErrCurrentPasswordInvalid):
			writeError(w,http.StatusUnauthorized,"当前密码不正确")
		case errors.Is(err,auth.ErrUnauthorized):
			writeError(w,http.StatusUnauthorized,"登录会话已失效，请重新登录")
		case errors.Is(err,auth.ErrAccountInactive):
			writeError(w,http.StatusForbidden,"账号当前不可用")
		default:
			h.logger.Error("change password failed","error",err)
			writeError(w,http.StatusInternalServerError,"internal server error")
		}
		return
	}
	writeJSON(w,http.StatusOK,session)
}

func (h *Handler) logout(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	token:=bearerToken(r)
	if token=="" { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	if err:=h.auth.Logout(r.Context(),token,r.RemoteAddr);err!=nil { h.logger.Error("logout failed","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return }
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateProfile(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	user,ok:=h.currentUser(w,r);if !ok{return}
	var input updateProfileRequest
	if !decodeJSON(w,r,&input){return}
	updated,err:=h.auth.UpdateDisplayName(r.Context(),user.ID,input.DisplayName)
	switch {
	case err==nil: writeJSON(w,http.StatusOK,updated)
	case errors.Is(err,auth.ErrInvalidDisplayName): writeError(w,http.StatusBadRequest,"显示名称不能为空且不能超过 40 个字符")
	case errors.Is(err,auth.ErrProfileUserNotFound): writeError(w,http.StatusNotFound,"账户不存在或不可用")
	default: h.logger.Error("update account profile","error",err);writeError(w,http.StatusInternalServerError,"更新个人资料失败")
	}
}

func (h *Handler) me(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	writeJSON(w,http.StatusOK,user)
}

func (h *Handler) listAuthSessions(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	token:=bearerToken(r)
	if token=="" { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	limit:=20
	if raw:=r.URL.Query().Get("limit");raw!="" {
		value,err:=strconv.Atoi(raw)
		if err!=nil || value<1 || value>100 { writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return }
		limit=value
	}
	sessions,err:=h.auth.ListSessions(r.Context(),token,limit)
	if err!=nil {
		if errors.Is(err,auth.ErrUnauthorized) { writeError(w,http.StatusUnauthorized,"unauthorized");return }
		h.logger.Error("list account sessions","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return
	}
	writeJSON(w,http.StatusOK,map[string]any{"items":sessions})
}

func (h *Handler) revokeAuthSession(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	token:=bearerToken(r)
	if token=="" { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	err:=h.auth.RevokeSession(r.Context(),token,r.PathValue("sessionID"),r.RemoteAddr)
	if err!=nil {
		switch {
		case errors.Is(err,auth.ErrUnauthorized): writeError(w,http.StatusUnauthorized,"unauthorized")
		case errors.Is(err,auth.ErrCurrentSessionRevoke): writeError(w,http.StatusBadRequest,"不能撤销当前正在使用的登录会话")
		case errors.Is(err,auth.ErrSessionNotFound): writeError(w,http.StatusNotFound,"登录会话不存在或已失效")
		default: h.logger.Error("revoke account session","error",err);writeError(w,http.StatusInternalServerError,"internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) securityEvents(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	limit:=20
	if raw:=r.URL.Query().Get("limit");raw!="" {
		value,parseErr:=strconv.Atoi(raw)
		if parseErr!=nil || value<1 || value>100 { writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return }
		limit=value
	}
	events,err:=h.auth.ListSecurityEvents(r.Context(),user.ID,limit)
	if err!=nil { h.logger.Error("list account security events","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return }
	writeJSON(w,http.StatusOK,map[string]any{"items":events})
}

func (h *Handler) telegramIdentity(w http.ResponseWriter,r *http.Request) {
    if !h.requireAuthStore(w) { return }
    user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
    if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
    identity,err:=h.auth.TelegramIdentity(r.Context(),user.ID)
    if err!=nil { h.logger.Error("read Telegram identity","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return }
    writeJSON(w,http.StatusOK,map[string]any{"linked":identity.ID!=nil,"telegram":identity})
}

func (h *Handler) createTelegramLink(w http.ResponseWriter,r *http.Request) {
    if !h.requireAuthStore(w) { return }
    user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
    if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
    username:=h.telegramAuthBotUsername
	if h.telegramAuthBotManager!=nil {if configured,_:=h.telegramAuthBotManager.Status();configured!="" {username=configured}}
	if username=="" { writeError(w,http.StatusServiceUnavailable,"平台 Telegram 身份 Bot 尚未配置");return }
    identity,err:=h.auth.TelegramIdentity(r.Context(),user.ID)
    if err!=nil { h.logger.Error("read Telegram identity before link","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return }
    if identity.ID!=nil { writeError(w,http.StatusConflict,"该账号已经绑定 Telegram");return }
    challenge,err:=h.auth.CreateTelegramLinkChallenge(r.Context(),user.ID)
    if err!=nil { h.logger.Error("create Telegram link challenge","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return }
    writeJSON(w,http.StatusCreated,map[string]any{"url":"https://t.me/"+username+"?start="+challenge.Token,"expires_at":challenge.ExpiresAt})
}

func (h *Handler) unlinkTelegramIdentity(w http.ResponseWriter,r *http.Request) {
    if !h.requireAuthStore(w) { return }
    user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
    if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
    if err:=h.auth.UnlinkTelegramIdentity(r.Context(),user.ID);err!=nil {
        if errors.Is(err,auth.ErrTelegramNotLinked) { writeError(w,http.StatusConflict,"当前账号尚未绑定 Telegram");return }
        h.logger.Error("unlink Telegram identity","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) agentAccount(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	account,err:=h.auth.GetAgentAccount(r.Context(),user.ID)
	if err!=nil { h.logger.Error("load agent account","error",err);writeError(w,http.StatusInternalServerError,"读取代理账户失败");return }
	writeJSON(w,http.StatusOK,account)
}

func (h *Handler) walletLedger(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	limit,offset:=20,0
	if raw:=r.URL.Query().Get("limit");raw!="" {
		value,parseErr:=strconv.Atoi(raw)
		if parseErr!=nil || value<1 || value>100 { writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return }
		limit=value
	}
	if raw:=r.URL.Query().Get("offset");raw!="" {
		value,parseErr:=strconv.Atoi(raw)
		if parseErr!=nil || value<0 || value>1000000 { writeError(w,http.StatusBadRequest,"offset must be between 0 and 1000000");return }
		offset=value
	}
	items,total,err:=h.auth.ListWalletLedger(r.Context(),user.ID,limit,offset)
	if err!=nil { h.logger.Error("list wallet ledger","error",err);writeError(w,http.StatusInternalServerError,"读取钱包流水失败");return }
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"total":total,"limit":limit,"offset":offset})
}

func (h *Handler) agentUpgradeOrders(w http.ResponseWriter,r *http.Request){
	if !h.requireAuthStore(w){return}
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil{writeError(w,http.StatusUnauthorized,"unauthorized");return}
	limit,offset:=20,0
	if raw:=r.URL.Query().Get("limit");raw!=""{
		value,parseErr:=strconv.Atoi(raw)
		if parseErr!=nil||value<1||value>100{writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return}
		limit=value
	}
	if raw:=r.URL.Query().Get("offset");raw!=""{
		value,parseErr:=strconv.Atoi(raw)
		if parseErr!=nil||value<0||value>1000000{writeError(w,http.StatusBadRequest,"offset must be between 0 and 1000000");return}
		offset=value
	}
	items,total,err:=h.auth.ListAgentUpgradeOrders(r.Context(),user.ID,limit,offset)
	if err!=nil{h.logger.Error("list agent upgrade orders","error",err);writeError(w,http.StatusInternalServerError,"读取代理升级订单失败");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"total":total,"limit":limit,"offset":offset})
}

type agentUpgradeRequest struct {
	TargetLevelID string `json:"target_level_id"`
}

func (h *Handler) upgradeAgent(w http.ResponseWriter,r *http.Request) {
	if !h.requireAuthStore(w) { return }
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil { writeError(w,http.StatusUnauthorized,"unauthorized");return }
	var input agentUpgradeRequest
	if !decodeJSON(w,r,&input) { return }
	if strings.TrimSpace(input.TargetLevelID)=="" { writeError(w,http.StatusBadRequest,"target_level_id is required");return }
	order,err:=h.auth.UpgradeAgentWithWallet(r.Context(),user.ID,input.TargetLevelID,r.Header.Get("Idempotency-Key"))
	switch {
	case err==nil: writeJSON(w,http.StatusOK,order)
	case errors.Is(err,auth.ErrInvalidIdempotencyKey): writeError(w,http.StatusBadRequest,"请提供有效的 Idempotency-Key 请求头")
	case errors.Is(err,auth.ErrInsufficientWalletBalance): writeError(w,http.StatusConflict,"USDT 余额不足，请先充值")
	case errors.Is(err,auth.ErrAgentLevelNotUpgradeable): writeError(w,http.StatusConflict,"目标代理等级不可升级或已停用")
	case errors.Is(err,auth.ErrAgentProfileSuspended): writeError(w,http.StatusForbidden,"代理账户当前已暂停")
	default: h.logger.Error("upgrade agent with wallet failed","error",err);writeError(w,http.StatusInternalServerError,"代理升级失败")
	}
}

type adminAgentLevelRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
	PriceUSDT int64 `json:"price_usdt"`
	SortOrder int32 `json:"sort_order"`
	Benefits map[string]any `json:"benefits"`
	Enabled bool `json:"enabled"`
}

func (h *Handler) adminAgentLevelUser(w http.ResponseWriter,r *http.Request)(auth.User,bool){
	if !h.requireAuthStore(w){return auth.User{},false}
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r))
	if err!=nil{writeError(w,http.StatusUnauthorized,"unauthorized");return auth.User{},false}
	if user.Role!="admin"{writeError(w,http.StatusForbidden,"platform administrator permission required");return auth.User{},false}
	return user,true
}

func (h *Handler) adminAgentLevels(w http.ResponseWriter,r *http.Request){
	if _,ok:=h.adminAgentLevelUser(w,r);!ok{return}
	items,err:=h.auth.ListAdminAgentLevels(r.Context())
	if err!=nil{h.logger.Error("list admin agent levels","error",err);writeError(w,http.StatusInternalServerError,"读取代理等级失败");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":items})
}

func (h *Handler) createAdminAgentLevel(w http.ResponseWriter,r *http.Request){
	user,ok:=h.adminAgentLevelUser(w,r);if !ok{return}
	var input adminAgentLevelRequest
	if !decodeJSON(w,r,&input){return}
	item,err:=h.auth.CreateAdminAgentLevel(r.Context(),user.ID,input.Code,input.Name,input.PriceUSDT,input.SortOrder,input.Benefits)
	switch{
	case err==nil:writeJSON(w,http.StatusCreated,item)
	case errors.Is(err,auth.ErrInvalidAgentLevel):writeError(w,http.StatusBadRequest,"等级标识、名称、排序或 USDT 整数价格无效")
	case errors.Is(err,auth.ErrAgentLevelConflict):writeError(w,http.StatusConflict,"代理等级标识已存在")
	default:h.logger.Error("create admin agent level","error",err);writeError(w,http.StatusInternalServerError,"创建代理等级失败")
	}
}

func (h *Handler) updateAdminAgentLevel(w http.ResponseWriter,r *http.Request){
	user,ok:=h.adminAgentLevelUser(w,r);if !ok{return}
	var input adminAgentLevelRequest
	if !decodeJSON(w,r,&input){return}
	item,err:=h.auth.UpdateAdminAgentLevel(r.Context(),user.ID,r.PathValue("levelID"),input.Name,input.PriceUSDT,input.SortOrder,input.Benefits,input.Enabled)
	switch{
	case err==nil:writeJSON(w,http.StatusOK,item)
	case errors.Is(err,auth.ErrInvalidAgentLevel):writeError(w,http.StatusBadRequest,"等级名称、排序或 USDT 整数价格无效")
	case errors.Is(err,auth.ErrAgentLevelNotUpgradeable):writeError(w,http.StatusNotFound,"代理等级不存在")
	default:h.logger.Error("update admin agent level","error",err);writeError(w,http.StatusInternalServerError,"更新代理等级失败")
	}
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
	case errors.Is(err,auth.ErrLoginRateLimited): w.Header().Set("Retry-After","900");writeError(w,http.StatusTooManyRequests,"登录失败次数过多，请 15 分钟后再试")
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


type platformTelegramBotRequest struct { Token string `json:"token"`; Username string `json:"username"` }

func (h *Handler) platformAdmin(w http.ResponseWriter,r *http.Request)(auth.User,bool){
	if h.auth==nil {writeError(w,http.StatusServiceUnavailable,"authentication service unavailable");return auth.User{},false}
	user,ok:=h.currentUser(w,r);if !ok{return auth.User{},false}
	if user.Role!="admin" {writeError(w,http.StatusForbidden,"platform administrator permission required");return auth.User{},false}
	if h.platformSettings==nil {writeError(w,http.StatusServiceUnavailable,"platform settings unavailable: configure BOT_TOKEN_ENCRYPTION_KEY and DATABASE_URL");return auth.User{},false}
	return user,true
}
func (h *Handler) platformTelegramBotStatus(w http.ResponseWriter,r *http.Request){
	if _,ok:=h.platformAdmin(w,r);!ok{return}
	status,err:=h.platformSettings.Status(r.Context());if err!=nil{h.logger.Error("read platform Telegram Bot status","error",err);if errors.Is(err,platformsettings.ErrSchemaNotReady){writeError(w,http.StatusServiceUnavailable,"平台 Bot 数据库迁移尚未完成，请更新并重启 API 服务");return};writeError(w,http.StatusInternalServerError,"读取平台 Bot 设置失败，请检查 API 服务日志");return}
	username,running:="",false
	if h.telegramAuthBotManager!=nil {username,running=h.telegramAuthBotManager.Status()}
	if status.Configured && username=="" {username=status.Username}
	configured:=status.Configured || username!=""
	writeJSON(w,http.StatusOK,map[string]any{"configured":configured,"managed":status.Managed,"username":username,"running":running,"updated_at":status.UpdatedAt})
}
func (h *Handler) updatePlatformTelegramBot(w http.ResponseWriter,r *http.Request){
	user,ok:=h.platformAdmin(w,r);if !ok{return}
	var input platformTelegramBotRequest;if !decodeJSON(w,r,&input){return}
	verifyCtx,cancel:=context.WithTimeout(r.Context(),12*time.Second);defer cancel()
	if err:=h.platformSettings.VerifyToken(verifyCtx,input.Token,input.Username);err!=nil {
		switch {case errors.Is(err,platformsettings.ErrInvalidConfig):writeError(w,http.StatusBadRequest,"Bot Token 与 Username 格式不匹配");case errors.Is(err,platformsettings.ErrTelegramRejected):writeError(w,http.StatusUnprocessableEntity,"Telegram 不接受该 Bot Token");default:writeError(w,http.StatusBadGateway,"暂时无法连接 Telegram 验证 Bot，请稍后重试")};return
	}
	if err:=h.platformSettings.Save(r.Context(),input.Token,input.Username,user.ID);err!=nil{
		if errors.Is(err,platformsettings.ErrInvalidConfig){writeError(w,http.StatusBadRequest,"Telegram Bot Token 或 Username 格式不正确");return}
		h.logger.Error("save platform Telegram Bot settings","error",err);if errors.Is(err,platformsettings.ErrSchemaNotReady){writeError(w,http.StatusServiceUnavailable,"平台 Bot 数据库迁移尚未完成，请更新并重启 API 服务");return};writeError(w,http.StatusInternalServerError,"保存平台 Bot 设置失败，请检查 API 服务日志");return
	}
	if h.telegramAuthBotManager!=nil {h.telegramAuthBotManager.Configure(input.Token,input.Username)}
	writeJSON(w,http.StatusOK,map[string]any{"configured":true,"username":strings.TrimPrefix(strings.TrimSpace(input.Username),"@"),"running":h.telegramAuthBotManager!=nil})
}
func (h *Handler) clearPlatformTelegramBot(w http.ResponseWriter,r *http.Request){
	user,ok:=h.platformAdmin(w,r);if !ok{return}
	if err:=h.platformSettings.Clear(r.Context(),user.ID);err!=nil{h.logger.Error("clear platform Telegram Bot settings","error",err);if errors.Is(err,platformsettings.ErrSchemaNotReady){writeError(w,http.StatusServiceUnavailable,"平台 Bot 数据库迁移尚未完成，请更新并重启 API 服务");return};writeError(w,http.StatusInternalServerError,"停用平台 Bot 失败，请检查 API 服务日志");return}
	if h.telegramAuthBotManager!=nil {h.telegramAuthBotManager.Configure("","")}
	writeJSON(w,http.StatusOK,map[string]any{"configured":false,"username":"","running":false})
}

type addBotRequest struct { Token string `json:"token"` }
type botStatusRequest struct { Status string `json:"status"` }
type adminUserStatusRequest struct { Status string `json:"status"` }

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
	limit:=20
	if raw:=r.URL.Query().Get("limit");raw!=""{
		value,err:=strconv.Atoi(raw)
		if err!=nil || value<1 || value>100{writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return}
		limit=value
	}
	offset:=0
	if raw:=r.URL.Query().Get("offset");raw!=""{
		value,err:=strconv.Atoi(raw)
		if err!=nil || value<0 || value>100000{writeError(w,http.StatusBadRequest,"offset must be between 0 and 100000");return}
		offset=value
	}
	search:=strings.TrimSpace(r.URL.Query().Get("search"))
	if len([]rune(search))>100{writeError(w,http.StatusBadRequest,"search must be at most 100 characters");return}
	result,total,err:=h.bots.ListGroupMembersPageWithSearch(r.Context(),user.TenantID,r.PathValue("groupID"),search,limit,offset)
	if err!=nil{h.logger.Error("list Telegram group members","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":result,"total":total,"limit":limit,"offset":offset,"search":search})
}


type groupRuleRequest struct {
 RuleType string `json:"rule_type"`
 Name string `json:"name"`
 Enabled bool `json:"enabled"`
 Priority int `json:"priority"`
 Config json.RawMessage `json:"config"`
}
func (h *Handler) listGroupRuleEvents(w http.ResponseWriter,r *http.Request){
 if !h.botServicesReady(w){return}
 user,ok:=h.currentUser(w,r);if !ok{return}
 limit:=50
 if raw:=r.URL.Query().Get("limit");raw!=""{
  value,err:=strconv.Atoi(raw)
  if err!=nil || value<1 || value>100{writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return}
  limit=value
 }
 result,err:=h.bots.ListGroupRuleEvents(r.Context(),user.TenantID,r.PathValue("groupID"),limit)
 if err!=nil{h.logger.Error("list Telegram group rule events","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return}
 writeJSON(w,http.StatusOK,map[string]any{"items":result})
}

func (h *Handler) listGroupRules(w http.ResponseWriter,r *http.Request){
 if !h.botServicesReady(w){return}
 user,ok:=h.currentUser(w,r);if !ok{return}
 result,err:=h.bots.ListGroupRules(r.Context(),user.TenantID,r.PathValue("groupID"))
 if err!=nil{h.logger.Error("list Telegram group rules","error",err);writeError(w,http.StatusInternalServerError,"internal server error");return}
 writeJSON(w,http.StatusOK,map[string]any{"items":result})
}
func (h *Handler) createGroupRule(w http.ResponseWriter,r *http.Request){
 if !h.botServicesReady(w){return}
 user,ok:=h.currentUser(w,r);if !ok{return}
 var input groupRuleRequest
 if !decodeJSON(w,r,&input){return}
 result,err:=h.bots.CreateGroupRule(r.Context(),user.TenantID,r.PathValue("groupID"),bots.GroupRuleInput{RuleType:input.RuleType,Name:input.Name,Enabled:input.Enabled,Priority:input.Priority,Config:input.Config})
 if err!=nil{h.writeGroupRuleError(w,err);return}
 writeJSON(w,http.StatusCreated,result)
}
func (h *Handler) updateGroupRule(w http.ResponseWriter,r *http.Request){
 if !h.botServicesReady(w){return}
 user,ok:=h.currentUser(w,r);if !ok{return}
 var input groupRuleRequest
 if !decodeJSON(w,r,&input){return}
 result,err:=h.bots.UpdateGroupRule(r.Context(),user.TenantID,r.PathValue("groupID"),r.PathValue("ruleID"),bots.GroupRuleInput{RuleType:input.RuleType,Name:input.Name,Enabled:input.Enabled,Priority:input.Priority,Config:input.Config})
 if err!=nil{h.writeGroupRuleError(w,err);return}
 writeJSON(w,http.StatusOK,result)
}
func (h *Handler) deleteGroupRule(w http.ResponseWriter,r *http.Request){
 if !h.botServicesReady(w){return}
 user,ok:=h.currentUser(w,r);if !ok{return}
 if err:=h.bots.DeleteGroupRule(r.Context(),user.TenantID,r.PathValue("groupID"),r.PathValue("ruleID"));err!=nil{h.writeGroupRuleError(w,err);return}
 w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) writeGroupRuleError(w http.ResponseWriter,err error){
 switch{
 case errors.Is(err,bots.ErrInvalidGroupRule):writeError(w,http.StatusBadRequest,"invalid rule type, name, priority or configuration")
 case errors.Is(err,bots.ErrGroupRuleConflict):writeError(w,http.StatusConflict,"a rule with this type and name already exists in the group")
 case errors.Is(err,bots.ErrNotFound):writeError(w,http.StatusNotFound,"group or rule not found")
 default:h.logger.Error("Telegram group rule operation failed","error",err);writeError(w,http.StatusInternalServerError,"internal server error")
 }
}


type groupMemberListRequest struct { TelegramUserID int64 `json:"telegram_user_id"`; ListType string `json:"list_type"`; Reason string `json:"reason"` }
type groupMemberWarningRequest struct { Reason string `json:"reason"`; Source string `json:"source"` }

func (h *Handler) listGroupMemberLists(w http.ResponseWriter,r *http.Request) {
	if !h.botServicesReady(w){return}; user,ok:=h.currentUser(w,r);if !ok{return}
	items,err:=h.bots.ListGroupMemberLists(r.Context(),user.TenantID,r.PathValue("groupID"))
	if err!=nil{h.logger.Error("list group member lists","error",err);writeError(w,http.StatusInternalServerError,"读取群组名单失败");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":items})
}
func (h *Handler) upsertGroupMemberList(w http.ResponseWriter,r *http.Request) {
	if !h.botServicesReady(w){return}; user,ok:=h.currentUser(w,r);if !ok{return}
	var input groupMemberListRequest;if !decodeJSON(w,r,&input){return}
	if err:=h.bots.UpsertGroupMemberList(r.Context(),user.TenantID,r.PathValue("groupID"),input.TelegramUserID,0,input.ListType,input.Reason);err!=nil{h.writeGroupRuleError(w,err);return}
	writeJSON(w,http.StatusOK,map[string]any{"ok":true})
}
func (h *Handler) deleteGroupMemberList(w http.ResponseWriter,r *http.Request) {
	if !h.botServicesReady(w){return}; user,ok:=h.currentUser(w,r);if !ok{return}
	userID,err:=strconv.ParseInt(r.PathValue("userID"),10,64);if err!=nil||userID<=0{writeError(w,http.StatusBadRequest,"invalid Telegram user ID");return}
	listType:=r.URL.Query().Get("list_type")
	if err:=h.bots.DeleteGroupMemberList(r.Context(),user.TenantID,r.PathValue("groupID"),userID,listType);err!=nil{h.writeGroupRuleError(w,err);return}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) listGroupMemberWarnings(w http.ResponseWriter,r *http.Request) {
	if !h.botServicesReady(w){return}; user,ok:=h.currentUser(w,r);if !ok{return}
	userID,err:=strconv.ParseInt(r.PathValue("userID"),10,64);if err!=nil||userID<=0{writeError(w,http.StatusBadRequest,"invalid Telegram user ID");return}
	limit:=20
	if raw:=r.URL.Query().Get("limit");raw!=""{value,parseErr:=strconv.Atoi(raw);if parseErr!=nil||value<1||value>100{writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return};limit=value}
	offset:=0
	if raw:=r.URL.Query().Get("offset");raw!=""{value,parseErr:=strconv.Atoi(raw);if parseErr!=nil||value<0||value>100000{writeError(w,http.StatusBadRequest,"offset must be between 0 and 100000");return};offset=value}
	items,err:=h.bots.ListGroupMemberWarningsPage(r.Context(),user.TenantID,r.PathValue("groupID"),userID,limit,offset)
	if err!=nil{h.writeGroupRuleError(w,err);return}
	total,err:=h.bots.CountGroupMemberWarnings(r.Context(),user.TenantID,r.PathValue("groupID"),userID)
	if err!=nil{h.writeGroupRuleError(w,err);return}
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"total":total,"limit":limit,"offset":offset})
}
func (h *Handler) clearGroupMemberWarnings(w http.ResponseWriter,r *http.Request) {
	if !h.botServicesReady(w){return}; user,ok:=h.currentUser(w,r);if !ok{return}
	userID,err:=strconv.ParseInt(r.PathValue("userID"),10,64);if err!=nil||userID<=0{writeError(w,http.StatusBadRequest,"invalid Telegram user ID");return}
	count,err:=h.bots.ClearGroupMemberWarnings(r.Context(),user.TenantID,r.PathValue("groupID"),userID)
	if err!=nil{h.writeGroupRuleError(w,err);return}
	writeJSON(w,http.StatusOK,map[string]any{"ok":true,"deleted":count})
}

func (h *Handler) addGroupMemberWarning(w http.ResponseWriter,r *http.Request) {
	if !h.botServicesReady(w){return}; user,ok:=h.currentUser(w,r);if !ok{return}
	userID,err:=strconv.ParseInt(r.PathValue("userID"),10,64);if err!=nil||userID<=0{writeError(w,http.StatusBadRequest,"invalid Telegram user ID");return}
	var input groupMemberWarningRequest;if !decodeJSON(w,r,&input){return}
	item,err:=h.bots.AddGroupMemberWarning(r.Context(),user.TenantID,r.PathValue("groupID"),userID,0,input.Reason,input.Source)
	if err!=nil{h.writeGroupRuleError(w,err);return}
	if input.Source=="manual" {
		if err:=h.bots.ApplyWarningEscalation(r.Context(),user.TenantID,r.PathValue("groupID"),userID);err!=nil {
			h.logger.Warn("warning recorded but automatic escalation did not complete","group_id",r.PathValue("groupID"),"telegram_user_id",userID,"error",err)
		}
	}
	writeJSON(w,http.StatusCreated,item)
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
type bulkMemberActionRequest struct {
	UserIDs []int64 `json:"user_ids"`
	Action string `json:"action"`
	DurationMinutes int `json:"duration_minutes"`
}

func validateBulkMemberAction(input bulkMemberActionRequest) error {
	if len(input.UserIDs)<1 || len(input.UserIDs)>50 { return errors.New("user_ids must contain between 1 and 50 members") }
	seen:=make(map[int64]struct{},len(input.UserIDs))
	for _,id:=range input.UserIDs {
		if id<=0 { return errors.New("every Telegram user ID must be positive") }
		if _,exists:=seen[id]; exists { return errors.New("user_ids must not contain duplicates") }
		seen[id]=struct{}{}
	}
	if input.Action!="ban" && input.Action!="unban" && input.Action!="mute" && input.Action!="unmute" { return errors.New("unsupported member action") }
	if input.Action=="mute" && (input.DurationMinutes<1 || input.DurationMinutes>525600) { return errors.New("duration_minutes must be between 1 and 525600") }
	if input.Action!="mute" && input.DurationMinutes!=0 { return errors.New("duration_minutes is only valid for mute") }
	return nil
}

type bulkMemberActionResult struct {
	TelegramUserID int64 `json:"telegram_user_id"`
	OK bool `json:"ok"`
	Error string `json:"error,omitempty"`
}

func executeBulkMemberAction(userIDs []int64, apply func(int64) error) ([]bulkMemberActionResult, int) {
	results:=make([]bulkMemberActionResult,0,len(userIDs))
	succeeded:=0
	for _,id:=range userIDs {
		result:=bulkMemberActionResult{TelegramUserID:id}
		if err:=apply(id); err!=nil {
			result.Error=err.Error()
			if len([]rune(result.Error))>500 { result.Error=string([]rune(result.Error)[:500]) }
		} else { result.OK=true; succeeded++ }
		results=append(results,result)
	}
	return results,succeeded
}

func (h *Handler) bulkModerateGroupMembers(w http.ResponseWriter,r *http.Request) {
	if !h.botServicesReady(w) { return }
	user,ok:=h.currentUser(w,r); if !ok { return }
	var input bulkMemberActionRequest
	if !decodeJSON(w,r,&input) { return }
	if err:=validateBulkMemberAction(input); err!=nil { writeError(w,http.StatusBadRequest,err.Error()); return }
	results,succeeded:=executeBulkMemberAction(input.UserIDs,func(id int64) error {
		return h.bots.ModerateGroupMember(r.Context(),user.TenantID,user.ID,r.PathValue("groupID"),id,input.Action,input.DurationMinutes)
	})
	writeJSON(w,http.StatusOK,map[string]any{"action":input.Action,"total":len(results),"succeeded":succeeded,"failed":len(results)-succeeded,"items":results})
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

func (h *Handler) adminUsers(w http.ResponseWriter,r *http.Request) {
	if h.auth == nil { writeError(w,http.StatusServiceUnavailable,"authentication service unavailable"); return }
	user,ok:=h.currentUser(w,r);if !ok{return}
	if user.Role!="admin" { writeError(w,http.StatusForbidden,"platform administrator permission required"); return }
	limit,offset:=25,0
	if raw:=r.URL.Query().Get("limit");raw!="" { value,err:=strconv.Atoi(raw);if err!=nil||value<1||value>100{writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return};limit=value }
	if raw:=r.URL.Query().Get("offset");raw!="" { value,err:=strconv.Atoi(raw);if err!=nil||value<0{writeError(w,http.StatusBadRequest,"offset must be a non-negative integer");return};offset=value }
	items,total,err:=h.auth.ListAdminUsers(r.Context(),r.URL.Query().Get("search"),limit,offset)
	if err!=nil { h.logger.Error("list platform users","error",err);writeError(w,http.StatusInternalServerError,"读取用户列表失败");return }
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"total":total,"limit":limit,"offset":offset})
}

func (h *Handler) updateAdminUserStatus(w http.ResponseWriter,r *http.Request) {
	if h.auth == nil { writeError(w,http.StatusServiceUnavailable,"authentication service unavailable"); return }
	user,ok:=h.currentUser(w,r);if !ok{return}
	if user.Role!="admin" { writeError(w,http.StatusForbidden,"platform administrator permission required"); return }
	var input adminUserStatusRequest
	if !decodeJSON(w,r,&input){return}
	err:=h.auth.UpdateAdminUserStatus(r.Context(),user.ID,r.PathValue("userID"),input.Status)
	switch {
	case err==nil: writeJSON(w,http.StatusOK,map[string]any{"ok":true,"status":input.Status})
	case errors.Is(err,auth.ErrInvalidAdminUserStatus): writeError(w,http.StatusBadRequest,"管理员账号不可锁定，状态只能设置为 active 或 locked")
	case errors.Is(err,auth.ErrAdminSelfLock): writeError(w,http.StatusBadRequest,"不能锁定当前登录的管理员账号")
	case errors.Is(err,auth.ErrAdminUserNotFound): writeError(w,http.StatusNotFound,"用户不存在")
	default: h.logger.Error("update platform user status","error",err);writeError(w,http.StatusInternalServerError,"更新用户状态失败")
	}
}


type adminPaymentChannelRequest struct {
	Name string `json:"name"`
	BaseURL string `json:"base_url"`
	MerchantPID string `json:"merchant_pid"`
	SecretKey string `json:"secret_key"`
	Currency string `json:"currency"`
	Token string `json:"token"`
	Network string `json:"network"`
	Enabled bool `json:"enabled"`
}

func (h *Handler) paymentAdmin(w http.ResponseWriter,r *http.Request)(auth.User,bool){
	if h.auth==nil{writeError(w,http.StatusServiceUnavailable,"authentication service unavailable");return auth.User{},false}
	user,ok:=h.currentUser(w,r);if !ok{return auth.User{},false}
	if user.Role!="admin"{writeError(w,http.StatusForbidden,"platform administrator permission required");return auth.User{},false}
	if h.paymentChannels==nil{writeError(w,http.StatusServiceUnavailable,"payment credentials encryption is not configured");return auth.User{},false}
	return user,true
}

func (h *Handler) adminPaymentChannels(w http.ResponseWriter,r *http.Request){
	if _,ok:=h.paymentAdmin(w,r);!ok{return}
	items,err:=h.paymentChannels.List(r.Context())
	if err!=nil{h.logger.Error("list payment channels","error",err);writeError(w,http.StatusInternalServerError,"读取支付渠道配置失败");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":items})
}

func (h *Handler) updateAdminPaymentChannel(w http.ResponseWriter,r *http.Request){
	user,ok:=h.paymentAdmin(w,r);if !ok{return}
	var input adminPaymentChannelRequest
	if !decodeJSON(w,r,&input){return}
	channel:=payments.ChannelConfig{Code:r.PathValue("channelCode"),Name:input.Name,BaseURL:input.BaseURL,MerchantPID:input.MerchantPID,SecretKey:input.SecretKey,Currency:input.Currency,Token:input.Token,Network:input.Network,Enabled:input.Enabled}
	if err:=h.paymentChannels.Save(r.Context(),channel,user.ID);err!=nil{
		if errors.Is(err,payments.ErrInvalidChannelConfig){writeError(w,http.StatusBadRequest,"支付渠道配置无效：请检查 HTTPS 网关地址、商户凭证、币种和网络");return}
		h.logger.Error("save payment channel","error",err);writeError(w,http.StatusInternalServerError,"保存支付渠道配置失败");return
	}
	items,err:=h.paymentChannels.List(r.Context());if err!=nil{h.logger.Error("reload payment channels","error",err);writeError(w,http.StatusInternalServerError,"配置已保存，但读取状态失败");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":items})
}

type walletRechargeRequest struct { AmountUSDT int64 `json:"amount_usdt"` }

func (h *Handler) createWalletRecharge(w http.ResponseWriter,r *http.Request){
	if !h.requireAuthStore(w){return}
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r));if err!=nil{writeError(w,http.StatusUnauthorized,"unauthorized");return}
	if h.recharge==nil||h.paymentPublicBaseURL==""{writeError(w,http.StatusServiceUnavailable,"钱包在线充值暂未配置");return}
	var input walletRechargeRequest
	if !decodeJSON(w,r,&input){return}
	if input.AmountUSDT<=0||input.AmountUSDT>1000000000{writeError(w,http.StatusBadRequest,"充值金额必须为 1 至 1000000000 整数 USDT");return}
	ctx,cancel:=context.WithTimeout(r.Context(),20*time.Second);defer cancel()
	notifyURL:=h.paymentPublicBaseURL+"/api/v1/payments/epusdt/gmpay/notify"
	redirectURL:=h.paymentPublicBaseURL+"/dashboard/account/agent"
	order,err:=h.recharge.CreateGMPayOrder(ctx,user.ID,input.AmountUSDT,notifyURL,redirectURL)
	if err!=nil{h.logger.Error("create wallet recharge order","error",err);if errors.Is(err,payments.ErrProviderRejected){writeError(w,http.StatusBadGateway,"支付网关未能创建订单，请稍后重试");return};writeError(w,http.StatusBadGateway,"创建充值订单失败，请检查支付渠道配置");return}
	writeJSON(w,http.StatusCreated,order)
}

func (h *Handler) listWalletRecharges(w http.ResponseWriter,r *http.Request){
	if !h.requireAuthStore(w){return}
	user,err:=h.auth.Authenticate(r.Context(),bearerToken(r));if err!=nil{writeError(w,http.StatusUnauthorized,"unauthorized");return}
	if h.recharge==nil{writeError(w,http.StatusServiceUnavailable,"钱包在线充值暂未配置");return}
	limit,offset:=20,0
	if raw:=r.URL.Query().Get("limit");raw!=""{value,e:=strconv.Atoi(raw);if e!=nil||value<1||value>100{writeError(w,http.StatusBadRequest,"limit must be between 1 and 100");return};limit=value}
	if raw:=r.URL.Query().Get("offset");raw!=""{value,e:=strconv.Atoi(raw);if e!=nil||value<0||value>1000000{writeError(w,http.StatusBadRequest,"offset must be between 0 and 1000000");return};offset=value}
	items,total,err:=h.recharge.ListUserRechargeOrders(r.Context(),user.ID,limit,offset)
	if err!=nil{h.logger.Error("list wallet recharge orders","error",err);writeError(w,http.StatusInternalServerError,"读取充值订单失败");return}
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"total":total,"limit":limit,"offset":offset})
}

func (h *Handler) gmpayNotify(w http.ResponseWriter,r *http.Request){
	if h.recharge==nil{http.Error(w,"payment service unavailable",http.StatusServiceUnavailable);return}
	r.Body=http.MaxBytesReader(w,r.Body,64<<10)
	raw,err:=io.ReadAll(r.Body);if err!=nil{http.Error(w,"invalid callback body",http.StatusBadRequest);return}
	var rawParams map[string]any
	decoder:=json.NewDecoder(strings.NewReader(string(raw)));decoder.UseNumber()
	if err:=decoder.Decode(&rawParams);err!=nil{http.Error(w,"invalid callback JSON",http.StatusBadRequest);return}
	params:=make(map[string]string,len(rawParams))
	for key,value:=range rawParams{
		switch typed:=value.(type){
		case nil:
			continue
		case string:
			params[key]=typed
		case json.Number:
			normalized,normalizeErr:=payments.NormalizeJSONNumber(typed.String())
			if normalizeErr!=nil{http.Error(w,"invalid callback number",http.StatusBadRequest);return}
			params[key]=normalized
		default:
			http.Error(w,"invalid callback field",http.StatusBadRequest);return
		}
	}
	hash:=sha256.Sum256(raw)
	ctx,cancel:=context.WithTimeout(r.Context(),10*time.Second);defer cancel()
	if err:=h.recharge.ApplyGMPayCallback(ctx,params,hash[:]);err!=nil{
		if errors.Is(err,payments.ErrRechargeNeedsReview){
			h.logger.Warn("GMPay payment received but fractional USDT amount requires manual reconciliation","order_id",params["order_id"],"trade_id",params["trade_id"])
			w.Header().Set("Content-Type","text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_,_=w.Write([]byte("ok"))
			return
		}
		if errors.Is(err,payments.ErrRechargeOrderNotFound){http.Error(w,"order not found",http.StatusNotFound);return}
		if errors.Is(err,payments.ErrInvalidGMPayCallback)||errors.Is(err,payments.ErrRechargeOrderConflict){http.Error(w,"invalid payment callback",http.StatusBadRequest);return}
		h.logger.Error("process GMPay callback","error",err);http.Error(w,"payment callback processing failed",http.StatusInternalServerError);return
	}
	w.Header().Set("Content-Type","text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_,_=w.Write([]byte("ok"))
}

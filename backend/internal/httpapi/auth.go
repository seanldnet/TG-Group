package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionLifetime = 12 * time.Hour
	loginLimit      = 5
	loginWindow     = 15 * time.Minute
)

type AuthHandler struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type passwordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type owner struct {
	ID           int64
	Username     string
	PasswordHash string
}

type sessionContextKey struct{}

type authenticatedSession struct {
	UserID   int64
	Username string
}

func (h AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/login", h.login)
	mux.Handle("POST /auth/logout", h.requireSession(http.HandlerFunc(h.logout)))
	mux.Handle("GET /auth/codes", h.requireSession(http.HandlerFunc(h.accessCodes)))
	mux.Handle("GET /user/info", h.requireSession(http.HandlerFunc(h.userInfo)))
	mux.Handle("PUT /auth/password", h.requireSession(http.HandlerFunc(h.changePassword)))
}

func (h AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || input.Password == "" || len(input.Password) > 72 {
		apiError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	ctx := r.Context()
	ip := remoteIP(r)
	allowed, err := h.allowLogin(ctx, ip)
	if err != nil {
		apiError(w, http.StatusServiceUnavailable, "authentication temporarily unavailable")
		return
	}
	if !allowed {
		apiError(w, http.StatusTooManyRequests, "too many login attempts; try again later")
		return
	}

	var account owner
	err = h.DB.QueryRow(ctx, `
		SELECT id, username, password_hash
		FROM platform_users
		WHERE username = $1 AND role = 'owner'
	`, input.Username).Scan(&account.ID, &account.Username, &account.PasswordHash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(input.Password)) != nil {
		apiError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		apiError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	token := hex.EncodeToString(rawToken)
	tokenHash := hashToken(token)
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	if _, err := h.DB.Exec(ctx, `
		INSERT INTO auth_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, account.ID, tokenHash, expiresAt); err != nil {
		apiError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	h.clearLoginFailures(ctx, ip)
	apiOK(w, http.StatusOK, map[string]any{
		"accessToken": token,
		"expiresAt":   expiresAt,
	})
}

func (h AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	session := sessionFromRequest(r)
	if session == nil {
		apiError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	_, err := h.DB.Exec(r.Context(), "DELETE FROM auth_sessions WHERE token_hash = $1", hashToken(bearerToken(r)))
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not end session")
		return
	}
	apiOK(w, http.StatusOK, map[string]bool{"success": true})
}

func (h AuthHandler) accessCodes(w http.ResponseWriter, _ *http.Request) {
	apiOK(w, http.StatusOK, []string{"OWNER"})
}

func (h AuthHandler) userInfo(w http.ResponseWriter, r *http.Request) {
	session := sessionFromRequest(r)
	apiOK(w, http.StatusOK, map[string]any{
		"userId":   session.UserID,
		"username": session.Username,
		"realName": session.Username,
		"roles":    []string{"OWNER"},
		"homePath": "/tg-saas/dashboard",
	})
}

func (h AuthHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	session := sessionFromRequest(r)
	var input passwordRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if len(input.NewPassword) < 8 || len(input.NewPassword) > 72 {
		apiError(w, http.StatusBadRequest, "new password must contain 8 to 72 bytes")
		return
	}
	var currentHash string
	if err := h.DB.QueryRow(r.Context(), "SELECT password_hash FROM platform_users WHERE id = $1", session.UserID).Scan(&currentHash); err != nil {
		apiError(w, http.StatusInternalServerError, "could not verify current password")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(input.CurrentPassword)) != nil {
		apiError(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), "UPDATE platform_users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(newHash), session.UserID); err == nil {
		_, err = tx.Exec(r.Context(), "DELETE FROM auth_sessions WHERE user_id = $1", session.UserID)
	}
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		apiError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	apiOK(w, http.StatusOK, map[string]bool{"success": true})
}

func (h AuthHandler) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			apiError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		var session authenticatedSession
		err := h.DB.QueryRow(r.Context(), `
			SELECT u.id, u.username
			FROM auth_sessions s
			JOIN platform_users u ON u.id = s.user_id
			WHERE s.token_hash = $1 AND s.expires_at > NOW()
		`, hashToken(token)).Scan(&session.UserID, &session.Username)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apiError(w, http.StatusUnauthorized, "session expired")
			} else {
				apiError(w, http.StatusInternalServerError, "could not validate session")
			}
			return
		}
		ctx := r.Context()
		ctx = withSession(ctx, &session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sessionFromRequest(r *http.Request) *authenticatedSession {
	session, _ := r.Context().Value(sessionContextKey{}).(*authenticatedSession)
	return session
}

func withSession(parent context.Context, session *authenticatedSession) context.Context {
	return context.WithValue(parent, sessionContextKey{}, session)
}

func (h AuthHandler) allowLogin(ctx context.Context, ip string) (bool, error) {
	key := "auth:login-attempts:" + hashToken(ip)
	count, err := h.Redis.Incr(ctx, key).Result()
	if err != nil { return false, err }
	if count == 1 {
		if err := h.Redis.Expire(ctx, key, loginWindow).Err(); err != nil { return false, err }
	}
	return count <= loginLimit, nil
}

func (h AuthHandler) clearLoginFailures(ctx context.Context, ip string) {
	_ = h.Redis.Del(ctx, "auth:login-attempts:"+hashToken(ip)).Err()
}
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil { return host }
	return r.RemoteAddr
}

func bearerToken(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) < 8 || !strings.EqualFold(value[:7], "Bearer ") { return "" }
	return strings.TrimSpace(value[7:])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		apiError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func apiOK(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, map[string]any{"code": 0, "data": data})
}

func apiError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"code": status, "message": message})
}


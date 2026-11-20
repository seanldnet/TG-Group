package httpapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CapVerifier struct {
	DB *pgxpool.Pool
	EncryptionKey string
	Client *http.Client
}

type capSettings struct {
	Enabled bool   `json:"enabled"`
	Mode string `json:"mode"`
	Domain string `json:"domain"`
	SiteKey string `json:"siteKey"`
	SecretEncrypted string `json:"secretEncrypted"`
	SecretDomain string `json:"secretDomain"`
	SecretSiteKey string `json:"secretSiteKey"`
}

type capPublicSettings struct {
	Enabled bool `json:"enabled"`
	Mode string `json:"mode"`
	Domain string `json:"domain"`
	SiteKey string `json:"siteKey"`
	SecretConfigured bool `json:"secretConfigured"`
}

type capSettingsRequest struct {
	Enabled bool `json:"enabled"`
	Mode string `json:"mode"`
	Domain string `json:"domain"`
	SiteKey string `json:"siteKey"`
	Secret string `json:"secret"`
}

type capVerifyRequest struct {
	Secret string `json:"secret"`
	Response string `json:"response"`
}

type capVerifyResponse struct {
	Success bool `json:"success"`
}

func (h CapVerifier) Register(mux *http.ServeMux, auth AuthHandler) {
	mux.HandleFunc("GET /auth/captcha-config", h.publicConfig)
	mux.Handle("GET /settings/captcha", auth.requireSession(http.HandlerFunc(h.getSettings)))
	mux.Handle("PUT /settings/captcha", auth.requireSession(http.HandlerFunc(h.updateSettings)))
}

func (h CapVerifier) load(ctx context.Context) (capSettings, error) {
	var raw []byte
	err := h.DB.QueryRow(ctx, "SELECT value FROM platform_settings WHERE key = 'cap_verification'").Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return capSettings{Mode:"checkbox"}, nil }
		return capSettings{}, err
	}
	var settings capSettings
	if err := json.Unmarshal(raw, &settings); err != nil { return capSettings{}, fmt.Errorf("decode Cap settings: %w", err) }
	if settings.Mode == "" { settings.Mode = "checkbox" }
	return settings, nil
}

func (h CapVerifier) publicConfig(w http.ResponseWriter, r *http.Request) {
	settings, err := h.load(r.Context())
	if err != nil { apiError(w, http.StatusInternalServerError, "could not load captcha settings"); return }
	apiOK(w, http.StatusOK, capPublicSettings{
		Enabled: settings.Enabled, Mode: settings.Mode, Domain: settings.Domain,
		SiteKey: settings.SiteKey, SecretConfigured: settings.SecretEncrypted != "",
	})
}

func (h CapVerifier) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.load(r.Context())
	if err != nil { apiError(w, http.StatusInternalServerError, "could not load captcha settings"); return }
	apiOK(w, http.StatusOK, capPublicSettings{
		Enabled: settings.Enabled, Mode: settings.Mode, Domain: settings.Domain,
		SiteKey: settings.SiteKey, SecretConfigured: settings.SecretEncrypted != "",
	})
}

func (h CapVerifier) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input capSettingsRequest
	if !decodeJSON(w, r, &input) { return }
	input.Domain = strings.TrimSpace(input.Domain)
	input.SiteKey = strings.TrimSpace(input.SiteKey)
	input.Secret = strings.TrimSpace(input.Secret)
	if input.Mode != "checkbox" && input.Mode != "floating" && input.Mode != "invisible" {
		apiError(w, http.StatusBadRequest, "invalid Cap display mode"); return
	}
	current, err := h.load(r.Context())
	if err != nil { apiError(w, http.StatusInternalServerError, "could not load captcha settings"); return }
	if input.Secret != "" {
		current.SecretEncrypted, err = h.encryptSecret(input.Secret)
		if err != nil { apiError(w, http.StatusServiceUnavailable, "Cap 密钥加密配置不可用"); return }
		current.SecretDomain = input.Domain
		current.SecretSiteKey = input.SiteKey
	}
	if input.Enabled {
		if _, err := capSiteVerifyURL(input.Domain, input.SiteKey); err != nil {
			apiError(w, http.StatusBadRequest, "Cap 域名或 Site Key 格式不正确"); return
		}
		if current.SecretEncrypted == "" {
			apiError(w, http.StatusBadRequest, "启用 Cap 前必须填写 Secret"); return
		}
		if current.SecretDomain != input.Domain || current.SecretSiteKey != input.SiteKey {
			apiError(w, http.StatusBadRequest, "当前 Cap 域名或 Site Key 对应的 Secret 未配置，请重新填写 Secret"); return
		}
	}
	current.Enabled = input.Enabled
	current.Mode = input.Mode
	current.Domain = input.Domain
	current.SiteKey = input.SiteKey
	raw, err := json.Marshal(current)
	if err != nil { apiError(w, http.StatusInternalServerError, "could not save captcha settings"); return }
	_, err = h.DB.Exec(r.Context(), `
		INSERT INTO platform_settings(key, value, updated_at)
		VALUES ('cap_verification', $1::jsonb, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, raw)
	if err != nil { apiError(w, http.StatusInternalServerError, "could not save captcha settings"); return }
	apiOK(w, http.StatusOK, capPublicSettings{
		Enabled: current.Enabled, Mode: current.Mode, Domain: current.Domain,
		SiteKey: current.SiteKey, SecretConfigured: current.SecretEncrypted != "",
	})
}

func (h CapVerifier) Verify(ctx context.Context, token string) error {
	settings, err := h.load(ctx)
	if err != nil { return fmt.Errorf("load Cap settings: %w", err) }
	if !settings.Enabled { return nil }
	if strings.TrimSpace(token) == "" { return errors.New("CAPTCHA token is required") }
	secret, err := h.decryptSecret(settings.SecretEncrypted)
	if err != nil { return fmt.Errorf("decrypt Cap secret: %w", err) }
	endpoint, err := capSiteVerifyURL(settings.Domain, settings.SiteKey)
	if err != nil { return err }
	return verifyCapToken(ctx, h.Client, endpoint, secret, token)
}

func verifyCapToken(ctx context.Context, client *http.Client, endpoint, secret, token string) error {
	payload, _ := json.Marshal(capVerifyRequest{Secret: secret, Response: token})
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/json")
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	resp, err := client.Do(req)
	if err != nil { return fmt.Errorf("Cap verification request failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return fmt.Errorf("Cap verification returned HTTP %d", resp.StatusCode) }
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil { return err }
	var result capVerifyResponse
	if err := json.Unmarshal(body, &result); err != nil { return fmt.Errorf("invalid Cap verification response: %w", err) }
	if !result.Success { return errors.New("Cap verification rejected the token") }
	return nil
}

func capSiteVerifyURL(domain, siteKey string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(domain))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid Cap domain")
	}
	if parsed.Scheme != "https" && parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" {
		return "", errors.New("Cap domain must use HTTPS")
	}
	key := strings.TrimSpace(siteKey)
	if key == "" || strings.Contains(key, "/") || key == "." || key == ".." {
		return "", errors.New("invalid Cap site key")
	}
	base := strings.TrimRight(parsed.String(), "/")
	return base + "/" + url.PathEscape(key) + "/siteverify", nil
}

func (h CapVerifier) encryptionKey() ([]byte, error) {
	key, err := hex.DecodeString(strings.TrimSpace(h.EncryptionKey))
	if err != nil || len(key) != 32 { return nil, errors.New("CAP_SECRET_ENCRYPTION_KEY or BOT_TOKEN_ENCRYPTION_KEY must be 64 hex characters") }
	return key, nil
}

func (h CapVerifier) encryptSecret(secret string) (string, error) {
	key, err := h.encryptionKey(); if err != nil { return "", err }
	block, err := aes.NewCipher(key); if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block); if err != nil { return "", err }
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil { return "", err }
	sealed := gcm.Seal(nonce, nonce, []byte(secret), []byte("tg-group/cap-secret/v1"))
	return "v1:" + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (h CapVerifier) decryptSecret(value string) (string, error) {
	if !strings.HasPrefix(value, "v1:") { return "", errors.New("unsupported encrypted Cap secret format") }
	key, err := h.encryptionKey(); if err != nil { return "", err }
	sealed, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, "v1:")); if err != nil { return "", err }
	block, err := aes.NewCipher(key); if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block); if err != nil { return "", err }
	if len(sealed) < gcm.NonceSize() { return "", errors.New("encrypted Cap secret is truncated") }
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, []byte("tg-group/cap-secret/v1"))
	if err != nil { return "", err }
	return string(plain), nil
}

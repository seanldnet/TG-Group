package httpapi

import (
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
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidTelegramToken = errors.New("invalid Telegram Bot Token")

type BotHandler struct {
	DB                  *pgxpool.Pool
	EncryptionKey       string
	TelegramAPIBaseURL  string
	HTTPClient          *http.Client
}

type createBotRequest struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

type renameBotRequest struct {
	Name string `json:"name"`
}

type telegramGetMeResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"first_name"`
	} `json:"result"`
}

func (h BotHandler) Register(mux *http.ServeMux, auth AuthHandler) {
	mux.Handle("GET /bots", auth.requireSession(http.HandlerFunc(h.list)))
	mux.Handle("POST /bots", auth.requireSession(http.HandlerFunc(h.create)))
	mux.Handle("PATCH /bots/{id}", auth.requireSession(http.HandlerFunc(h.rename)))
	mux.Handle("DELETE /bots/{id}", auth.requireSession(http.HandlerFunc(h.delete)))
}

func (h BotHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT b.id, b.name, COALESCE(b.username, ''), b.status, b.created_at, COUNT(g.id)
		FROM telegram_bots b
		LEFT JOIN telegram_groups g ON g.bot_id = b.id
		GROUP BY b.id
		ORDER BY b.created_at DESC
	`)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load bots")
		return
	}
	defer rows.Close()
	bots := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var name, username, status string
		var createdAt time.Time
		var groupCount int64
		if err := rows.Scan(&id, &name, &username, &status, &createdAt, &groupCount); err != nil {
			apiError(w, http.StatusInternalServerError, "could not load bots")
			return
		}
		bots = append(bots, map[string]any{
			"id": id, "name": name, "username": username, "status": status,
			"groupCount": groupCount, "createdAt": createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		apiError(w, http.StatusInternalServerError, "could not load bots")
		return
	}
	apiOK(w, http.StatusOK, bots)
}

func (h BotHandler) create(w http.ResponseWriter, r *http.Request) {
	var input createBotRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Token = strings.TrimSpace(input.Token)
	if input.Name == "" || len(input.Name) > 120 || len(input.Token) < 20 || len(input.Token) > 256 || strings.ContainsAny(input.Token, "/?#%\\") {
		apiError(w, http.StatusBadRequest, "a bot name and valid Telegram token are required")
		return
	}
	key, err := decodeEncryptionKey(h.EncryptionKey)
	if err != nil {
		apiError(w, http.StatusServiceUnavailable, "BOT_TOKEN_ENCRYPTION_KEY must be 64 hexadecimal characters")
		return
	}
	username, err := h.verifyTelegramToken(r.Context(), input.Token)
	if err != nil {
		if errors.Is(err, ErrInvalidTelegramToken) {
			apiError(w, http.StatusBadRequest, "Telegram rejected this Bot Token")
		} else {
			apiError(w, http.StatusBadGateway, "could not verify Bot Token with Telegram")
		}
		return
	}
	encrypted, err := encryptSecret(key, input.Token)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not protect Bot Token")
		return
	}
	var id int64
	var createdAt time.Time
	err = h.DB.QueryRow(r.Context(), `
		INSERT INTO telegram_bots (name, username, token_encrypted, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, created_at
	`, input.Name, username, encrypted).Scan(&id, &createdAt)
	if err != nil {
		apiError(w, http.StatusConflict, "this Telegram Bot may already be registered")
		return
	}
	apiOK(w, http.StatusCreated, map[string]any{
		"id": id, "name": input.Name, "username": username,
		"status": "active", "groupCount": 0,
		"tokenMasked": maskToken(input.Token), "createdAt": createdAt,
	})
}

func (h BotHandler) rename(w http.ResponseWriter, r *http.Request) {
	id, ok := botID(w, r)
	if !ok { return }
	var input renameBotRequest
	if !decodeJSON(w, r, &input) { return }
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 120 {
		apiError(w, http.StatusBadRequest, "bot name must contain 1 to 120 characters")
		return
	}
	tag, err := h.DB.Exec(r.Context(), "UPDATE telegram_bots SET name = $1, updated_at = NOW() WHERE id = $2", input.Name, id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not update bot")
		return
	}
	if tag.RowsAffected() == 0 {
		apiError(w, http.StatusNotFound, "bot not found")
		return
	}
	apiOK(w, http.StatusOK, map[string]any{"id": id, "name": input.Name})
}

func (h BotHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := botID(w, r)
	if !ok { return }
	tag, err := h.DB.Exec(r.Context(), "DELETE FROM telegram_bots WHERE id = $1", id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not delete bot")
		return
	}
	if tag.RowsAffected() == 0 {
		apiError(w, http.StatusNotFound, "bot not found")
		return
	}
	apiOK(w, http.StatusOK, map[string]bool{"success": true})
}

func (h BotHandler) verifyTelegramToken(ctx context.Context, token string) (string, error) {
	base := strings.TrimRight(h.TelegramAPIBaseURL, "/")
	if base == "" { base = "https://api.telegram.org" }
	client := h.HTTPClient
	if client == nil { client = &http.Client{Timeout: 8 * time.Second} }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/bot"+token+"/getMe", nil)
	if err != nil { return "", err }
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil { return "", err }
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidTelegramToken
	}
	if resp.StatusCode != http.StatusOK { return "", fmt.Errorf("Telegram returned HTTP %d", resp.StatusCode) }
	var result telegramGetMeResponse
	if err := json.Unmarshal(body, &result); err != nil { return "", err }
	if !result.OK {
		return "", ErrInvalidTelegramToken
	}
	if result.Result.ID == 0 || result.Result.Username == "" {
		return "", errors.New("Telegram returned an incomplete Bot profile")
	}
	return result.Result.Username, nil
}

func botID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, "invalid bot id")
		return 0, false
	}
	return id, true
}

func decodeEncryptionKey(value string) ([]byte, error) {
	key, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(key) != 32 { return nil, errors.New("encryption key must be 32 bytes") }
	return key, nil
}

func encryptSecret(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil { return "", err }
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "v1:"+base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decryptSecret(key []byte, ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, "v1:") { return "", errors.New("unsupported encrypted token version") }
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, "v1:"))
	if err != nil { return "", err }
	block, err := aes.NewCipher(key)
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	if len(raw) < gcm.NonceSize() { return "", errors.New("encrypted token is truncated") }
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil { return "", err }
	return string(plain), nil
}

func maskToken(token string) string {
	if len(token) <= 4 { return "••••" }
	return "••••" + token[len(token)-4:]
}


package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BotRuntime struct {
	DB                 *pgxpool.Pool
	EncryptionKey      string
	TelegramAPIBaseURL string
	HTTPClient         *http.Client
	Logger             *slog.Logger
	SyncInterval       time.Duration

	runners map[int64]botRunner
}

type botRunner struct {
	cancel context.CancelFunc
	tokenHash string
}

type telegramUpdate struct {
	UpdateID      int64                      `json:"update_id"`
	Message       *telegramMessage           `json:"message"`
	MyChatMember  *telegramChatMemberUpdated `json:"my_chat_member"`
	ChatMember    *telegramChatMemberUpdated `json:"chat_member"`
}

type telegramMessage struct {
	MessageID     int64          `json:"message_id"`
	Text          string         `json:"text"`
	Chat          telegramChat   `json:"chat"`
	From          *telegramUser  `json:"from"`
	NewChatMembers []telegramUser `json:"new_chat_members"`
}

type telegramChatMemberUpdated struct {
	Chat          telegramChat       `json:"chat"`
	NewChatMember telegramMemberInfo `json:"new_chat_member"`
}

type telegramMemberInfo struct {
	User   telegramUser `json:"user"`
	Status string       `json:"status"`
}

type telegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type telegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type telegramUpdatesResponse struct {
	OK          bool             `json:"ok"`
	Description string           `json:"description"`
	ErrorCode   int              `json:"error_code"`
	Result      []telegramUpdate `json:"result"`
}

func (r *BotRuntime) Run(ctx context.Context) {
	if r.Logger == nil { r.Logger = slog.Default() }
	if r.SyncInterval <= 0 { r.SyncInterval = 10 * time.Second }
	if r.HTTPClient == nil { r.HTTPClient = &http.Client{Timeout: 30 * time.Second} }
	if r.runners == nil { r.runners = make(map[int64]botRunner) }

	r.syncRunners(ctx)
	ticker := time.NewTicker(r.SyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.syncRunners(ctx)
		case <-ctx.Done():
			for _, runner := range r.runners { runner.cancel() }
			return
		}
	}
}

func (r *BotRuntime) syncRunners(ctx context.Context) {
	rows, err := r.DB.Query(ctx, `
		SELECT id, token_encrypted, last_update_id
		FROM telegram_bots
		WHERE status = 'active'
	`)
	if err != nil {
		r.Logger.Error("load active Telegram Bots failed", "error", err)
		return
	}
	defer rows.Close()

	key, keyErr := decodeEncryptionKey(r.EncryptionKey)
	desired := make(map[int64]struct{})
	type activeBot struct { id int64; token string; offset int64 }
	var bots []activeBot
	for rows.Next() {
		var id, offset int64
		var encrypted string
		if err := rows.Scan(&id, &encrypted, &offset); err != nil {
			r.Logger.Error("scan active Telegram Bot failed", "error", err)
			return
		}
		if keyErr != nil {
			r.markBotError(ctx, id)
			continue
		}
		token, err := decryptSecret(key, encrypted)
		if err != nil {
			r.Logger.Error("decrypt Telegram Bot token failed", "bot_id", id)
			r.markBotError(ctx, id)
			continue
		}
		desired[id] = struct{}{}
		bots = append(bots, activeBot{id: id, token: token, offset: offset})
	}
	if err := rows.Err(); err != nil {
		r.Logger.Error("read active Telegram Bots failed", "error", err)
		return
	}
	for id, runner := range r.runners {
		if _, ok := desired[id]; !ok { runner.cancel(); delete(r.runners, id) }
	}
	for _, bot := range bots {
		tokenHash := hashToken(bot.token)
		if current, ok := r.runners[bot.id]; ok && current.tokenHash == tokenHash { continue } else if ok { current.cancel() }
		runnerCtx, cancel := context.WithCancel(ctx)
		r.runners[bot.id] = botRunner{cancel: cancel, tokenHash: tokenHash}
		go r.runBot(runnerCtx, bot.id, bot.token, bot.offset)
	}
}

func (r *BotRuntime) runBot(ctx context.Context, botID int64, token string, offset int64) {
	for {
		updates, err := r.getUpdates(ctx, token, offset)
		if err != nil {
			var apiErr *telegramAPIError
			if errors.As(err, &apiErr) && apiErr.Code == http.StatusUnauthorized {
				r.Logger.Error("Telegram rejected Bot Token; stopping worker", "bot_id", botID)
				r.markBotError(ctx, botID)
				return
			}
			r.Logger.Warn("Telegram polling failed", "bot_id", botID, "error", err)
			if !sleepContext(ctx, 3*time.Second) { return }
			continue
		}
		processFailed := false
		for _, update := range updates {
			if err := r.processUpdate(ctx, botID, token, update); err != nil {
				r.Logger.Error("process Telegram update failed", "bot_id", botID, "update_id", update.UpdateID, "error", err)
				processFailed = true
				break
			}
			nextOffset := update.UpdateID + 1
			if _, err := r.DB.Exec(ctx, "UPDATE telegram_bots SET last_update_id = GREATEST(last_update_id, $1) WHERE id = $2", nextOffset, botID); err != nil {
				r.Logger.Error("persist Telegram polling offset failed", "bot_id", botID, "error", err)
				processFailed = true
				break
			}
			offset = nextOffset
		}
		if processFailed && !sleepContext(ctx, 2*time.Second) { return }
	}
}

type telegramAPIError struct { Code int; Description string }
func (e *telegramAPIError) Error() string { return fmt.Sprintf("Telegram API %d: %s", e.Code, e.Description) }

func (r *BotRuntime) getUpdates(ctx context.Context, token string, offset int64) ([]telegramUpdate, error) {
	base := strings.TrimRight(r.TelegramAPIBaseURL, "/")
	if base == "" { base = "https://api.telegram.org" }
	query := url.Values{}
	query.Set("timeout", "20")
	query.Set("allowed_updates", "[\"message\",\"my_chat_member\",\"chat_member\"]")
	if offset > 0 { query.Set("offset", strconv.FormatInt(offset, 10)) }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/bot"+token+"/getUpdates?"+query.Encode(), nil)
	if err != nil { return nil, err }
	resp, err := r.HTTPClient.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil { return nil, err }
	var result telegramUpdatesResponse
	if err := json.Unmarshal(body, &result); err != nil { return nil, err }
	if resp.StatusCode != http.StatusOK || !result.OK {
		return nil, &telegramAPIError{Code: result.ErrorCodeOr(resp.StatusCode), Description: result.Description}
	}
	return result.Result, nil
}

func (r *telegramUpdatesResponse) ErrorCodeOr(fallback int) int {
	if r.ErrorCode != 0 { return r.ErrorCode }
	return fallback
}

func (r *BotRuntime) processUpdate(ctx context.Context, botID int64, token string, update telegramUpdate) error {
	var chat *telegramChat
	if update.Message != nil { chat = &update.Message.Chat }
	if update.MyChatMember != nil { chat = &update.MyChatMember.Chat }
	if update.ChatMember != nil { chat = &update.ChatMember.Chat }
	if chat == nil || chat.ID == 0 || (chat.Type != "group" && chat.Type != "supergroup") { return nil }

	var groupID int64
	err := r.DB.QueryRow(ctx, `
		INSERT INTO telegram_groups (bot_id, telegram_chat_id, title)
		VALUES ($1, $2, $3)
		ON CONFLICT (bot_id, telegram_chat_id)
		DO UPDATE SET title = EXCLUDED.title
		RETURNING id
	`, botID, chat.ID, chat.Title).Scan(&groupID)
	if err != nil { return err }

	if update.Message != nil {
		if update.Message.From != nil {
			if err := r.upsertMember(ctx, groupID, *update.Message.From, "member"); err != nil { return err }
		}
		for _, member := range update.Message.NewChatMembers {
			if err := r.upsertMember(ctx, groupID, member, "member"); err != nil { return err }
		}
		if err := r.enforceGroupRules(ctx, token, groupID, chat.ID, *update.Message); err != nil {
			r.Logger.Warn("group moderation action failed", "bot_id", botID, "group_id", groupID, "error", err)
		}
	}
	if update.ChatMember != nil {
		member := update.ChatMember.NewChatMember
		if member.User.ID != 0 {
			if err := r.upsertMember(ctx, groupID, member.User, member.Status); err != nil { return err }
		}
	}
	return nil
}


type storedGroupRules struct {
	BlockLinks bool
	BlockedKeywords []string
}

func (r *BotRuntime) enforceGroupRules(ctx context.Context, token string, groupID, chatID int64, message telegramMessage) error {
	if message.MessageID <= 0 || strings.TrimSpace(message.Text) == "" { return nil }
	var rules storedGroupRules
	err := r.DB.QueryRow(ctx, `
		SELECT block_links, blocked_keywords
		FROM group_moderation_rules WHERE group_id = $1
	`, groupID).Scan(&rules.BlockLinks, &rules.BlockedKeywords)
	if errors.Is(err, pgx.ErrNoRows) { return nil }
	if err != nil { return err }
	text := strings.ToLower(message.Text)
	matched := false
	if rules.BlockLinks && (strings.Contains(text, "http://") || strings.Contains(text, "https://") ||
		strings.Contains(text, "www.") || strings.Contains(text, "t.me/")) {
		matched = true
	}
	if !matched {
		for _, keyword := range rules.BlockedKeywords {
			if keyword != "" && strings.Contains(text, strings.ToLower(keyword)) {
				matched = true
				break
			}
		}
	}
	if !matched { return nil }
	return r.deleteTelegramMessage(ctx, token, chatID, message.MessageID)
}

func (r *BotRuntime) deleteTelegramMessage(ctx context.Context, token string, chatID, messageID int64) error {
	base := strings.TrimRight(r.TelegramAPIBaseURL, "/")
	if base == "" { base = "https://api.telegram.org" }
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("message_id", strconv.FormatInt(messageID, 10))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/bot"+token+"/deleteMessage", strings.NewReader(form.Encode()))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := r.HTTPClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil { return err }
	var result struct { OK bool `json:"ok"`; Description string `json:"description"` }
	if err := json.Unmarshal(body, &result); err != nil { return err }
	if resp.StatusCode != http.StatusOK || !result.OK {
		return fmt.Errorf("Telegram deleteMessage failed (HTTP %d): %s", resp.StatusCode, result.Description)
	}
	return nil
}

func (r *BotRuntime) upsertMember(ctx context.Context, groupID int64, user telegramUser, status string) error {
	displayName := strings.TrimSpace(user.FirstName + " " + user.LastName)
	_, err := r.DB.Exec(ctx, `
		INSERT INTO telegram_members (group_id, telegram_user_id, username, display_name, status)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5)
		ON CONFLICT (group_id, telegram_user_id)
		DO UPDATE SET username = EXCLUDED.username, display_name = EXCLUDED.display_name, status = EXCLUDED.status
	`, groupID, user.ID, user.Username, displayName, status)
	return err
}

func (r *BotRuntime) markBotError(ctx context.Context, botID int64) {
	_, _ = r.DB.Exec(ctx, "UPDATE telegram_bots SET status = 'error', updated_at = NOW() WHERE id = $1", botID)
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done(): return false
	case <-timer.C: return true
	}
}


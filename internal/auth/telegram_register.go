package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type TelegramRegistration struct {
	UserID string
	Email string
	Password string
}

// RegisterTelegramAccount creates a normal platform user and binds the Telegram
// identity atomically. The password is generated server-side and returned only
// to the caller so the platform Bot can deliver it in the user's private chat.
func (s *Service) RegisterTelegramAccount(ctx context.Context, identity TelegramIdentity) (TelegramRegistration, error) {
	if identity.ID <= 0 {
		return TelegramRegistration{}, ErrInvalidInput
	}
	password, _, err := newToken()
	if err != nil {
		return TelegramRegistration{}, fmt.Errorf("generate Telegram account password: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return TelegramRegistration{}, fmt.Errorf("hash Telegram account password: %w", err)
	}
	displayName := strings.TrimSpace(strings.Join([]string{identity.FirstName, identity.LastName}, " "))
	if displayName == "" && identity.Username != "" {
		displayName = identity.Username
	}
	if displayName == "" {
		displayName = fmt.Sprintf("Telegram %d", identity.ID)
	}
	if len(displayName) > 120 {
		displayName = displayName[:120]
	}
	email := fmt.Sprintf("tg-%d@telegram.local", identity.ID)

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TelegramRegistration{}, fmt.Errorf("begin Telegram registration: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", identity.ID); err != nil {
		return TelegramRegistration{}, fmt.Errorf("lock Telegram registration identity: %w", err)
	}
	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE telegram_id=$1)", identity.ID).Scan(&exists); err != nil {
		return TelegramRegistration{}, fmt.Errorf("check Telegram registration identity: %w", err)
	}
	if exists {
		return TelegramRegistration{}, ErrTelegramAlreadyLinked
	}

	var tenantID, userID string
	if err := tx.QueryRow(ctx, "INSERT INTO tenants(name) VALUES($1) RETURNING id::text", displayName).Scan(&tenantID); err != nil {
		return TelegramRegistration{}, fmt.Errorf("create Telegram user tenant: %w", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO users(tenant_id,email,password_hash,telegram_id,telegram_username,telegram_first_name,telegram_last_name,display_name)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id::text`,
		tenantID, email, string(hash), identity.ID, identity.Username, identity.FirstName, identity.LastName, displayName).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return TelegramRegistration{}, ErrTelegramAlreadyLinked
		}
		return TelegramRegistration{}, fmt.Errorf("create Telegram platform user: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,metadata)
		VALUES($1,$2,'auth.telegram_register','user',$3,jsonb_build_object('telegram_id',$4::bigint))`,
		tenantID, userID, userID, identity.ID); err != nil {
		return TelegramRegistration{}, fmt.Errorf("audit Telegram registration: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		if isUniqueViolation(err) {
			return TelegramRegistration{}, ErrTelegramAlreadyLinked
		}
		return TelegramRegistration{}, fmt.Errorf("commit Telegram registration: %w", err)
	}
	return TelegramRegistration{UserID: userID, Email: email, Password: password}, nil
}

var _ = errors.Is

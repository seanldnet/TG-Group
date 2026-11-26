package auth

import (
    "context"
    "crypto/sha256"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/jackc/pgx/v5"
)

var (
    ErrTelegramAlreadyLinked = errors.New("Telegram account is already linked")
    ErrTelegramChallengeExpired = errors.New("Telegram binding challenge is invalid or expired")
    ErrTelegramAlreadyBound = errors.New("user already has a Telegram account linked")
)

const telegramLinkLifetime = 5 * time.Minute

type TelegramLinkChallenge struct {
    Token string
    ExpiresAt time.Time
}

type TelegramIdentity struct {
    ID int64
    Username string
    FirstName string
    LastName string
}

func (s *Service) CreateTelegramLinkChallenge(ctx context.Context, userID string) (TelegramLinkChallenge, error) {
    token, _, err := newToken()
    if err != nil { return TelegramLinkChallenge{}, err }
    hash := sha256.Sum256([]byte(token))
    expires := time.Now().UTC().Add(telegramLinkLifetime)
    tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil { return TelegramLinkChallenge{}, fmt.Errorf("begin Telegram link challenge: %w", err) }
    defer tx.Rollback(ctx)
    if _, err := tx.Exec(ctx, `UPDATE telegram_identity_link_challenges SET consumed_at=now()
        WHERE user_id=$1 AND consumed_at IS NULL`, userID); err != nil {
        return TelegramLinkChallenge{}, fmt.Errorf("invalidate previous Telegram link challenges: %w", err)
    }
    if _, err := tx.Exec(ctx, `INSERT INTO telegram_identity_link_challenges(user_id,token_hash,expires_at)
        VALUES($1,$2,$3)`, userID, hash[:], expires); err != nil {
        return TelegramLinkChallenge{}, fmt.Errorf("create Telegram link challenge: %w", err)
    }
    if err := tx.Commit(ctx); err != nil { return TelegramLinkChallenge{}, fmt.Errorf("commit Telegram link challenge: %w", err) }
    return TelegramLinkChallenge{Token: token, ExpiresAt: expires}, nil
}

// CompleteTelegramLink accepts identity only from a Telegram Bot API update handler.
// The caller must never populate identity fields from an HTTP client request.
func (s *Service) CompleteTelegramLink(ctx context.Context, token string, identity TelegramIdentity) error {
    token = strings.TrimSpace(token)
    if token == "" || identity.ID <= 0 || identity.ID > 9223372036854775807 {
        return ErrTelegramChallengeExpired
    }
    hash := sha256.Sum256([]byte(token))
    tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil { return fmt.Errorf("begin Telegram identity binding: %w", err) }
    defer tx.Rollback(ctx)

    var userID string
    err = tx.QueryRow(ctx, `SELECT user_id::text FROM telegram_identity_link_challenges
        WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at>now()
        FOR UPDATE`, hash[:]).Scan(&userID)
    if errors.Is(err, pgx.ErrNoRows) { return ErrTelegramChallengeExpired }
    if err != nil { return fmt.Errorf("load Telegram link challenge: %w", err) }

    var existingID *int64
    if err := tx.QueryRow(ctx, "SELECT telegram_id FROM users WHERE id=$1 FOR UPDATE", userID).Scan(&existingID); err != nil {
        if errors.Is(err, pgx.ErrNoRows) { return ErrUnauthorized }
        return fmt.Errorf("load user for Telegram binding: %w", err)
    }
    if existingID != nil { return ErrTelegramAlreadyBound }

    tag, err := tx.Exec(ctx, `UPDATE users SET telegram_id=$2,telegram_username=$3,telegram_first_name=$4,telegram_last_name=$5,updated_at=now()
        WHERE id=$1 AND telegram_id IS NULL`, userID, identity.ID, identity.Username, identity.FirstName, identity.LastName)
    if err != nil {
        if isUniqueViolation(err) { return ErrTelegramAlreadyLinked }
        return fmt.Errorf("bind Telegram identity: %w", err)
    }
    if tag.RowsAffected() != 1 { return ErrTelegramAlreadyBound }

    tag, err = tx.Exec(ctx, `UPDATE telegram_identity_link_challenges SET consumed_at=now()
        WHERE token_hash=$1 AND consumed_at IS NULL`, hash[:])
    if err != nil { return fmt.Errorf("consume Telegram link challenge: %w", err) }
    if tag.RowsAffected() != 1 { return ErrTelegramChallengeExpired }

    _, err = tx.Exec(ctx, `INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,metadata)
        SELECT tenant_id,id,'auth.telegram_link','user',id::text,jsonb_build_object('telegram_id',$2)
        FROM users WHERE id=$1`, userID, identity.ID)
    if err != nil { return fmt.Errorf("audit Telegram identity binding: %w", err) }
    if err := tx.Commit(ctx); err != nil {
        if isUniqueViolation(err) { return ErrTelegramAlreadyLinked }
        return fmt.Errorf("commit Telegram identity binding: %w", err)
    }
    return nil
}

func (s *Service) UnlinkTelegramIdentity(ctx context.Context, userID string) error {
    tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil { return fmt.Errorf("begin Telegram identity unlink: %w", err) }
    defer tx.Rollback(ctx)
    var telegramID int64
    var tenantID string
    err = tx.QueryRow(ctx, "SELECT telegram_id,tenant_id::text FROM users WHERE id=$1 AND telegram_id IS NOT NULL FOR UPDATE", userID).Scan(&telegramID,&tenantID)
    if errors.Is(err,pgx.ErrNoRows) { return ErrTelegramAlreadyBound }
    if err != nil { return fmt.Errorf("load Telegram identity for unlink: %w",err) }
    if _,err:=tx.Exec(ctx,"UPDATE users SET telegram_id=NULL,updated_at=now() WHERE id=$1",userID);err!=nil{return fmt.Errorf("unlink Telegram identity: %w",err)}
    if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,metadata)
        VALUES($1,$2,'auth.telegram_unlink','user',$2,jsonb_build_object('telegram_id',$3))`,tenantID,userID,telegramID);err!=nil{return fmt.Errorf("audit Telegram identity unlink: %w",err)}
    if err:=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit Telegram identity unlink: %w",err)}
    return nil
}

type LinkedTelegramIdentity struct {
    ID *int64 `json:"id"`
    Username string `json:"username"`
    FirstName string `json:"first_name"`
    LastName string `json:"last_name"`
}

func (s *Service) TelegramIdentity(ctx context.Context, userID string) (LinkedTelegramIdentity, error) {
    var identity LinkedTelegramIdentity
    if err := s.pool.QueryRow(ctx, "SELECT telegram_id,telegram_username,telegram_first_name,telegram_last_name FROM users WHERE id=$1", userID).Scan(&identity.ID,&identity.Username,&identity.FirstName,&identity.LastName); err != nil {
        return LinkedTelegramIdentity{}, fmt.Errorf("read Telegram identity: %w", err)
    }
    return identity, nil
}

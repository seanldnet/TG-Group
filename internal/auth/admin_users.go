package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidAdminUserStatus = errors.New("invalid admin user status")
	ErrAdminSelfLock = errors.New("administrator cannot lock their own account")
	ErrAdminUserNotFound = errors.New("admin user not found")
)

type AdminUser struct {
	ID string `json:"id"`
	TenantID string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantStatus string `json:"tenant_status"`
	Email string `json:"email"`
	DisplayName string `json:"display_name"`
	Role string `json:"role"`
	Status string `json:"status"`
	TelegramID *int64 `json:"telegram_id"`
	TelegramUsername string `json:"telegram_username"`
	TelegramFirstName string `json:"telegram_first_name"`
	TelegramLastName string `json:"telegram_last_name"`
	BotCount int `json:"bot_count"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) ListAdminUsers(ctx context.Context, search string, limit, offset int) ([]AdminUser, int, error) {
	search = strings.TrimSpace(search)
	if len(search) > 120 { search = search[:120] }
	if limit < 1 || limit > 100 { limit = 25 }
	if offset < 0 { offset = 0 }
	where := "WHERE $1 = '' OR u.email ILIKE '%' || $1 || '%' OR u.display_name ILIKE '%' || $1 || '%' OR u.id::text ILIKE '%' || $1 || '%' OR COALESCE(u.telegram_id::text,'') ILIKE '%' || $1 || '%' OR u.telegram_username ILIKE '%' || $1 || '%'"
	var total int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM users u "+where, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.pool.Query(ctx, "SELECT u.id::text,u.tenant_id::text,t.name,t.status,u.email,u.display_name,u.role,u.status,u.telegram_id,u.telegram_username,u.telegram_first_name,u.telegram_last_name,(SELECT count(*)::int FROM telegram_bots b WHERE b.tenant_id=u.tenant_id),u.created_at FROM users u JOIN tenants t ON t.id=u.tenant_id "+where+" ORDER BY u.created_at DESC,u.id LIMIT $2 OFFSET $3", search, limit, offset)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	items := make([]AdminUser, 0)
	for rows.Next() {
		var item AdminUser
		if err := rows.Scan(&item.ID,&item.TenantID,&item.TenantName,&item.TenantStatus,&item.Email,&item.DisplayName,&item.Role,&item.Status,&item.TelegramID,&item.TelegramUsername,&item.TelegramFirstName,&item.TelegramLastName,&item.BotCount,&item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items,item)
	}
	if err := rows.Err(); err != nil { return nil, 0, err }
	return items,total,nil
}

func (s *Service) UpdateAdminUserStatus(ctx context.Context, actorID, targetID, status string) error {
	if status != "active" && status != "locked" { return ErrInvalidAdminUserStatus }
	if actorID == targetID && status == "locked" { return ErrAdminSelfLock }
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil { return err }
	defer tx.Rollback(ctx)
	var tenantID, role string
	err = tx.QueryRow(ctx, "SELECT tenant_id::text,role FROM users WHERE id=$1 FOR UPDATE", targetID).Scan(&tenantID,&role)
	if errors.Is(err,pgx.ErrNoRows) { return ErrAdminUserNotFound }
	if err != nil { return err }
	if role == "admin" && status == "locked" { return ErrInvalidAdminUserStatus }
	if _,err = tx.Exec(ctx,"UPDATE users SET status=$2,updated_at=now() WHERE id=$1",targetID,status);err!=nil{return err}
	if _,err = tx.Exec(ctx,"INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,metadata) VALUES($1,$2,'platform.user_status','user',$3,jsonb_build_object('status',$4::text))",tenantID,actorID,targetID,status);err!=nil{return err}
	return tx.Commit(ctx)
}

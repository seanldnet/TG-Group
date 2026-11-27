package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidDisplayName = errors.New("invalid display name")
	ErrProfileUserNotFound = errors.New("profile user not found")
)

func (s *Service) UpdateDisplayName(ctx context.Context, userID, displayName string) (User, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || len([]rune(displayName)) > 40 {
		return User{}, ErrInvalidDisplayName
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil { return User{}, err }
	defer tx.Rollback(ctx)

	var user User
	err = tx.QueryRow(ctx, `UPDATE users SET display_name=$2,updated_at=now()
		WHERE id=$1 AND status='active'
		RETURNING id::text,tenant_id::text,email,display_name,role,created_at`,
		userID, displayName).Scan(&user.ID,&user.TenantID,&user.Email,&user.DisplayName,&user.Role,&user.CreatedAt)
	if errors.Is(err,pgx.ErrNoRows) { return User{},ErrProfileUserNotFound }
	if err != nil { return User{},err }
	if _,err=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id,metadata)
		VALUES($1,$2,'auth.profile_update','user',$2,jsonb_build_object('display_name',$3::text))`,
		user.TenantID,user.ID,displayName);err!=nil{return User{},err}
	if err=tx.Commit(ctx);err!=nil{return User{},err}
	return user,nil
}

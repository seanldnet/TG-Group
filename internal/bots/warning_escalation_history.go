package bots

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrWarningEscalationNotRetryable = errors.New("warning escalation is not retryable")

type WarningEscalationExecution struct {
	ID int64 `json:"id"`
	GroupID string `json:"group_id"`
	RuleID string `json:"rule_id"`
	TelegramUserID int64 `json:"telegram_user_id"`
	WarningCount int `json:"warning_count"`
	Action string `json:"action"`
	DurationMinutes int `json:"duration_minutes"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) ListWarningEscalations(ctx context.Context, tenantID, groupID string, userID int64, limit int) ([]WarningEscalationExecution, error) {
	if limit < 1 || limit > 100 { limit = 50 }
	rows, err := s.pool.Query(ctx, `SELECT e.id,e.group_id::text,e.rule_id::text,e.telegram_user_id,e.warning_count,e.action,e.duration_minutes,e.status,e.detail,e.created_at,e.updated_at
		FROM telegram_group_warning_escalations e
		JOIN telegram_groups g ON g.id=e.group_id AND g.tenant_id=e.tenant_id
		WHERE e.tenant_id=$1 AND e.group_id=$2 AND ($3::bigint=0 OR e.telegram_user_id=$3)
		ORDER BY e.created_at DESC,e.id DESC LIMIT $4`, tenantID, groupID, userID, limit)
	if err != nil { return nil, fmt.Errorf("list warning escalation executions: %w", err) }
	defer rows.Close()
	items := make([]WarningEscalationExecution, 0)
	for rows.Next() {
		var item WarningEscalationExecution
		if err := rows.Scan(&item.ID,&item.GroupID,&item.RuleID,&item.TelegramUserID,&item.WarningCount,&item.Action,&item.DurationMinutes,&item.Status,&item.Detail,&item.CreatedAt,&item.UpdatedAt); err != nil { return nil, err }
		items = append(items,item)
	}
	return items, rows.Err()
}

// RetryWarningEscalation retries only a previously failed execution. The status
// transition is an atomic compare-and-set, so concurrent requests cannot apply
// the same Telegram moderation action twice.
func (s *Service) RetryWarningEscalation(ctx context.Context, tenantID, actorUserID, groupID string, executionID int64) (WarningEscalationExecution, error) {
	var item WarningEscalationExecution
	err := s.pool.QueryRow(ctx, `UPDATE telegram_group_warning_escalations e
		SET status='processing',detail='manual retry requested',updated_at=now()
		FROM telegram_groups g
		WHERE e.id=$1 AND e.tenant_id=$2 AND e.group_id=$3
		AND e.status='failed' AND g.id=e.group_id AND g.tenant_id=e.tenant_id
		RETURNING e.id,e.group_id::text,e.rule_id::text,e.telegram_user_id,e.warning_count,e.action,e.duration_minutes,e.status,e.detail,e.created_at,e.updated_at`,
		executionID,tenantID,groupID).Scan(&item.ID,&item.GroupID,&item.RuleID,&item.TelegramUserID,&item.WarningCount,&item.Action,&item.DurationMinutes,&item.Status,&item.Detail,&item.CreatedAt,&item.UpdatedAt)
	if errors.Is(err,pgx.ErrNoRows) { return WarningEscalationExecution{},ErrWarningEscalationNotRetryable }
	if err != nil { return WarningEscalationExecution{},fmt.Errorf("reserve warning escalation retry: %w",err) }

	actionErr := s.ModerateGroupMember(ctx,tenantID,actorUserID,groupID,item.TelegramUserID,item.Action,item.DurationMinutes)
	status,detail := "succeeded","manual retry applied successfully"
	if actionErr != nil {
		status,detail = "failed",actionErr.Error()
		if len([]rune(detail)) > 500 { detail=string([]rune(detail)[:500]) }
	}
	if err := s.pool.QueryRow(ctx, `UPDATE telegram_group_warning_escalations
		SET status=$2,detail=$3,updated_at=$4 WHERE id=$1
		RETURNING id,group_id::text,rule_id::text,telegram_user_id,warning_count,action,duration_minutes,status,detail,created_at,updated_at`,
		item.ID,status,detail,time.Now().UTC()).Scan(&item.ID,&item.GroupID,&item.RuleID,&item.TelegramUserID,&item.WarningCount,&item.Action,&item.DurationMinutes,&item.Status,&item.Detail,&item.CreatedAt,&item.UpdatedAt); err != nil {
		return item,fmt.Errorf("save warning escalation retry result: %w",err)
	}
	if actionErr != nil { return item,fmt.Errorf("retry warning escalation: %w",actionErr) }
	return item,nil
}

package bots

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ApplyWarningEscalation evaluates enabled escalation rules after a warning is
// recorded. A unique execution key prevents the same rule/threshold from
// applying twice for the same warning count. ModerateGroupMember re-checks the
// Bot's live permission and refuses to act on group administrators/creators.
func (s *Service) ApplyWarningEscalation(ctx context.Context, tenantID, groupID string, userID int64) error {
	if userID <= 0 { return ErrInvalidGroupRule }
	var warningCount int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM telegram_group_member_warnings w
		JOIN telegram_groups g ON g.id=w.group_id AND g.tenant_id=w.tenant_id
		WHERE w.tenant_id=$1 AND w.group_id=$2 AND w.telegram_user_id=$3`, tenantID, groupID, userID).Scan(&warningCount); err != nil {
		return fmt.Errorf("count warnings for escalation: %w", err)
	}
	if warningCount == 0 { return nil }
	rows, err := s.pool.Query(ctx, `SELECT id::text,config FROM telegram_group_rules
		WHERE tenant_id=$1 AND group_id=$2 AND rule_type='warning_escalation' AND enabled=true
		ORDER BY priority,created_at`, tenantID, groupID)
	if err != nil { return fmt.Errorf("load warning escalation rules: %w", err) }
	defer rows.Close()
	type candidate struct { id string; action string; duration int }
	var candidates []candidate
	for rows.Next() {
		var ruleID string
		var raw []byte
		if err := rows.Scan(&ruleID,&raw); err != nil { return err }
		var cfg warningEscalationRuleConfig
		if json.Unmarshal(raw,&cfg) != nil { continue }
		// Select only the highest threshold reached in this rule.
		var selected *warningEscalationLevel
		for i := range cfg.Levels { if cfg.Levels[i].Threshold <= warningCount { selected=&cfg.Levels[i] } }
		if selected != nil { candidates=append(candidates,candidate{id:ruleID,action:selected.Action,duration:selected.DurationMinutes}) }
	}
	if err := rows.Err(); err != nil { return err }
	for _, item := range candidates {
		var executionID int64
		err := s.pool.QueryRow(ctx, `INSERT INTO telegram_group_warning_escalations
			(tenant_id,group_id,rule_id,telegram_user_id,warning_count,action,duration_minutes,status,processing_started_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,'processing',now())
			ON CONFLICT(tenant_id,group_id,rule_id,telegram_user_id,warning_count) DO NOTHING
			RETURNING id`, tenantID,groupID,item.id,userID,warningCount,item.action,item.duration).Scan(&executionID)
		if errors.Is(err,pgx.ErrNoRows) { continue }
		if err != nil { return fmt.Errorf("reserve warning escalation: %w",err) }
		actionErr := s.ModerateGroupMember(ctx,tenantID,"",groupID,userID,item.action,item.duration)
		status,detail := "succeeded","automatic warning escalation applied"
		if actionErr != nil {
			status,detail = "failed",actionErr.Error()
			if len([]rune(detail))>500 { detail=string([]rune(detail)[:500]) }
		}
		if _,err:=s.pool.Exec(ctx,`UPDATE telegram_group_warning_escalations SET status=$2,detail=$3,updated_at=$4,processing_started_at=NULL,next_retry_at=$5 WHERE id=$1`,executionID,status,detail,time.Now().UTC(),func() any { if actionErr != nil { return time.Now().UTC().Add(WarningEscalationRetryDelay(1)) }; return nil }());err!=nil {
			return fmt.Errorf("update warning escalation execution %d: %w",executionID,err)
		}
		if actionErr != nil { return fmt.Errorf("warning escalation %s failed: %w",item.action,actionErr) }
	}
	return nil
}

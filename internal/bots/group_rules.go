package bots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidGroupRule = errors.New("invalid group rule")
	ErrGroupRuleConflict = errors.New("group rule already exists")
)

type GroupRule struct {
	ID string `json:"id"`
	GroupID string `json:"group_id"`
	RuleType string `json:"rule_type"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
	Priority int `json:"priority"`
	Config json.RawMessage `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GroupRuleInput struct {
	RuleType string `json:"rule_type"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
	Priority int `json:"priority"`
	Config json.RawMessage `json:"config"`
}

type keywordRuleConfig struct {
	Words []string `json:"words"`
	Action string `json:"action"`
}
type spamRuleConfig struct {
	MaxMessages int `json:"max_messages"`
	WindowSeconds int `json:"window_seconds"`
	Action string `json:"action"`
}
type adRuleConfig struct {
	Patterns []string `json:"patterns"`
	Action string `json:"action"`
}
type noticeRuleConfig struct {
	Text string `json:"text"`
}

func ValidateGroupRule(input GroupRuleInput) error {
	input.RuleType = strings.TrimSpace(input.RuleType)
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) == 0 || len([]rune(input.Name)) > 80 || input.Priority < 0 || input.Priority > 1000 {
		return ErrInvalidGroupRule
	}
	if len(input.Config) == 0 || !json.Valid(input.Config) {
		return ErrInvalidGroupRule
	}
	decode := func(dst any) error {
		decoder := json.NewDecoder(bytes.NewReader(input.Config))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(dst); err != nil { return ErrInvalidGroupRule }
		return nil
	}
	validTerms := func(values []string) bool {
		if len(values) == 0 || len(values) > 100 { return false }
		for _, value := range values {
			if value == "" || len([]rune(value)) > 100 { return false }
		}
		return true
	}
	switch input.RuleType {
	case "keyword_filter", "sensitive_word":
		var cfg keywordRuleConfig
		if decode(&cfg) != nil || !validTerms(cfg.Words) || cfg.Action != "delete" { return ErrInvalidGroupRule }
	case "anti_spam":
		var cfg spamRuleConfig
		if decode(&cfg) != nil || cfg.MaxMessages < 2 || cfg.MaxMessages > 100 || cfg.WindowSeconds < 2 || cfg.WindowSeconds > 3600 || cfg.Action != "delete" { return ErrInvalidGroupRule }
	case "anti_ad":
		var cfg adRuleConfig
		if decode(&cfg) != nil || !validTerms(cfg.Patterns) || cfg.Action != "delete" { return ErrInvalidGroupRule }
	case "welcome", "leave_notice":
		var cfg noticeRuleConfig
		if decode(&cfg) != nil || strings.TrimSpace(cfg.Text) == "" || len([]rune(cfg.Text)) > 4096 { return ErrInvalidGroupRule }
	default:
		return ErrInvalidGroupRule
	}
	return nil
}

func (s *Service) ListGroupRules(ctx context.Context, tenantID, groupID string) ([]GroupRule, error) {
	rows, err := s.pool.Query(ctx, `SELECT r.id::text,r.group_id::text,r.rule_type,r.name,r.enabled,r.priority,r.config,r.created_at,r.updated_at
		FROM telegram_group_rules r JOIN telegram_groups g ON g.id=r.group_id AND g.tenant_id=r.tenant_id
		WHERE r.tenant_id=$1 AND r.group_id=$2 ORDER BY r.priority,r.created_at`, tenantID, groupID)
	if err != nil { return nil, fmt.Errorf("list group rules: %w", err) }
	defer rows.Close()
	result := make([]GroupRule, 0)
	for rows.Next() {
		var item GroupRule
		if err := rows.Scan(&item.ID,&item.GroupID,&item.RuleType,&item.Name,&item.Enabled,&item.Priority,&item.Config,&item.CreatedAt,&item.UpdatedAt); err != nil { return nil, err }
		result = append(result,item)
	}
	return result, rows.Err()
}

func (s *Service) CreateGroupRule(ctx context.Context, tenantID, groupID string, input GroupRuleInput) (GroupRule, error) {
	if err := ValidateGroupRule(input); err != nil { return GroupRule{}, err }
	var item GroupRule
	err := s.pool.QueryRow(ctx, `INSERT INTO telegram_group_rules(tenant_id,group_id,rule_type,name,enabled,priority,config)
		SELECT $1,g.id,$3,$4,$5,$6,$7::jsonb FROM telegram_groups g WHERE g.tenant_id=$1 AND g.id=$2
		RETURNING id::text,group_id::text,rule_type,name,enabled,priority,config,created_at,updated_at`,
		tenantID,groupID,input.RuleType,strings.TrimSpace(input.Name),input.Enabled,input.Priority,[]byte(input.Config)).
		Scan(&item.ID,&item.GroupID,&item.RuleType,&item.Name,&item.Enabled,&item.Priority,&item.Config,&item.CreatedAt,&item.UpdatedAt)
	if errors.Is(err,pgx.ErrNoRows) { return GroupRule{},ErrNotFound }
	if err != nil {
		if isUniqueViolation(err) { return GroupRule{},ErrGroupRuleConflict }
		return GroupRule{},fmt.Errorf("create group rule: %w",err)
	}
	return item,nil
}

func (s *Service) UpdateGroupRule(ctx context.Context, tenantID, groupID, ruleID string, input GroupRuleInput) (GroupRule,error) {
	if err := ValidateGroupRule(input); err != nil { return GroupRule{},err }
	var item GroupRule
	err := s.pool.QueryRow(ctx, `UPDATE telegram_group_rules SET rule_type=$4,name=$5,enabled=$6,priority=$7,config=$8::jsonb,updated_at=now()
		WHERE tenant_id=$1 AND group_id=$2 AND id=$3
		RETURNING id::text,group_id::text,rule_type,name,enabled,priority,config,created_at,updated_at`,
		tenantID,groupID,ruleID,input.RuleType,strings.TrimSpace(input.Name),input.Enabled,input.Priority,[]byte(input.Config)).
		Scan(&item.ID,&item.GroupID,&item.RuleType,&item.Name,&item.Enabled,&item.Priority,&item.Config,&item.CreatedAt,&item.UpdatedAt)
	if errors.Is(err,pgx.ErrNoRows) { return GroupRule{},ErrNotFound }
	if err != nil {
		if isUniqueViolation(err) { return GroupRule{},ErrGroupRuleConflict }
		return GroupRule{},fmt.Errorf("update group rule: %w",err)
	}
	return item,nil
}

func (s *Service) DeleteGroupRule(ctx context.Context, tenantID, groupID, ruleID string) error {
	tag,err := s.pool.Exec(ctx,"DELETE FROM telegram_group_rules WHERE tenant_id=$1 AND group_id=$2 AND id=$3",tenantID,groupID,ruleID)
	if err != nil { return fmt.Errorf("delete group rule: %w",err) }
	if tag.RowsAffected()==0 { return ErrNotFound }
	return nil
}

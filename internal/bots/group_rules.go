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
type joinVerificationRuleConfig struct {
	ChallengeType string `json:"challenge_type"`
	TimeoutSeconds int `json:"timeout_seconds"`
	FailureAction string `json:"failure_action"`
	RequiredChannels []string `json:"required_channels"`
}

type warningEscalationLevel struct { Threshold int `json:"threshold"`; Action string `json:"action"`; DurationMinutes int `json:"duration_minutes,omitempty"` }
type warningEscalationRuleConfig struct { Levels []warningEscalationLevel `json:"levels"` }

type qaRuleConfig struct {
	Triggers []string `json:"triggers"`
	MatchMode string `json:"match_mode"`
	Response string `json:"response"`
	CaseSensitive bool `json:"case_sensitive"`
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
	case "warning_escalation":
		var cfg warningEscalationRuleConfig
		if decode(&cfg) != nil || len(cfg.Levels) == 0 || len(cfg.Levels) > 5 { return ErrInvalidGroupRule }
		previous := 0
		for _, level := range cfg.Levels {
			if level.Threshold <= previous || level.Threshold > 100 || (level.Action != "mute" && level.Action != "ban") { return ErrInvalidGroupRule }
			if level.Action == "mute" && (level.DurationMinutes < 1 || level.DurationMinutes > 10080) { return ErrInvalidGroupRule }
			if level.Action == "ban" && level.DurationMinutes != 0 { return ErrInvalidGroupRule }
			previous = level.Threshold
		}
	case "qa":
		var cfg qaRuleConfig
		if decode(&cfg) != nil || !validTerms(cfg.Triggers) || strings.TrimSpace(cfg.Response) == "" || len([]rune(cfg.Response)) > 4096 || (cfg.MatchMode != "exact" && cfg.MatchMode != "contains" && cfg.MatchMode != "prefix") { return ErrInvalidGroupRule }
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
	case "join_verification":
		var cfg joinVerificationRuleConfig
		if decode(&cfg) != nil || (cfg.ChallengeType != "math" && cfg.ChallengeType != "letters" && cfg.ChallengeType != "alphanumeric") || cfg.TimeoutSeconds < 30 || cfg.TimeoutSeconds > 600 || (cfg.FailureAction != "restrict" && cfg.FailureAction != "kick") || len(cfg.RequiredChannels) > 10 { return ErrInvalidGroupRule }
		seen := make(map[string]struct{}, len(cfg.RequiredChannels))
		for _, channel := range cfg.RequiredChannels { if !validRequiredChannel(channel) { return ErrInvalidGroupRule }; if _, exists := seen[channel]; exists { return ErrInvalidGroupRule }; seen[channel] = struct{}{} }
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

type GroupRuleEvent struct {
	ID int64 `json:"id"`
	GroupID string `json:"group_id"`
	RuleID string `json:"rule_id"`
	TelegramUpdateID int64 `json:"telegram_update_id"`
	TelegramMessageID int64 `json:"telegram_message_id"`
	TelegramUserID int64 `json:"telegram_user_id"`
	Action string `json:"action"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) ListGroupRuleEvents(ctx context.Context, tenantID, groupID string, limit int) ([]GroupRuleEvent, error) {
	if limit < 1 || limit > 100 { limit = 50 }
	rows, err := s.pool.Query(ctx, `SELECT e.id,e.group_id::text,COALESCE(e.rule_id::text,''),e.telegram_update_id,
		COALESCE(e.telegram_message_id,0),COALESCE(e.telegram_user_id,0),e.action,e.status,e.detail,e.created_at
		FROM telegram_group_rule_events e JOIN telegram_groups g ON g.id=e.group_id AND g.tenant_id=e.tenant_id
		WHERE e.tenant_id=$1 AND e.group_id=$2 ORDER BY e.created_at DESC,e.id DESC LIMIT $3`,
		tenantID, groupID, limit)
	if err != nil { return nil, fmt.Errorf("list group rule events: %w", err) }
	defer rows.Close()
	result := make([]GroupRuleEvent, 0)
	for rows.Next() {
		var item GroupRuleEvent
		if err := rows.Scan(&item.ID,&item.GroupID,&item.RuleID,&item.TelegramUpdateID,&item.TelegramMessageID,
			&item.TelegramUserID,&item.Action,&item.Status,&item.Detail,&item.CreatedAt); err != nil { return nil, err }
		result = append(result,item)
	}
	return result, rows.Err()
}

func validRequiredChannel(value string) bool {
	if strings.HasPrefix(value, "@") {
		name := value[1:]
		if len(name) < 5 || len(name) > 32 { return false }
		for _, char := range name {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') { return false }
		}
		return true
	}
	if !strings.HasPrefix(value, "-100") || len(value) < 10 { return false }
	for _, char := range value[4:] { if char < '0' || char > '9' { return false } }
	return true
}

type GroupMemberListEntry struct {
	TelegramUserID int64 `json:"telegram_user_id"`
	ListType string `json:"list_type"`
	Reason string `json:"reason"`
	ActorTelegramUserID int64 `json:"actor_telegram_user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type GroupMemberWarning struct {
	ID int64 `json:"id"`
	TelegramUserID int64 `json:"telegram_user_id"`
	ActorTelegramUserID int64 `json:"actor_telegram_user_id"`
	Reason string `json:"reason"`
	Source string `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

func validGroupMemberListType(value string) bool {
	return value == "blacklist" || value == "whitelist" || value == "exempt"
}

func (s *Service) ListGroupMemberLists(ctx context.Context, tenantID, groupID string) ([]GroupMemberListEntry, error) {
	rows, err := s.pool.Query(ctx, `SELECT l.telegram_user_id,l.list_type,l.reason,l.actor_telegram_user_id,l.created_at
		FROM telegram_group_member_lists l JOIN telegram_groups g ON g.id=l.group_id AND g.tenant_id=l.tenant_id
		WHERE l.tenant_id=$1 AND l.group_id=$2 ORDER BY l.created_at DESC LIMIT 500`, tenantID, groupID)
	if err != nil { return nil, fmt.Errorf("list group member lists: %w", err) }
	defer rows.Close()
	items := make([]GroupMemberListEntry, 0)
	for rows.Next() {
		var item GroupMemberListEntry
		if err := rows.Scan(&item.TelegramUserID,&item.ListType,&item.Reason,&item.ActorTelegramUserID,&item.CreatedAt); err != nil { return nil, err }
		items = append(items,item)
	}
	return items, rows.Err()
}

func (s *Service) UpsertGroupMemberList(ctx context.Context, tenantID, groupID string, userID, actorID int64, listType, reason string) error {
	listType = strings.TrimSpace(listType)
	reason = strings.TrimSpace(reason)
	if userID <= 0 || actorID < 0 || !validGroupMemberListType(listType) || len([]rune(reason)) > 500 { return ErrInvalidGroupRule }
	_, err := s.pool.Exec(ctx, `INSERT INTO telegram_group_member_lists(tenant_id,group_id,telegram_user_id,list_type,reason,actor_telegram_user_id)
		SELECT $1,g.id,$3,$4,$5,$6 FROM telegram_groups g WHERE g.tenant_id=$1 AND g.id=$2
		ON CONFLICT(tenant_id,group_id,telegram_user_id,list_type) DO UPDATE SET reason=EXCLUDED.reason,actor_telegram_user_id=EXCLUDED.actor_telegram_user_id,created_at=now()`,
		tenantID,groupID,userID,listType,reason,nullableTelegramActor(actorID))
	if err != nil { return fmt.Errorf("save group member list entry: %w", err) }
	return nil
}

func (s *Service) DeleteGroupMemberList(ctx context.Context, tenantID, groupID string, userID int64, listType string) error {
	if userID <= 0 || !validGroupMemberListType(listType) { return ErrInvalidGroupRule }
	tag, err := s.pool.Exec(ctx, `DELETE FROM telegram_group_member_lists l USING telegram_groups g
		WHERE l.tenant_id=$1 AND l.group_id=$2 AND l.telegram_user_id=$3 AND l.list_type=$4
		AND g.id=l.group_id AND g.tenant_id=l.tenant_id`,tenantID,groupID,userID,listType)
	if err != nil { return fmt.Errorf("delete group member list entry: %w", err) }
	if tag.RowsAffected()==0 { return ErrNotFound }
	return nil
}

func (s *Service) AddGroupMemberWarning(ctx context.Context, tenantID, groupID string, userID, actorID int64, reason, source string) (GroupMemberWarning,error) {
	reason = strings.TrimSpace(reason)
	if userID <= 0 || actorID < 0 || reason == "" || len([]rune(reason)) > 500 || (source != "manual" && source != "automatic") { return GroupMemberWarning{},ErrInvalidGroupRule }
	var item GroupMemberWarning
	err := s.pool.QueryRow(ctx, `INSERT INTO telegram_group_member_warnings(tenant_id,group_id,telegram_user_id,actor_telegram_user_id,reason,source)
		SELECT $1,g.id,$3,$4,$5,$6 FROM telegram_groups g WHERE g.tenant_id=$1 AND g.id=$2
		RETURNING id,telegram_user_id,actor_telegram_user_id,reason,source,created_at`,tenantID,groupID,userID,nullableTelegramActor(actorID),reason,source).
		Scan(&item.ID,&item.TelegramUserID,&item.ActorTelegramUserID,&item.Reason,&item.Source,&item.CreatedAt)
	if errors.Is(err,pgx.ErrNoRows) { return GroupMemberWarning{},ErrNotFound }
	if err != nil { return GroupMemberWarning{},fmt.Errorf("add group member warning: %w",err) }
	return item,nil
}

func (s *Service) ListGroupMemberWarnings(ctx context.Context, tenantID, groupID string, userID int64) ([]GroupMemberWarning,error) {
	if userID <= 0 { return nil,ErrInvalidGroupRule }
	rows,err:=s.pool.Query(ctx,`SELECT w.id,w.telegram_user_id,w.actor_telegram_user_id,w.reason,w.source,w.created_at
		FROM telegram_group_member_warnings w JOIN telegram_groups g ON g.id=w.group_id AND g.tenant_id=w.tenant_id
		WHERE w.tenant_id=$1 AND w.group_id=$2 AND w.telegram_user_id=$3 ORDER BY w.created_at DESC LIMIT 100`,tenantID,groupID,userID)
	if err!=nil{return nil,fmt.Errorf("list group member warnings: %w",err)}
	defer rows.Close()
	items:=make([]GroupMemberWarning,0)
	for rows.Next(){var item GroupMemberWarning;if err:=rows.Scan(&item.ID,&item.TelegramUserID,&item.ActorTelegramUserID,&item.Reason,&item.Source,&item.CreatedAt);err!=nil{return nil,err};items=append(items,item)}
	return items,rows.Err()
}


func (s *Service) ListGroupMemberWarningsPage(ctx context.Context, tenantID, groupID string, userID int64, limit, offset int) ([]GroupMemberWarning, error) {
	if userID <= 0 { return nil, ErrInvalidGroupRule }
	if limit < 1 || limit > 100 { limit = 20 }
	if offset < 0 { offset = 0 }
	rows, err := s.pool.Query(ctx, `SELECT w.id,w.telegram_user_id,w.actor_telegram_user_id,w.reason,w.source,w.created_at
		FROM telegram_group_member_warnings w JOIN telegram_groups g ON g.id=w.group_id AND g.tenant_id=w.tenant_id
		WHERE w.tenant_id=$1 AND w.group_id=$2 AND w.telegram_user_id=$3
		ORDER BY w.created_at DESC,w.id DESC LIMIT $4 OFFSET $5`, tenantID, groupID, userID, limit, offset)
	if err != nil { return nil, fmt.Errorf("list paginated group member warnings: %w", err) }
	defer rows.Close()
	items := make([]GroupMemberWarning, 0)
	for rows.Next() {
		var item GroupMemberWarning
		if err := rows.Scan(&item.ID,&item.TelegramUserID,&item.ActorTelegramUserID,&item.Reason,&item.Source,&item.CreatedAt); err != nil { return nil, err }
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) CountGroupMemberWarnings(ctx context.Context, tenantID, groupID string, userID int64) (int64, error) {
	if userID <= 0 { return 0, ErrInvalidGroupRule }
	var count int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM telegram_group_member_warnings w
		JOIN telegram_groups g ON g.id=w.group_id AND g.tenant_id=w.tenant_id
		WHERE w.tenant_id=$1 AND w.group_id=$2 AND w.telegram_user_id=$3`, tenantID, groupID, userID).Scan(&count)
	if err != nil { return 0, fmt.Errorf("count group member warnings: %w", err) }
	return count, nil
}

func (s *Service) ClearGroupMemberWarnings(ctx context.Context, tenantID, groupID string, userID int64) (int64, error) {
	if userID <= 0 { return 0, ErrInvalidGroupRule }
	tag, err := s.pool.Exec(ctx, `DELETE FROM telegram_group_member_warnings w USING telegram_groups g
		WHERE w.tenant_id=$1 AND w.group_id=$2 AND w.telegram_user_id=$3
		AND g.id=w.group_id AND g.tenant_id=w.tenant_id`, tenantID, groupID, userID)
	if err != nil { return 0, fmt.Errorf("clear group member warnings: %w", err) }
	return tag.RowsAffected(), nil
}

func nullableTelegramActor(id int64) any {
	if id <= 0 { return nil }
	return id
}

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type AgentLevel struct {
	ID string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	PriceUSDT int64 `json:"price_usdt"`
	SortOrder int32 `json:"sort_order"`
	Benefits map[string]any `json:"benefits"`
	Enabled bool `json:"enabled"`
}

type AgentAccount struct {
	IsAgent bool `json:"is_agent"`
	Status string `json:"status,omitempty"`
	CurrentLevel *AgentLevel `json:"current_level"`
	ParentUserID *string `json:"parent_user_id"`
	WalletBalanceUSDT int64 `json:"wallet_balance_usdt"`
	Levels []AgentLevel `json:"levels"`
}

type WalletLedgerEntry struct {
	ID int64 `json:"id"`
	EntryType string `json:"entry_type"`
	AmountUSDT int64 `json:"amount_usdt"`
	BalanceAfterUSDT int64 `json:"balance_after_usdt"`
	ReferenceType string `json:"reference_type"`
	ReferenceID string `json:"reference_id"`
	Description string `json:"description"`
	CreatedAt string `json:"created_at"`
}

func (s *Service) GetAgentAccount(ctx context.Context, userID string) (AgentAccount, error) {
	var result AgentAccount
	result.Levels = make([]AgentLevel, 0)
	if err := s.pool.QueryRow(ctx, `SELECT balance_usdt FROM wallet_accounts WHERE user_id=$1`, userID).Scan(&result.WalletBalanceUSDT); err != nil {
		return AgentAccount{}, fmt.Errorf("load wallet balance: %w", err)
	}
	var current AgentLevel
	var status string
	var parentID *string
	err := s.pool.QueryRow(ctx, `SELECT l.id::text,l.code,l.name,l.price_usdt,l.sort_order,l.benefits,l.enabled,p.status,p.parent_user_id::text
		FROM agent_profiles p JOIN agent_levels l ON l.id=p.level_id WHERE p.user_id=$1`, userID).
		Scan(&current.ID,&current.Code,&current.Name,&current.PriceUSDT,&current.SortOrder,&current.Benefits,&current.Enabled,&status,&parentID)
	if err == nil {
		result.IsAgent = true
		result.Status = status
		result.CurrentLevel = &current
		result.ParentUserID = parentID
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AgentAccount{}, fmt.Errorf("load agent profile: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,code,name,price_usdt,sort_order,benefits,enabled FROM agent_levels WHERE enabled=true ORDER BY sort_order ASC,price_usdt ASC,id ASC`)
	if err != nil { return AgentAccount{}, fmt.Errorf("list agent levels: %w", err) }
	defer rows.Close()
	for rows.Next() {
		var level AgentLevel
		if err := rows.Scan(&level.ID,&level.Code,&level.Name,&level.PriceUSDT,&level.SortOrder,&level.Benefits,&level.Enabled); err != nil { return AgentAccount{}, fmt.Errorf("scan agent level: %w", err) }
		result.Levels = append(result.Levels, level)
	}
	if err := rows.Err(); err != nil { return AgentAccount{}, fmt.Errorf("iterate agent levels: %w", err) }
	return result,nil
}

func (s *Service) ListWalletLedger(ctx context.Context, userID string, limit, offset int) ([]WalletLedgerEntry, int64, error) {
	var total int64
	if err := s.pool.QueryRow(ctx,`SELECT count(*) FROM wallet_ledger WHERE user_id=$1`,userID).Scan(&total); err != nil {
		return nil,0,fmt.Errorf("count wallet ledger: %w",err)
	}
	rows,err:=s.pool.Query(ctx,`SELECT id,entry_type,amount_usdt,balance_after_usdt,reference_type,reference_id,description,created_at
		FROM wallet_ledger WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`,userID,limit,offset)
	if err!=nil { return nil,0,fmt.Errorf("list wallet ledger: %w",err) }
	defer rows.Close()
	items:=make([]WalletLedgerEntry,0)
	for rows.Next() {
		var item WalletLedgerEntry
		var createdAt time.Time
		if err:=rows.Scan(&item.ID,&item.EntryType,&item.AmountUSDT,&item.BalanceAfterUSDT,&item.ReferenceType,&item.ReferenceID,&item.Description,&createdAt);err!=nil {
			return nil,0,fmt.Errorf("scan wallet ledger: %w",err)
		}
		item.CreatedAt=createdAt.UTC().Format(time.RFC3339Nano)
		items=append(items,item)
	}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate wallet ledger: %w",err)}
	return items,total,nil
}


var (
	ErrInsufficientWalletBalance = errors.New("insufficient wallet balance")
	ErrAgentLevelNotUpgradeable = errors.New("target agent level is not an upgrade")
	ErrAgentProfileSuspended = errors.New("agent profile is suspended")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
)

type AgentUpgradeOrder struct {
	ID string `json:"id"`
	OrderNo string `json:"order_no"`
	FromLevelID *string `json:"from_level_id"`
	TargetLevelID string `json:"target_level_id"`
	AmountUSDT int64 `json:"amount_usdt"`
	PaymentMethod string `json:"payment_method"`
	Status string `json:"status"`
	PaidAt *time.Time `json:"paid_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) UpgradeAgentWithWallet(ctx context.Context, userID, targetLevelID, idempotencyKey string) (AgentUpgradeOrder, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" || len(idempotencyKey) > 120 {
		return AgentUpgradeOrder{}, ErrInvalidIdempotencyKey
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil { return AgentUpgradeOrder{}, fmt.Errorf("begin agent upgrade: %w", err) }
	defer tx.Rollback(ctx)

	var balance int64
	if err := tx.QueryRow(ctx, `SELECT balance_usdt FROM wallet_accounts WHERE user_id=$1 FOR UPDATE`, userID).Scan(&balance); err != nil {
		return AgentUpgradeOrder{}, fmt.Errorf("lock user wallet: %w", err)
	}
	var existing AgentUpgradeOrder
	err = tx.QueryRow(ctx, `SELECT id::text,order_no,from_level_id::text,target_level_id::text,amount_usdt,payment_method,status,paid_at,created_at
		FROM agent_upgrade_orders WHERE user_id=$1 AND idempotency_key=$2`, userID, idempotencyKey).
		Scan(&existing.ID,&existing.OrderNo,&existing.FromLevelID,&existing.TargetLevelID,&existing.AmountUSDT,&existing.PaymentMethod,&existing.Status,&existing.PaidAt,&existing.CreatedAt)
	if err == nil {
		if err := tx.Commit(ctx); err != nil { return AgentUpgradeOrder{}, fmt.Errorf("commit idempotent agent upgrade: %w", err) }
		return existing,nil
	}
	if !errors.Is(err,pgx.ErrNoRows) { return AgentUpgradeOrder{}, fmt.Errorf("check idempotent agent upgrade: %w",err) }

	var targetOrder int32
	var amount int64
	var enabled bool
	if err:=tx.QueryRow(ctx,`SELECT sort_order,price_usdt,enabled FROM agent_levels WHERE id=$1 FOR SHARE`,targetLevelID).Scan(&targetOrder,&amount,&enabled);err!=nil {
		if errors.Is(err,pgx.ErrNoRows) { return AgentUpgradeOrder{},ErrAgentLevelNotUpgradeable }
		return AgentUpgradeOrder{},fmt.Errorf("load target agent level: %w",err)
	}
	if !enabled { return AgentUpgradeOrder{},ErrAgentLevelNotUpgradeable }

	var fromLevelID *string
	currentOrder:=int32(-1)
	var profileStatus string
	err=tx.QueryRow(ctx,`SELECT p.level_id::text,l.sort_order,p.status FROM agent_profiles p JOIN agent_levels l ON l.id=p.level_id WHERE p.user_id=$1 FOR UPDATE OF p`,userID).
		Scan(&fromLevelID,&currentOrder,&profileStatus)
	if err==nil {
		if profileStatus!="active" { return AgentUpgradeOrder{},ErrAgentProfileSuspended }
	} else if !errors.Is(err,pgx.ErrNoRows) {
		return AgentUpgradeOrder{},fmt.Errorf("lock current agent profile: %w",err)
	}
	if targetOrder<=currentOrder { return AgentUpgradeOrder{},ErrAgentLevelNotUpgradeable }
	if balance<amount { return AgentUpgradeOrder{},ErrInsufficientWalletBalance }

	var order AgentUpgradeOrder
	if amount>0 {
		if err:=tx.QueryRow(ctx,`UPDATE wallet_accounts SET balance_usdt=balance_usdt-$2,updated_at=now()
			WHERE user_id=$1 AND balance_usdt >= $2 RETURNING balance_usdt`,userID,amount).Scan(&balance);err!=nil {
			return AgentUpgradeOrder{},ErrInsufficientWalletBalance
		}
	}
	err=tx.QueryRow(ctx,`INSERT INTO agent_upgrade_orders(user_id,from_level_id,target_level_id,amount_usdt,payment_method,status,idempotency_key,paid_at)
		VALUES($1,$2,$3,$4,'wallet','paid',$5,now())
		RETURNING id::text,order_no,from_level_id::text,target_level_id::text,amount_usdt,payment_method,status,paid_at,created_at`,
		userID,fromLevelID,targetLevelID,amount,idempotencyKey).
		Scan(&order.ID,&order.OrderNo,&order.FromLevelID,&order.TargetLevelID,&order.AmountUSDT,&order.PaymentMethod,&order.Status,&order.PaidAt,&order.CreatedAt)
	if err!=nil { return AgentUpgradeOrder{},fmt.Errorf("create agent upgrade order: %w",err) }
	if amount>0 {
		ledgerKey:="agent-upgrade:"+order.ID
		if _,err:=tx.Exec(ctx,`INSERT INTO wallet_ledger(user_id,entry_type,amount_usdt,balance_after_usdt,reference_type,reference_id,idempotency_key,description)
			VALUES($1,'agent_upgrade',$2,$3,'agent_upgrade',$4,$5,$6)`,userID,-amount,balance,order.ID,ledgerKey,"代理等级升级扣款");err!=nil {
			return AgentUpgradeOrder{},fmt.Errorf("write agent upgrade ledger: %w",err)
		}
	}
	if _,err:=tx.Exec(ctx,`INSERT INTO agent_profiles(user_id,level_id,status)
		VALUES($1,$2,'active')
		ON CONFLICT(user_id) DO UPDATE SET level_id=EXCLUDED.level_id,status='active',updated_at=now()`,userID,targetLevelID);err!=nil {
		return AgentUpgradeOrder{},fmt.Errorf("activate agent profile: %w",err)
	}
	if _,err:=tx.Exec(ctx,`UPDATE users SET role='agent',updated_at=now() WHERE id=$1`,userID);err!=nil {
		return AgentUpgradeOrder{},fmt.Errorf("update user agent role: %w",err)
	}
	if err:=tx.Commit(ctx);err!=nil { return AgentUpgradeOrder{},fmt.Errorf("commit agent upgrade: %w",err) }
	return order,nil
}


var ErrInvalidAgentLevel = errors.New("invalid agent level")

func validateAgentLevel(code,name string,price int64,sortOrder int32) error {
	if len(code)<2 || len(code)>40 || len(name)==0 || len(name)>80 || price<0 || sortOrder<0 {
		return ErrInvalidAgentLevel
	}
	for _,r:=range code {
		if !(r>='a'&&r<='z') && !(r>='0'&&r<='9') && r!='_' && r!='-' { return ErrInvalidAgentLevel }
	}
	return nil
}

func (s *Service) ListAdminAgentLevels(ctx context.Context) ([]AgentLevel,error) {
	rows,err:=s.pool.Query(ctx,`SELECT id::text,code,name,price_usdt,sort_order,benefits,enabled FROM agent_levels ORDER BY sort_order,price_usdt,id`)
	if err!=nil{return nil,fmt.Errorf("list admin agent levels: %w",err)}
	defer rows.Close()
	items:=make([]AgentLevel,0)
	for rows.Next(){var item AgentLevel;if err:=rows.Scan(&item.ID,&item.Code,&item.Name,&item.PriceUSDT,&item.SortOrder,&item.Benefits,&item.Enabled);err!=nil{return nil,fmt.Errorf("scan admin agent level: %w",err)};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate admin agent levels: %w",err)}
	return items,nil
}

func (s *Service) CreateAdminAgentLevel(ctx context.Context,actorID,code,name string,price int64,sortOrder int32,benefits map[string]any)(AgentLevel,error){
	code=strings.TrimSpace(strings.ToLower(code));name=strings.TrimSpace(name)
	if err:=validateAgentLevel(code,name,price,sortOrder);err!=nil{return AgentLevel{},err}
	if benefits==nil{benefits=map[string]any{}}
	raw,err:=json.Marshal(benefits);if err!=nil{return AgentLevel{},ErrInvalidAgentLevel}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{});if err!=nil{return AgentLevel{},fmt.Errorf("begin create agent level: %w",err)}
	defer tx.Rollback(ctx)
	var item AgentLevel
	err=tx.QueryRow(ctx,`INSERT INTO agent_levels(code,name,price_usdt,sort_order,benefits) VALUES($1,$2,$3,$4,$5::jsonb)
		RETURNING id::text,code,name,price_usdt,sort_order,benefits,enabled`,code,name,price,sortOrder,string(raw)).
		Scan(&item.ID,&item.Code,&item.Name,&item.PriceUSDT,&item.SortOrder,&item.Benefits,&item.Enabled)
	if err!=nil{return AgentLevel{},fmt.Errorf("create agent level: %w",err)}
	var tenantID string
	if err:=tx.QueryRow(ctx,`SELECT tenant_id::text FROM users WHERE id=$1`,actorID).Scan(&tenantID);err!=nil{return AgentLevel{},fmt.Errorf("load agent level actor tenant: %w",err)}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,'agent.level.create','agent_level',$3)`,tenantID,actorID,item.ID);err!=nil{return AgentLevel{},fmt.Errorf("audit agent level create: %w",err)}
	if err:=tx.Commit(ctx);err!=nil{return AgentLevel{},fmt.Errorf("commit agent level create: %w",err)}
	return item,nil
}

func (s *Service) UpdateAdminAgentLevel(ctx context.Context,actorID,levelID,name string,price int64,sortOrder int32,benefits map[string]any,enabled bool)(AgentLevel,error){
	name=strings.TrimSpace(name)
	if len(name)==0 || len(name)>80 || price<0 || sortOrder<0{return AgentLevel{},ErrInvalidAgentLevel}
	if benefits==nil{benefits=map[string]any{}}
	raw,err:=json.Marshal(benefits);if err!=nil{return AgentLevel{},ErrInvalidAgentLevel}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{});if err!=nil{return AgentLevel{},fmt.Errorf("begin update agent level: %w",err)}
	defer tx.Rollback(ctx)
	var item AgentLevel
	err=tx.QueryRow(ctx,`UPDATE agent_levels SET name=$2,price_usdt=$3,sort_order=$4,benefits=$5::jsonb,enabled=$6,updated_at=now()
		WHERE id=$1 RETURNING id::text,code,name,price_usdt,sort_order,benefits,enabled`,levelID,name,price,sortOrder,string(raw),enabled).
		Scan(&item.ID,&item.Code,&item.Name,&item.PriceUSDT,&item.SortOrder,&item.Benefits,&item.Enabled)
	if errors.Is(err,pgx.ErrNoRows){return AgentLevel{},ErrAgentLevelNotUpgradeable}
	if err!=nil{return AgentLevel{},fmt.Errorf("update agent level: %w",err)}
	var tenantID string
	if err:=tx.QueryRow(ctx,`SELECT tenant_id::text FROM users WHERE id=$1`,actorID).Scan(&tenantID);err!=nil{return AgentLevel{},fmt.Errorf("load agent level actor tenant: %w",err)}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,'agent.level.update','agent_level',$3)`,tenantID,actorID,item.ID);err!=nil{return AgentLevel{},fmt.Errorf("audit agent level update: %w",err)}
	if err:=tx.Commit(ctx);err!=nil{return AgentLevel{},fmt.Errorf("commit agent level update: %w",err)}
	return item,nil
}

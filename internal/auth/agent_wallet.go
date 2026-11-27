package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"fmt"
)

type AgentLevel struct {
	ID string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	PriceUSDT int64 `json:"price_usdt"`
	SortOrder int32 `json:"sort_order"`
	Benefits map[string]any `json:"benefits"`
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
	err := s.pool.QueryRow(ctx, `SELECT l.id::text,l.code,l.name,l.price_usdt,l.sort_order,l.benefits,p.status,p.parent_user_id::text
		FROM agent_profiles p JOIN agent_levels l ON l.id=p.level_id WHERE p.user_id=$1`, userID).
		Scan(&current.ID,&current.Code,&current.Name,&current.PriceUSDT,&current.SortOrder,&current.Benefits,&status,&parentID)
	if err == nil {
		result.IsAgent = true
		result.Status = status
		result.CurrentLevel = &current
		result.ParentUserID = parentID
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return AgentAccount{}, fmt.Errorf("load agent profile: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text,code,name,price_usdt,sort_order,benefits FROM agent_levels WHERE enabled=true ORDER BY sort_order ASC,price_usdt ASC,id ASC`)
	if err != nil { return AgentAccount{}, fmt.Errorf("list agent levels: %w", err) }
	defer rows.Close()
	for rows.Next() {
		var level AgentLevel
		if err := rows.Scan(&level.ID,&level.Code,&level.Name,&level.PriceUSDT,&level.SortOrder,&level.Benefits); err != nil { return AgentAccount{}, fmt.Errorf("scan agent level: %w", err) }
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

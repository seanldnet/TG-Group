package auth

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AdminFinanceSummary struct {
	WalletCount int64 `json:"wallet_count"`
	TotalWalletBalanceUSDT int64 `json:"total_wallet_balance_usdt"`
	LedgerEntryCount int64 `json:"ledger_entry_count"`
	CreditTotalUSDT int64 `json:"credit_total_usdt"`
	DebitTotalUSDT int64 `json:"debit_total_usdt"`
}

type AdminWalletLedgerEntry struct {
	ID int64 `json:"id"`
	UserID string `json:"user_id"`
	UserEmail string `json:"user_email"`
	DisplayName string `json:"display_name"`
	EntryType string `json:"entry_type"`
	AmountUSDT int64 `json:"amount_usdt"`
	BalanceAfterUSDT int64 `json:"balance_after_usdt"`
	ReferenceType string `json:"reference_type"`
	ReferenceID string `json:"reference_id"`
	Description string `json:"description"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Service) GetAdminFinanceSummary(ctx context.Context)(AdminFinanceSummary,error) {
	var result AdminFinanceSummary
	err:=s.pool.QueryRow(ctx,`SELECT
		(SELECT count(*) FROM wallet_accounts),
		COALESCE((SELECT sum(balance_usdt) FROM wallet_accounts),0),
		(SELECT count(*) FROM wallet_ledger),
		COALESCE((SELECT sum(amount_usdt) FROM wallet_ledger WHERE amount_usdt>0),0),
		COALESCE((SELECT sum(-amount_usdt) FROM wallet_ledger WHERE amount_usdt<0),0)`).
		Scan(&result.WalletCount,&result.TotalWalletBalanceUSDT,&result.LedgerEntryCount,&result.CreditTotalUSDT,&result.DebitTotalUSDT)
	if err!=nil{return AdminFinanceSummary{},fmt.Errorf("load platform finance summary: %w",err)}
	return result,nil
}

func (s *Service) ListAdminWalletLedger(ctx context.Context,entryType,query string,limit,offset int)([]AdminWalletLedgerEntry,int64,error) {
	entryType=strings.TrimSpace(entryType);query=strings.TrimSpace(query)
	switch entryType {
	case "","deposit","agent_upgrade","order_payment","refund","commission","adjustment":
	default:return nil,0,ErrInvalidInput
	}
	pattern:="%"+query+"%"
	var total int64
	err:=s.pool.QueryRow(ctx,`SELECT count(*) FROM wallet_ledger l JOIN users u ON u.id=l.user_id
		WHERE ($1='' OR l.entry_type=$1) AND ($2='' OR u.email ILIKE $3 OR u.display_name ILIKE $3 OR l.reference_id ILIKE $3 OR l.description ILIKE $3)`,
		entryType,query,pattern).Scan(&total)
	if err!=nil{return nil,0,fmt.Errorf("count platform wallet ledger: %w",err)}
	rows,err:=s.pool.Query(ctx,`SELECT l.id,l.user_id::text,u.email,u.display_name,l.entry_type,l.amount_usdt,l.balance_after_usdt,
		l.reference_type,l.reference_id,l.description,l.created_at
		FROM wallet_ledger l JOIN users u ON u.id=l.user_id
		WHERE ($1='' OR l.entry_type=$1) AND ($2='' OR u.email ILIKE $3 OR u.display_name ILIKE $3 OR l.reference_id ILIKE $3 OR l.description ILIKE $3)
		ORDER BY l.created_at DESC,l.id DESC LIMIT $4 OFFSET $5`,entryType,query,pattern,limit,offset)
	if err!=nil{return nil,0,fmt.Errorf("list platform wallet ledger: %w",err)}
	defer rows.Close()
	items:=make([]AdminWalletLedgerEntry,0)
	for rows.Next(){var item AdminWalletLedgerEntry;if err:=rows.Scan(&item.ID,&item.UserID,&item.UserEmail,&item.DisplayName,&item.EntryType,&item.AmountUSDT,&item.BalanceAfterUSDT,&item.ReferenceType,&item.ReferenceID,&item.Description,&item.CreatedAt);err!=nil{return nil,0,fmt.Errorf("scan platform wallet ledger: %w",err)};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate platform wallet ledger: %w",err)}
	return items,total,nil
}

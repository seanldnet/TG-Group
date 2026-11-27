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

var (
	ErrInsufficientPlanBalance = errors.New("insufficient wallet balance for plan")
	ErrPlanUnavailable = errors.New("plan is not available")
	ErrLicenseSuspended = errors.New("license is suspended")
	ErrOrderIdempotencyConflict = errors.New("idempotency key already used for another plan")
	ErrInvalidOrderIdempotencyKey = errors.New("invalid order idempotency key")
)

type AccountOrder struct {
	ID string `json:"id"`
	OrderNo string `json:"order_no"`
	PlanID string `json:"plan_id"`
	PlanName string `json:"plan_name"`
	AmountUSDT int64 `json:"amount_usdt"`
	PaymentMethod string `json:"payment_method"`
	Status string `json:"status"`
	PaidAt *time.Time `json:"paid_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AccountLicense struct {
	ID string `json:"id"`
	PlanID string `json:"plan_id"`
	PlanName string `json:"plan_name"`
	Status string `json:"status"`
	StartsAt time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	BotLimit int32 `json:"bot_limit"`
	GroupLimit int32 `json:"group_limit"`
	Features map[string]any `json:"features"`
}

func (s *Service) PurchasePlanWithWallet(ctx context.Context,userID,planID,idempotencyKey string)(AccountOrder,error) {
	idempotencyKey=strings.TrimSpace(idempotencyKey)
	if idempotencyKey=="" || len(idempotencyKey)>120 { return AccountOrder{},ErrInvalidOrderIdempotencyKey }
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{})
	if err!=nil{return AccountOrder{},fmt.Errorf("begin plan purchase: %w",err)}
	defer tx.Rollback(ctx)

	if _,err:=tx.Exec(ctx,`INSERT INTO wallet_accounts(user_id) VALUES($1) ON CONFLICT(user_id) DO NOTHING`,userID);err!=nil{return AccountOrder{},fmt.Errorf("ensure wallet account: %w",err)}
	var balance int64
	if err:=tx.QueryRow(ctx,`SELECT balance_usdt FROM wallet_accounts WHERE user_id=$1 FOR UPDATE`,userID).Scan(&balance);err!=nil{return AccountOrder{},fmt.Errorf("lock plan buyer wallet: %w",err)}

	var existing AccountOrder
	err=tx.QueryRow(ctx,`SELECT id::text,order_no,plan_id::text,plan_name,amount_usdt,payment_method,status,paid_at,created_at
		FROM orders WHERE user_id=$1 AND idempotency_key=$2`,userID,idempotencyKey).
		Scan(&existing.ID,&existing.OrderNo,&existing.PlanID,&existing.PlanName,&existing.AmountUSDT,&existing.PaymentMethod,&existing.Status,&existing.PaidAt,&existing.CreatedAt)
	if err==nil {
		if existing.PlanID!=planID{return AccountOrder{},ErrOrderIdempotencyConflict}
		if err:=tx.Commit(ctx);err!=nil{return AccountOrder{},fmt.Errorf("commit idempotent plan purchase: %w",err)}
		return existing,nil
	}
	if !errors.Is(err,pgx.ErrNoRows){return AccountOrder{},fmt.Errorf("check plan purchase idempotency: %w",err)}

	var plan Plan
	err=tx.QueryRow(ctx,`SELECT id::text,code,name,description,price_usdt,duration_days,bot_limit,group_limit,features,sort_order,enabled
		FROM plans WHERE id=$1 FOR SHARE`,planID).
		Scan(&plan.ID,&plan.Code,&plan.Name,&plan.Description,&plan.PriceUSDT,&plan.DurationDays,&plan.BotLimit,&plan.GroupLimit,&plan.Features,&plan.SortOrder,&plan.Enabled)
	if errors.Is(err,pgx.ErrNoRows){return AccountOrder{},ErrPlanUnavailable}
	if err!=nil{return AccountOrder{},fmt.Errorf("load plan for purchase: %w",err)}
	if !plan.Enabled{return AccountOrder{},ErrPlanUnavailable}
	if balance<plan.PriceUSDT{return AccountOrder{},ErrInsufficientPlanBalance}

	var licenseStatus string
	err=tx.QueryRow(ctx,`SELECT status FROM licenses WHERE user_id=$1 AND plan_id=$2 FOR UPDATE`,userID,planID).Scan(&licenseStatus)
	if err==nil && licenseStatus=="suspended"{return AccountOrder{},ErrLicenseSuspended}
	if err!=nil && !errors.Is(err,pgx.ErrNoRows){return AccountOrder{},fmt.Errorf("lock existing license: %w",err)}

	var order AccountOrder
	err=tx.QueryRow(ctx,`INSERT INTO orders(user_id,plan_id,plan_name,amount_usdt,payment_method,status,idempotency_key,paid_at)
		VALUES($1,$2,$3,$4,'wallet','paid',$5,now())
		RETURNING id::text,order_no,plan_id::text,plan_name,amount_usdt,payment_method,status,paid_at,created_at`,
		userID,planID,plan.Name,plan.PriceUSDT,idempotencyKey).
		Scan(&order.ID,&order.OrderNo,&order.PlanID,&order.PlanName,&order.AmountUSDT,&order.PaymentMethod,&order.Status,&order.PaidAt,&order.CreatedAt)
	if err!=nil{return AccountOrder{},fmt.Errorf("create plan order: %w",err)}

	if plan.PriceUSDT>0 {
		if err:=tx.QueryRow(ctx,`UPDATE wallet_accounts SET balance_usdt=balance_usdt-$2,updated_at=now()
			WHERE user_id=$1 AND balance_usdt >= $2 RETURNING balance_usdt`,userID,plan.PriceUSDT).Scan(&balance);err!=nil{return AccountOrder{},ErrInsufficientPlanBalance}
		ledgerKey:="plan-order:"+order.ID
		if _,err:=tx.Exec(ctx,`INSERT INTO wallet_ledger(user_id,entry_type,amount_usdt,balance_after_usdt,reference_type,reference_id,idempotency_key,description)
			VALUES($1,'order_payment',$2,$3,'order',$4,$5,$6)`,userID,-plan.PriceUSDT,balance,order.ID,ledgerKey,"套餐购买扣款："+plan.Name);err!=nil{return AccountOrder{},fmt.Errorf("write plan purchase ledger: %w",err)}
	}

	features:=plan.Features
	if features==nil{features=map[string]any{}}
	rawFeatures,err:=json.Marshal(features);if err!=nil{return AccountOrder{},fmt.Errorf("encode plan feature snapshot: %w",err)}
	_,err=tx.Exec(ctx,`INSERT INTO licenses(user_id,plan_id,last_order_id,plan_name,status,starts_at,expires_at,bot_limit,group_limit,features)
		VALUES($1,$2,$3,$4,'active',now(),now()+($5 * interval '1 day'),$6,$7,$8::jsonb)
		ON CONFLICT(user_id,plan_id) DO UPDATE SET
			last_order_id=EXCLUDED.last_order_id,plan_name=EXCLUDED.plan_name,status='active',
			expires_at=GREATEST(licenses.expires_at,now())+($5 * interval '1 day'),
			bot_limit=EXCLUDED.bot_limit,group_limit=EXCLUDED.group_limit,features=EXCLUDED.features,updated_at=now()`,
		userID,planID,order.ID,plan.Name,plan.DurationDays,plan.BotLimit,plan.GroupLimit,string(rawFeatures))
	if err!=nil{return AccountOrder{},fmt.Errorf("activate plan license: %w",err)}

	var tenantID string
	if err:=tx.QueryRow(ctx,`SELECT tenant_id::text FROM users WHERE id=$1`,userID).Scan(&tenantID);err!=nil{return AccountOrder{},fmt.Errorf("load plan buyer tenant: %w",err)}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,'order.purchase','order',$3)`,tenantID,userID,order.ID);err!=nil{return AccountOrder{},fmt.Errorf("audit plan purchase: %w",err)}
	if err:=tx.Commit(ctx);err!=nil{return AccountOrder{},fmt.Errorf("commit plan purchase: %w",err)}
	return order,nil
}

func (s *Service) ListAccountOrders(ctx context.Context,userID string,limit,offset int)([]AccountOrder,int64,error) {
	var total int64
	if err:=s.pool.QueryRow(ctx,`SELECT count(*) FROM orders WHERE user_id=$1`,userID).Scan(&total);err!=nil{return nil,0,fmt.Errorf("count account orders: %w",err)}
	rows,err:=s.pool.Query(ctx,`SELECT id::text,order_no,plan_id::text,plan_name,amount_usdt,payment_method,status,paid_at,created_at
		FROM orders WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`,userID,limit,offset)
	if err!=nil{return nil,0,fmt.Errorf("list account orders: %w",err)}
	defer rows.Close()
	items:=make([]AccountOrder,0)
	for rows.Next(){var item AccountOrder;if err:=rows.Scan(&item.ID,&item.OrderNo,&item.PlanID,&item.PlanName,&item.AmountUSDT,&item.PaymentMethod,&item.Status,&item.PaidAt,&item.CreatedAt);err!=nil{return nil,0,fmt.Errorf("scan account order: %w",err)};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate account orders: %w",err)}
	return items,total,nil
}

func (s *Service) ListAccountLicenses(ctx context.Context,userID string)([]AccountLicense,error) {
	rows,err:=s.pool.Query(ctx,`SELECT id::text,plan_id::text,plan_name,
		CASE WHEN status='active' AND expires_at<=now() THEN 'expired' ELSE status END,
		starts_at,expires_at,bot_limit,group_limit,features
		FROM licenses WHERE user_id=$1 ORDER BY expires_at DESC,id DESC`,userID)
	if err!=nil{return nil,fmt.Errorf("list account licenses: %w",err)}
	defer rows.Close()
	items:=make([]AccountLicense,0)
	for rows.Next(){var item AccountLicense;if err:=rows.Scan(&item.ID,&item.PlanID,&item.PlanName,&item.Status,&item.StartsAt,&item.ExpiresAt,&item.BotLimit,&item.GroupLimit,&item.Features);err!=nil{return nil,fmt.Errorf("scan account license: %w",err)};if item.Features==nil{item.Features=map[string]any{}};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate account licenses: %w",err)}
	return items,nil
}


var (
	ErrInvalidLicenseStatus = errors.New("invalid license status")
	ErrLicenseExpired = errors.New("license has expired")
	ErrLicenseNotFound = errors.New("license not found")
)

type AdminOrder struct {
	ID string `json:"id"`
	OrderNo string `json:"order_no"`
	UserID string `json:"user_id"`
	UserEmail string `json:"user_email"`
	DisplayName string `json:"display_name"`
	PlanName string `json:"plan_name"`
	AmountUSDT int64 `json:"amount_usdt"`
	PaymentMethod string `json:"payment_method"`
	Status string `json:"status"`
	PaidAt *time.Time `json:"paid_at"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminLicense struct {
	ID string `json:"id"`
	UserID string `json:"user_id"`
	UserEmail string `json:"user_email"`
	DisplayName string `json:"display_name"`
	PlanName string `json:"plan_name"`
	Status string `json:"status"`
	StartsAt time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	BotLimit int32 `json:"bot_limit"`
	GroupLimit int32 `json:"group_limit"`
}

func (s *Service) ListAdminOrders(ctx context.Context,status,query string,limit,offset int)([]AdminOrder,int64,error) {
	status=strings.TrimSpace(status);query=strings.TrimSpace(query)
	if status!="" && status!="pending" && status!="paid" && status!="failed" && status!="cancelled" {return nil,0,ErrInvalidInput}
	pattern:="%"+query+"%"
	var total int64
	if err:=s.pool.QueryRow(ctx,`SELECT count(*) FROM orders o JOIN users u ON u.id=o.user_id
		WHERE ($1='' OR o.status=$1) AND ($2='' OR o.order_no ILIKE $3 OR u.email ILIKE $3 OR u.display_name ILIKE $3)`,status,query,pattern).Scan(&total);err!=nil{return nil,0,fmt.Errorf("count platform orders: %w",err)}
	rows,err:=s.pool.Query(ctx,`SELECT o.id::text,o.order_no,o.user_id::text,u.email,u.display_name,o.plan_name,o.amount_usdt,o.payment_method,o.status,o.paid_at,o.created_at
		FROM orders o JOIN users u ON u.id=o.user_id
		WHERE ($1='' OR o.status=$1) AND ($2='' OR o.order_no ILIKE $3 OR u.email ILIKE $3 OR u.display_name ILIKE $3)
		ORDER BY o.created_at DESC,o.id DESC LIMIT $4 OFFSET $5`,status,query,pattern,limit,offset)
	if err!=nil{return nil,0,fmt.Errorf("list platform orders: %w",err)}
	defer rows.Close()
	items:=make([]AdminOrder,0)
	for rows.Next(){var item AdminOrder;if err:=rows.Scan(&item.ID,&item.OrderNo,&item.UserID,&item.UserEmail,&item.DisplayName,&item.PlanName,&item.AmountUSDT,&item.PaymentMethod,&item.Status,&item.PaidAt,&item.CreatedAt);err!=nil{return nil,0,fmt.Errorf("scan platform order: %w",err)};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate platform orders: %w",err)}
	return items,total,nil
}

func (s *Service) ListAdminLicenses(ctx context.Context,status,query string,limit,offset int)([]AdminLicense,int64,error) {
	status=strings.TrimSpace(status);query=strings.TrimSpace(query)
	if status!="" && status!="active" && status!="expired" && status!="suspended" {return nil,0,ErrInvalidInput}
	pattern:="%"+query+"%"
	var total int64
	if err:=s.pool.QueryRow(ctx,`SELECT count(*) FROM licenses l JOIN users u ON u.id=l.user_id
		WHERE ($1='' OR (CASE WHEN l.status='active' AND l.expires_at<=now() THEN 'expired' ELSE l.status END)=$1)
		AND ($2='' OR u.email ILIKE $3 OR u.display_name ILIKE $3 OR l.plan_name ILIKE $3)`,status,query,pattern).Scan(&total);err!=nil{return nil,0,fmt.Errorf("count platform licenses: %w",err)}
	rows,err:=s.pool.Query(ctx,`SELECT l.id::text,l.user_id::text,u.email,u.display_name,l.plan_name,
		CASE WHEN l.status='active' AND l.expires_at<=now() THEN 'expired' ELSE l.status END,
		l.starts_at,l.expires_at,l.bot_limit,l.group_limit
		FROM licenses l JOIN users u ON u.id=l.user_id
		WHERE ($1='' OR (CASE WHEN l.status='active' AND l.expires_at<=now() THEN 'expired' ELSE l.status END)=$1)
		AND ($2='' OR u.email ILIKE $3 OR u.display_name ILIKE $3 OR l.plan_name ILIKE $3)
		ORDER BY l.expires_at DESC,l.id DESC LIMIT $4 OFFSET $5`,status,query,pattern,limit,offset)
	if err!=nil{return nil,0,fmt.Errorf("list platform licenses: %w",err)}
	defer rows.Close()
	items:=make([]AdminLicense,0)
	for rows.Next(){var item AdminLicense;if err:=rows.Scan(&item.ID,&item.UserID,&item.UserEmail,&item.DisplayName,&item.PlanName,&item.Status,&item.StartsAt,&item.ExpiresAt,&item.BotLimit,&item.GroupLimit);err!=nil{return nil,0,fmt.Errorf("scan platform license: %w",err)};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate platform licenses: %w",err)}
	return items,total,nil
}

func (s *Service) SetAdminLicenseStatus(ctx context.Context,actorID,licenseID,status string)(AdminLicense,error) {
	if status!="active" && status!="suspended" {return AdminLicense{},ErrInvalidLicenseStatus}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{})
	if err!=nil{return AdminLicense{},fmt.Errorf("begin license status update: %w",err)}
	defer tx.Rollback(ctx)
	var item AdminLicense
	err=tx.QueryRow(ctx,`UPDATE licenses l SET status=$2,updated_at=now()
		FROM users u WHERE l.id=$1 AND u.id=l.user_id AND ($2<>'active' OR l.expires_at>now())
		RETURNING l.id::text,l.user_id::text,u.email,u.display_name,l.plan_name,l.status,l.starts_at,l.expires_at,l.bot_limit,l.group_limit`,
		licenseID,status).Scan(&item.ID,&item.UserID,&item.UserEmail,&item.DisplayName,&item.PlanName,&item.Status,&item.StartsAt,&item.ExpiresAt,&item.BotLimit,&item.GroupLimit)
	if errors.Is(err,pgx.ErrNoRows){
		var exists bool
		if checkErr:=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM licenses WHERE id=$1)`,licenseID).Scan(&exists);checkErr!=nil{return AdminLicense{},fmt.Errorf("check license after update: %w",checkErr)}
		if exists && status=="active" {return AdminLicense{},ErrLicenseExpired}
		return AdminLicense{},ErrLicenseNotFound
	}
	if err!=nil{return AdminLicense{},fmt.Errorf("update license status: %w",err)}
	var tenantID string
	if err:=tx.QueryRow(ctx,`SELECT tenant_id::text FROM users WHERE id=$1`,actorID).Scan(&tenantID);err!=nil{return AdminLicense{},fmt.Errorf("load license admin tenant: %w",err)}
	if _,err:=tx.Exec(ctx,`INSERT INTO audit_logs(tenant_id,actor_user_id,action,resource_type,resource_id)
		VALUES($1,$2,$3,'license',$4)`,tenantID,actorID,"license."+status,item.ID);err!=nil{return AdminLicense{},fmt.Errorf("audit license status update: %w",err)}
	if err:=tx.Commit(ctx);err!=nil{return AdminLicense{},fmt.Errorf("commit license status update: %w",err)}
	return item,nil
}

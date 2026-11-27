package payments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRechargeOrderNotFound = errors.New("recharge order not found")
	ErrRechargeOrderConflict = errors.New("recharge order state conflict")
	ErrRechargeNeedsReview = errors.New("signed payment received with a non-integer USDT amount; manual review required")
)

type RechargeOrder struct {
	ID string `json:"id"`
	OrderNo string `json:"order_no"`
	MerchantOrderID string `json:"merchant_order_id"`
	ProviderTradeID string `json:"provider_trade_id,omitempty"`
	AmountUSDT int64 `json:"amount_usdt"`
	ActualAmount string `json:"actual_amount,omitempty"`
	Token string `json:"token,omitempty"`
	Network string `json:"network,omitempty"`
	ReceiveAddress string `json:"receive_address,omitempty"`
	PaymentURL string `json:"payment_url,omitempty"`
	Status string `json:"status"`
	ProviderStatus int `json:"provider_status"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type RechargeService struct {
	pool *pgxpool.Pool
	channels *ConfigStore
	client GMPayClient
}

func NewRechargeService(pool *pgxpool.Pool,channels *ConfigStore,client GMPayClient)*RechargeService{
	return &RechargeService{pool:pool,channels:channels,client:client}
}

func newMerchantOrderID()(string,error){
	random:=make([]byte,12)
	if _,err:=rand.Read(random);err!=nil{return "",err}
	return "RC"+hex.EncodeToString(random),nil
}

func (s *RechargeService) CreateGMPayOrder(ctx context.Context,userID string,amountUSDT int64,notifyURL,redirectURL string)(RechargeOrder,error){
	if amountUSDT<=0{return RechargeOrder{},ErrInvalidChannelConfig}
	channel,err:=s.channels.Load(ctx,"epusdt_gmpay")
	if err!=nil{return RechargeOrder{},fmt.Errorf("load GMPay channel: %w",err)}
	if !channel.Enabled{return RechargeOrder{},ErrInvalidChannelConfig}
	merchantOrderID,err:=newMerchantOrderID()
	if err!=nil{return RechargeOrder{},fmt.Errorf("generate recharge merchant order ID: %w",err)}
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{})
	if err!=nil{return RechargeOrder{},fmt.Errorf("begin recharge order: %w",err)}
	defer tx.Rollback(ctx)
	var order RechargeOrder
	err=tx.QueryRow(ctx,`INSERT INTO wallet_recharge_orders(user_id,channel_id,channel_code,merchant_order_id,amount_usdt,status)
		SELECT $1,id,'epusdt_gmpay',$2,$3,'pending' FROM payment_channels WHERE code='epusdt_gmpay' AND enabled=TRUE
		RETURNING id::text,order_no,merchant_order_id,amount_usdt,status,created_at`,userID,merchantOrderID,amountUSDT).
		Scan(&order.ID,&order.OrderNo,&order.MerchantOrderID,&order.AmountUSDT,&order.Status,&order.CreatedAt)
	if err!=nil{return RechargeOrder{},fmt.Errorf("persist pending recharge order: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return RechargeOrder{},fmt.Errorf("commit pending recharge order: %w",err)}

	provider,err:=s.client.CreateOrder(ctx,channel,merchantOrderID,amountUSDT,notifyURL,redirectURL)
	if err!=nil{
		// A transport timeout is ambiguous: the gateway may have accepted the order
		// even if its response was lost. Keep the local order pending for callback
		// delivery and later reconciliation instead of falsely marking it failed.
		return RechargeOrder{},fmt.Errorf("create provider recharge order (local order remains pending for reconciliation): %w",err)
	}
	if provider.Status!=1||provider.PaymentURL==""{
		_,_ = s.pool.Exec(ctx,`UPDATE wallet_recharge_orders SET status='failed',provider_trade_id=$2,provider_status=$3,updated_at=now() WHERE id=$1 AND status='pending'`,order.ID,provider.TradeID,provider.Status)
		return RechargeOrder{},ErrProviderRejected
	}
	var expiresAt *time.Time
	if provider.ExpirationTime>0{value:=time.Unix(provider.ExpirationTime,0).UTC();expiresAt=&value}
	_,err=s.pool.Exec(ctx,`UPDATE wallet_recharge_orders SET provider_trade_id=$2,actual_amount=NULLIF($3,'')::numeric,
		actual_token=NULLIF($4,''),receive_address=NULLIF($5,''),payment_url=$6,provider_status=$7,expires_at=$8,updated_at=now()
		WHERE id=$1 AND status='pending'`,order.ID,provider.TradeID,provider.ActualAmount,provider.Token,provider.ReceiveAddress,provider.PaymentURL,provider.Status,expiresAt)
	if err!=nil{return RechargeOrder{},fmt.Errorf("save provider recharge details: %w",err)}
	order.ProviderTradeID=provider.TradeID
	order.ProviderStatus=provider.Status
	order.ActualAmount=provider.ActualAmount
	order.Token=provider.Token
	order.ReceiveAddress=provider.ReceiveAddress
	order.PaymentURL=provider.PaymentURL
	order.ExpiresAt=expiresAt
	return order,nil
}

func (s *RechargeService) ApplyGMPayCallback(ctx context.Context,params map[string]string,payloadHash []byte)error{
	if len(payloadHash)!=32{return ErrInvalidGMPayCallback}
	var channelCode string
	var expectedPID string
	var secret string
	var orderID string
	var expectedAmount int64
	var userID string
	var currentStatus string
	var providerTradeID string
	var rechargeID string
	err:=s.pool.QueryRow(ctx,`SELECT channel_code,merchant_order_id,amount_usdt,user_id::text,status,COALESCE(provider_trade_id,''),id::text
		FROM wallet_recharge_orders WHERE channel_code='epusdt_gmpay' AND merchant_order_id=$1`,params["order_id"]).
		Scan(&channelCode,&orderID,&expectedAmount,&userID,&currentStatus,&providerTradeID,&rechargeID)
	if err!=nil{if errors.Is(err,pgx.ErrNoRows){return ErrRechargeOrderNotFound};return fmt.Errorf("find GMPay recharge order: %w",err)}
	channel,err:=s.channels.Load(ctx,channelCode)
	if err!=nil{return fmt.Errorf("load GMPay callback credentials: %w",err)}
	expectedPID,secret=channel.MerchantPID,channel.SecretKey
	verified,verifyErr:=VerifyGMPayCallback(params,secret,expectedPID,orderID,expectedAmount)
	if verifyErr!=nil&&!errors.Is(verifyErr,ErrNonIntegerUSDT){return verifyErr}
	if providerTradeID==""||verified.TradeID!=providerTradeID{return ErrInvalidGMPayCallback}

	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{})
	if err!=nil{return fmt.Errorf("begin GMPay settlement: %w",err)}
	defer tx.Rollback(ctx)
	var lockedStatus string
	var lockedUserID string
	var lockedTradeID string
	if err:=tx.QueryRow(ctx,`SELECT status,user_id::text,COALESCE(provider_trade_id,'') FROM wallet_recharge_orders WHERE id=$1 FOR UPDATE`,rechargeID).
		Scan(&lockedStatus,&lockedUserID,&lockedTradeID);err!=nil{return fmt.Errorf("lock recharge order: %w",err)}
	if lockedStatus=="paid"||lockedStatus=="review"{
		if lockedTradeID!=verified.TradeID{return ErrRechargeOrderConflict}
		if err:=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit duplicate GMPay callback: %w",err)}
		return nil
	}
	if lockedStatus!="pending"||lockedUserID!=userID||lockedTradeID!=verified.TradeID{return ErrRechargeOrderConflict}
	if _,err:=tx.Exec(ctx,`INSERT INTO payment_callback_events(channel_code,provider_trade_id,merchant_order_id,payload_hash)
		VALUES($1,$2,$3,$4) ON CONFLICT(channel_code,provider_trade_id,payload_hash) DO NOTHING`,channelCode,verified.TradeID,verified.OrderID,payloadHash);err!=nil{
		return fmt.Errorf("record payment callback event: %w",err)
	}
	if errors.Is(verifyErr,ErrNonIntegerUSDT){
		if _,err:=tx.Exec(ctx,`UPDATE wallet_recharge_orders SET status='review',provider_status=2,actual_amount=$2::numeric,
			actual_token='USDT',block_transaction_id=$3,updated_at=now() WHERE id=$1`,rechargeID,params["actual_amount"],verified.TransactionID);err!=nil{
			return fmt.Errorf("mark fractional GMPay settlement for review: %w",err)
		}
		if err:=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit GMPay settlement review: %w",err)}
		return ErrRechargeNeedsReview
	}
	var balance int64
	if err:=tx.QueryRow(ctx,`INSERT INTO wallet_accounts(user_id,balance_usdt) VALUES($1,$2)
		ON CONFLICT(user_id) DO UPDATE SET balance_usdt=wallet_accounts.balance_usdt+EXCLUDED.balance_usdt,updated_at=now()
		RETURNING balance_usdt`,userID,verified.CreditedUSDT).Scan(&balance);err!=nil{return fmt.Errorf("credit recharge wallet: %w",err)}
	ledgerKey:="wallet-recharge:"+rechargeID
	if _,err:=tx.Exec(ctx,`INSERT INTO wallet_ledger(user_id,entry_type,amount_usdt,balance_after_usdt,reference_type,reference_id,idempotency_key,description)
		VALUES($1,'deposit',$2,$3,'wallet_recharge',$4,$5,'EPUSDT GMPay 钱包充值')`,userID,verified.CreditedUSDT,balance,rechargeID,ledgerKey);err!=nil{
		return fmt.Errorf("write recharge ledger: %w",err)
	}
	_,err=tx.Exec(ctx,`UPDATE wallet_recharge_orders SET status='paid',provider_status=2,actual_amount=$2::numeric,actual_token='USDT',
		block_transaction_id=$3,paid_at=now(),updated_at=now() WHERE id=$1`,rechargeID,strconv.FormatInt(verified.CreditedUSDT,10),verified.TransactionID)
	if err!=nil{return fmt.Errorf("mark recharge order paid: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit GMPay wallet settlement: %w",err)}
	return nil
}


func (s *RechargeService) ListUserRechargeOrders(ctx context.Context,userID string,limit,offset int)([]RechargeOrder,int64,error){
	var total int64
	if err:=s.pool.QueryRow(ctx,`SELECT count(*) FROM wallet_recharge_orders WHERE user_id=$1`,userID).Scan(&total);err!=nil{return nil,0,fmt.Errorf("count wallet recharge orders: %w",err)}
	rows,err:=s.pool.Query(ctx,`SELECT id::text,order_no,merchant_order_id,COALESCE(provider_trade_id,''),amount_usdt,COALESCE(actual_amount::text,''),COALESCE(actual_token,''),COALESCE(actual_network,''),COALESCE(receive_address,''),COALESCE(payment_url,''),status,COALESCE(provider_status,0),expires_at,created_at
		FROM wallet_recharge_orders WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`,userID,limit,offset)
	if err!=nil{return nil,0,fmt.Errorf("list wallet recharge orders: %w",err)}
	defer rows.Close()
	items:=make([]RechargeOrder,0)
	for rows.Next(){var item RechargeOrder;if err:=rows.Scan(&item.ID,&item.OrderNo,&item.MerchantOrderID,&item.ProviderTradeID,&item.AmountUSDT,&item.ActualAmount,&item.Token,&item.Network,&item.ReceiveAddress,&item.PaymentURL,&item.Status,&item.ProviderStatus,&item.ExpiresAt,&item.CreatedAt);err!=nil{return nil,0,fmt.Errorf("scan wallet recharge order: %w",err)};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate wallet recharge orders: %w",err)}
	return items,total,nil
}


func (s *RechargeService) GetGMPayPublicConfig(ctx context.Context)(PublicConfig,error){
	channel,err:=s.channels.Load(ctx,"epusdt_gmpay")
	if err!=nil{return PublicConfig{},fmt.Errorf("load GMPay channel: %w",err)}
	return s.client.GetPublicConfig(ctx,channel.BaseURL)
}


type AdminRechargeOrder struct {
	ID string `json:"id"`
	OrderNo string `json:"order_no"`
	UserID string `json:"user_id"`
	Email string `json:"email"`
	ChannelCode string `json:"channel_code"`
	AmountUSDT int64 `json:"amount_usdt"`
	ActualAmount string `json:"actual_amount,omitempty"`
	Token string `json:"token,omitempty"`
	Network string `json:"network,omitempty"`
	ProviderTradeID string `json:"provider_trade_id,omitempty"`
	TransactionID string `json:"block_transaction_id,omitempty"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	PaidAt *time.Time `json:"paid_at,omitempty"`
}

func (s *RechargeService) ListAdminRechargeOrders(ctx context.Context,status string,limit,offset int)([]AdminRechargeOrder,int64,error){
	var total int64
	if err:=s.pool.QueryRow(ctx,`SELECT count(*) FROM wallet_recharge_orders WHERE ($1='' OR status=$1)`,status).Scan(&total);err!=nil{return nil,0,fmt.Errorf("count admin recharge orders: %w",err)}
	rows,err:=s.pool.Query(ctx,`SELECT o.id::text,o.order_no,o.user_id::text,u.email,o.channel_code,o.amount_usdt,
		COALESCE(o.actual_amount::text,''),COALESCE(o.actual_token,''),COALESCE(o.actual_network,''),
		COALESCE(o.provider_trade_id,''),COALESCE(o.block_transaction_id,''),o.status,o.created_at,o.paid_at
		FROM wallet_recharge_orders o JOIN users u ON u.id=o.user_id
		WHERE ($1='' OR o.status=$1) ORDER BY o.created_at DESC,o.id DESC LIMIT $2 OFFSET $3`,status,limit,offset)
	if err!=nil{return nil,0,fmt.Errorf("list admin recharge orders: %w",err)}
	defer rows.Close()
	items:=make([]AdminRechargeOrder,0)
	for rows.Next(){var item AdminRechargeOrder;if err:=rows.Scan(&item.ID,&item.OrderNo,&item.UserID,&item.Email,&item.ChannelCode,&item.AmountUSDT,&item.ActualAmount,&item.Token,&item.Network,&item.ProviderTradeID,&item.TransactionID,&item.Status,&item.CreatedAt,&item.PaidAt);err!=nil{return nil,0,fmt.Errorf("scan admin recharge order: %w",err)};items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate admin recharge orders: %w",err)}
	return items,total,nil
}

func (s *RechargeService) RefreshUserRechargeOrder(ctx context.Context,userID,orderID string)(RechargeOrder,error){
	var channelCode,tradeID,status string
	err:=s.pool.QueryRow(ctx,`SELECT channel_code,COALESCE(provider_trade_id,''),status FROM wallet_recharge_orders WHERE id=$1 AND user_id=$2`,orderID,userID).Scan(&channelCode,&tradeID,&status)
	if err!=nil{if errors.Is(err,pgx.ErrNoRows){return RechargeOrder{},ErrRechargeOrderNotFound};return RechargeOrder{},fmt.Errorf("load recharge order for status refresh: %w",err)}
	if status=="paid"||status=="expired"||status=="failed"||status=="review"||tradeID==""{
		return s.getUserRechargeOrder(ctx,userID,orderID)
	}
	channel,err:=s.channels.Load(ctx,channelCode)
	if err!=nil{return RechargeOrder{},fmt.Errorf("load channel for status refresh: %w",err)}
	providerStatus,err:=s.client.CheckOrderStatus(ctx,channel.BaseURL,tradeID)
	if err!=nil{return RechargeOrder{},fmt.Errorf("refresh provider order status: %w",err)}
	newStatus:=status
	switch providerStatus{
	case 3:newStatus="expired"
	case 6:newStatus="review"
	// Status 2 is not used to credit the wallet. The signed merchant callback
	// remains the only automatic wallet-credit authority.
	}
	if _,err:=s.pool.Exec(ctx,`UPDATE wallet_recharge_orders SET provider_status=$2,status=$3,updated_at=now() WHERE id=$1 AND status='pending'`,orderID,providerStatus,newStatus);err!=nil{return RechargeOrder{},fmt.Errorf("save refreshed recharge status: %w",err)}
	return s.getUserRechargeOrder(ctx,userID,orderID)
}

func (s *RechargeService) getUserRechargeOrder(ctx context.Context,userID,orderID string)(RechargeOrder,error){
	var item RechargeOrder
	err:=s.pool.QueryRow(ctx,`SELECT id::text,order_no,merchant_order_id,COALESCE(provider_trade_id,''),amount_usdt,COALESCE(actual_amount::text,''),COALESCE(actual_token,''),COALESCE(actual_network,''),COALESCE(receive_address,''),COALESCE(payment_url,''),status,COALESCE(provider_status,0),expires_at,created_at
		FROM wallet_recharge_orders WHERE id=$1 AND user_id=$2`,orderID,userID).Scan(&item.ID,&item.OrderNo,&item.MerchantOrderID,&item.ProviderTradeID,&item.AmountUSDT,&item.ActualAmount,&item.Token,&item.Network,&item.ReceiveAddress,&item.PaymentURL,&item.Status,&item.ProviderStatus,&item.ExpiresAt,&item.CreatedAt)
	if err!=nil{if errors.Is(err,pgx.ErrNoRows){return RechargeOrder{},ErrRechargeOrderNotFound};return RechargeOrder{},fmt.Errorf("read refreshed recharge order: %w",err)}
	return item,nil
}

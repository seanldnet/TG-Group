package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/seanldnet/tg-group/internal/database"
)

func TestPurchasePlanWithWalletPostgres(t *testing.T) {
	databaseURL:=os.Getenv("TEST_DATABASE_URL")
	if databaseURL==""{t.Skip("TEST_DATABASE_URL is not configured")}
	ctx,cancel:=context.WithTimeout(context.Background(),45*time.Second)
	defer cancel()
	db,err:=database.Open(ctx,databaseURL)
	if err!=nil{t.Fatalf("open test database: %v",err)}
	defer db.Close()
	if err:=db.Migrate(ctx);err!=nil{t.Fatalf("apply migrations: %v",err)}

	service:=NewService(db.Pool)
	email:=fmt.Sprintf("plan-purchase-%d@example.test",time.Now().UnixNano())
	session,err:=service.Register(ctx,email,"correct-horse-battery","Plan Purchase Test","","127.0.0.1:4321")
	if err!=nil{t.Fatalf("register buyer: %v",err)}
	if _,err:=db.Pool.Exec(ctx,`UPDATE wallet_accounts SET balance_usdt=1000 WHERE user_id=$1`,session.User.ID);err!=nil{t.Fatalf("fund buyer wallet: %v",err)}

	var planID,secondPlanID string
	code:=fmt.Sprintf("integration-%d",time.Now().UnixNano())
	err=db.Pool.QueryRow(ctx,`INSERT INTO plans(code,name,description,price_usdt,duration_days,bot_limit,group_limit,features)
		VALUES($1,'集成测试套餐','',100,30,2,5,'{"feature":"enabled"}'::jsonb) RETURNING id::text`,code).Scan(&planID)
	if err!=nil{t.Fatalf("create test plan: %v",err)}
	err=db.Pool.QueryRow(ctx,`INSERT INTO plans(code,name,description,price_usdt,duration_days,bot_limit,group_limit,features)
		VALUES($1,'另一个套餐','',200,30,1,1,'{}'::jsonb) RETURNING id::text`,code+"-other").Scan(&secondPlanID)
	if err!=nil{t.Fatalf("create second test plan: %v",err)}

	order,err:=service.PurchasePlanWithWallet(ctx,session.User.ID,planID,"purchase-key-1")
	if err!=nil{t.Fatalf("purchase plan: %v",err)}
	if order.Status!="paid"||order.AmountUSDT!=100||order.PaymentMethod!="wallet"{t.Fatalf("unexpected order: %+v",order)}
	var balance int64
	if err:=db.Pool.QueryRow(ctx,`SELECT balance_usdt FROM wallet_accounts WHERE user_id=$1`,session.User.ID).Scan(&balance);err!=nil{t.Fatal(err)}
	if balance!=900{t.Fatalf("wallet balance = %d, want 900",balance)}
	var ledgerCount int
	if err:=db.Pool.QueryRow(ctx,`SELECT count(*) FROM wallet_ledger WHERE user_id=$1 AND reference_id=$2`,session.User.ID,order.ID).Scan(&ledgerCount);err!=nil{t.Fatal(err)}
	if ledgerCount!=1{t.Fatalf("purchase ledger entries = %d, want 1",ledgerCount)}

	replayed,err:=service.PurchasePlanWithWallet(ctx,session.User.ID,planID,"purchase-key-1")
	if err!=nil{t.Fatalf("repeat idempotent purchase: %v",err)}
	if replayed.ID!=order.ID{t.Fatalf("idempotent order id = %s, want %s",replayed.ID,order.ID)}
	if _,err:=service.PurchasePlanWithWallet(ctx,session.User.ID,secondPlanID,"purchase-key-1");!errors.Is(err,ErrOrderIdempotencyConflict){t.Fatalf("reuse key for another plan error = %v",err)}

	var firstExpiry time.Time
	if err:=db.Pool.QueryRow(ctx,`SELECT expires_at FROM licenses WHERE user_id=$1 AND plan_id=$2`,session.User.ID,planID).Scan(&firstExpiry);err!=nil{t.Fatalf("load activated license: %v",err)}
	renewed,err:=service.PurchasePlanWithWallet(ctx,session.User.ID,planID,"purchase-key-2")
	if err!=nil{t.Fatalf("renew plan: %v",err)}
	if renewed.ID==order.ID{t.Fatal("renewal unexpectedly returned original order")}
	var nextExpiry time.Time
	if err:=db.Pool.QueryRow(ctx,`SELECT expires_at FROM licenses WHERE user_id=$1 AND plan_id=$2`,session.User.ID,planID).Scan(&nextExpiry);err!=nil{t.Fatalf("load renewed license: %v",err)}
	if !nextExpiry.After(firstExpiry.Add(29*24*time.Hour)){t.Fatalf("renewed expiry %v did not extend previous expiry %v by 30 days",nextExpiry,firstExpiry)}
	if err:=db.Pool.QueryRow(ctx,`SELECT balance_usdt FROM wallet_accounts WHERE user_id=$1`,session.User.ID).Scan(&balance);err!=nil{t.Fatal(err)}
	if balance!=800{t.Fatalf("wallet balance after renewal = %d, want 800",balance)}

	adminOrders,adminOrderTotal,err:=service.ListAdminOrders(ctx,"paid",email,20,0)
	if err!=nil{t.Fatalf("list admin orders: %v",err)}
	if adminOrderTotal!=2||len(adminOrders)!=2{t.Fatalf("admin orders total/items = %d/%d, want 2/2",adminOrderTotal,len(adminOrders))}
	adminLicenses,adminLicenseTotal,err:=service.ListAdminLicenses(ctx,"active",email,20,0)
	if err!=nil{t.Fatalf("list admin licenses: %v",err)}
	if adminLicenseTotal!=1||len(adminLicenses)!=1{t.Fatalf("admin licenses total/items = %d/%d, want 1/1",adminLicenseTotal,len(adminLicenses))}
	licenseID:=adminLicenses[0].ID
	suspended,err:=service.SetAdminLicenseStatus(ctx,session.User.ID,licenseID,"suspended")
	if err!=nil{t.Fatalf("suspend license: %v",err)}
	if suspended.Status!="suspended"{t.Fatalf("license status = %s, want suspended",suspended.Status)}
	if _,err:=service.PurchasePlanWithWallet(ctx,session.User.ID,planID,"purchase-key-suspended");!errors.Is(err,ErrLicenseSuspended){t.Fatalf("purchase while license suspended error = %v",err)}
	restored,err:=service.SetAdminLicenseStatus(ctx,session.User.ID,licenseID,"active")
	if err!=nil{t.Fatalf("restore license: %v",err)}
	if restored.Status!="active"{t.Fatalf("restored license status = %s, want active",restored.Status)}

	summary,err:=service.GetAdminFinanceSummary(ctx)
	if err!=nil{t.Fatalf("get platform finance summary: %v",err)}
	if summary.WalletCount<1||summary.LedgerEntryCount<2||summary.DebitTotalUSDT<200{t.Fatalf("unexpected platform finance summary: %+v",summary)}
	ledger,ledgerTotal,err:=service.ListAdminWalletLedger(ctx,"order_payment",email,20,0)
	if err!=nil{t.Fatalf("list platform wallet ledger: %v",err)}
	if ledgerTotal!=2||len(ledger)!=2{t.Fatalf("platform ledger total/items = %d/%d, want 2/2",ledgerTotal,len(ledger))}
}

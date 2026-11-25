package bots

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/seanldnet/tg-group/internal/auth"
	"github.com/seanldnet/tg-group/internal/database"
)

func TestBotLifecycleAndTenantIsolationPostgres(t *testing.T) {
	databaseURL:=os.Getenv("TEST_DATABASE_URL")
	if databaseURL==""{t.Skip("TEST_DATABASE_URL is not configured")}
	ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
	db,err:=database.Open(ctx,databaseURL);if err!=nil{t.Fatalf("open database: %v",err)};defer db.Close()
	if err:=db.Migrate(ctx);err!=nil{t.Fatalf("migrate: %v",err)}
	authService:=auth.NewService(db.Pool)
	suffix:=time.Now().UnixNano()
	owner,err:=authService.Register(ctx,fmt.Sprintf("bot-owner-%d@example.test",suffix),"correct-horse-battery","Bot Owner","","")
	if err!=nil{t.Fatalf("register owner: %v",err)}
	other,err:=authService.Register(ctx,fmt.Sprintf("bot-other-%d@example.test",suffix),"correct-horse-battery","Other Owner","","")
	if err!=nil{t.Fatalf("register other tenant: %v",err)}

	telegramID:=suffix
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.URL.Path!="/bot111:token-one-abcdefghijklmnop/getMe" && r.URL.Path!="/bot222:token-two-abcdefghijklmnop/getMe"{http.NotFound(w,r);return}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok":true,"result":map[string]any{"id":telegramID,"is_bot":true,"first_name":"Integration Bot","username":"integration_test_bot"}})
	}))
	defer server.Close()
	key:=make([]byte,32)
	for i:=range key{key[i]=byte(i+1)}
	service,err:=NewService(db.Pool,encodeKey(key));if err!=nil{t.Fatalf("create service: %v",err)}
	service.apiBaseURL=server.URL

	created,err:=service.Add(ctx,owner.User.TenantID,owner.User.ID,"111:token-one-abcdefghijklmnop")
	if err!=nil{t.Fatalf("add bot: %v",err)}
	if created.TelegramBotID!=telegramID || created.Username!="integration_test_bot" || created.Status!="active"{t.Fatalf("unexpected bot response: %+v",created)}
	var stored []byte
	if err:=db.Pool.QueryRow(ctx,"SELECT token_ciphertext FROM telegram_bots WHERE id=$1",created.ID).Scan(&stored);err!=nil{t.Fatal(err)}
	if string(stored)=="111:token-one-abcdefghijklmnop"{t.Fatal("bot token was stored in plaintext")}
	recovered,err:=service.GetToken(ctx,owner.User.TenantID,created.ID)
	if err!=nil || recovered!="111:token-one-abcdefghijklmnop"{t.Fatalf("GetToken = %q, %v",recovered,err)}
	if _,err:=service.GetToken(ctx,other.User.TenantID,created.ID);err!=ErrNotFound{t.Fatalf("cross-tenant token lookup error = %v",err)}

	otherBots,err:=service.List(ctx,other.User.TenantID)
	if err!=nil || len(otherBots)!=0{t.Fatalf("other tenant bot list = %v, %v",otherBots,err)}
	disabled,err:=service.SetStatus(ctx,owner.User.TenantID,created.ID,"disabled")
	if err!=nil || disabled.Status!="disabled"{t.Fatalf("disable bot = %+v, %v",disabled,err)}
	if _,err:=service.GetToken(ctx,owner.User.TenantID,created.ID);err!=ErrBotDisabled{t.Fatalf("disabled Bot token lookup error = %v",err)}
	if _,err:=service.RotateToken(ctx,owner.User.TenantID,created.ID,"222:token-two-abcdefghijklmnop");err!=nil{t.Fatalf("rotate token: %v",err)}
	if _,err:=service.SetStatus(ctx,owner.User.TenantID,created.ID,"active");err!=nil{t.Fatalf("re-enable bot: %v",err)}
	recovered,err=service.GetToken(ctx,owner.User.TenantID,created.ID)
	if err!=nil || recovered!="222:token-two-abcdefghijklmnop"{t.Fatalf("rotated token = %q, %v",recovered,err)}
	if err:=service.Remove(ctx,owner.User.TenantID,created.ID);err!=nil{t.Fatalf("remove bot: %v",err)}
}

func encodeKey(key []byte)string{return base64.StdEncoding.EncodeToString(key)}

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

	var groupID string
	if err:=db.Pool.QueryRow(ctx,`INSERT INTO telegram_groups(tenant_id,bot_id,telegram_chat_id,title,chat_type)
		VALUES($1,$2,$3,'Integration Group','supergroup') RETURNING id::text`,owner.User.TenantID,created.ID,-telegramID).Scan(&groupID);err!=nil{t.Fatalf("create test group: %v",err)}
	rule,err:=service.CreateGroupRule(ctx,owner.User.TenantID,groupID,GroupRuleInput{
		RuleType:"keyword_filter",Name:"integration keyword",Enabled:true,Priority:10,
		Config:json.RawMessage(`{"words":["integration"],"action":"delete"}`),
	})
	if err!=nil{t.Fatalf("create group rule: %v",err)}
	rules,err:=service.ListGroupRules(ctx,owner.User.TenantID,groupID)
	if err!=nil || len(rules)!=1 || rules[0].ID!=rule.ID{t.Fatalf("owner group rules = %+v, %v",rules,err)}
	otherRules,err:=service.ListGroupRules(ctx,other.User.TenantID,groupID)
	if err!=nil || len(otherRules)!=0{t.Fatalf("cross-tenant group rules = %+v, %v",otherRules,err)}
	if _,err:=db.Pool.Exec(ctx,`INSERT INTO telegram_group_rule_events(
		tenant_id,group_id,rule_id,telegram_update_id,telegram_message_id,telegram_user_id,action,status,detail)
		VALUES($1,$2,$3,9101,88,12345,'delete_message','succeeded','message deleted')`,
		owner.User.TenantID,groupID,rule.ID);err!=nil{t.Fatalf("create rule event: %v",err)}
	ruleEvents,err:=service.ListGroupRuleEvents(ctx,owner.User.TenantID,groupID,50)
	if err!=nil || len(ruleEvents)!=1 || ruleEvents[0].Status!="succeeded" || ruleEvents[0].RuleID!=rule.ID{
		t.Fatalf("owner group rule events = %+v, %v",ruleEvents,err)
	}
	otherRuleEvents,err:=service.ListGroupRuleEvents(ctx,other.User.TenantID,groupID,50)
	if err!=nil || len(otherRuleEvents)!=0{t.Fatalf("cross-tenant rule events = %+v, %v",otherRuleEvents,err)}
	if _,err:=db.Pool.Exec(ctx,`INSERT INTO telegram_message_events(
		tenant_id,bot_id,group_id,telegram_update_id,telegram_message_id,telegram_user_id,
		sender_username,sender_display_name,message_type,message_text)
		VALUES($1,$2,$3,9001,77,12345,'alice','Alice Example','text','hello group')`,
		owner.User.TenantID,created.ID,groupID);err!=nil{t.Fatalf("create message event: %v",err)}

	if _,err:=db.Pool.Exec(ctx,`INSERT INTO telegram_group_members(
		tenant_id,bot_id,group_id,telegram_user_id,username,first_name,status,can_delete_messages,joined_at)
		VALUES($1,$2,$3,54321,'member_one','Member One','administrator',true,now())`,
		owner.User.TenantID,created.ID,groupID);err!=nil{t.Fatalf("create group member: %v",err)}
	members,err:=service.ListGroupMembers(ctx,owner.User.TenantID,groupID,100)
	if err!=nil || len(members)!=1{t.Fatalf("owner group members = %+v, %v",members,err)}
	if members[0].TelegramUserID!=54321 || members[0].Status!="administrator" || !members[0].CanDeleteMessages {
		t.Fatalf("unexpected group member: %+v",members[0])
	}
	otherMembers,err:=service.ListGroupMembers(ctx,other.User.TenantID,groupID,100)
	if err!=nil || len(otherMembers)!=0{t.Fatalf("cross-tenant group members = %+v, %v",otherMembers,err)}
	events,err:=service.ListGroupEvents(ctx,owner.User.TenantID,groupID,50)
	if err!=nil || len(events)!=1{t.Fatalf("owner group events = %+v, %v",events,err)}
	if events[0].TelegramUserID!=12345 || events[0].MessageText!="hello group" || events[0].SenderUsername!="alice"{t.Fatalf("unexpected message event: %+v",events[0])}
	otherEvents,err:=service.ListGroupEvents(ctx,other.User.TenantID,groupID,50)
	if err!=nil || len(otherEvents)!=0{t.Fatalf("cross-tenant group events = %+v, %v",otherEvents,err)}
	otherBots,err:=service.List(ctx,other.User.TenantID)
	if err!=nil || len(otherBots)!=0{t.Fatalf("other tenant bot list = %v, %v",otherBots,err)}
	disabled,err:=service.SetStatus(ctx,owner.User.TenantID,owner.User.ID,created.ID,"disabled")
	if err!=nil || disabled.Status!="disabled"{t.Fatalf("disable bot = %+v, %v",disabled,err)}
	if _,err:=service.GetToken(ctx,owner.User.TenantID,created.ID);err!=ErrBotDisabled{t.Fatalf("disabled Bot token lookup error = %v",err)}
	if _,err:=service.RotateToken(ctx,owner.User.TenantID,owner.User.ID,created.ID,"222:token-two-abcdefghijklmnop");err!=nil{t.Fatalf("rotate token: %v",err)}
	if _,err:=service.SetStatus(ctx,owner.User.TenantID,owner.User.ID,created.ID,"active");err!=nil{t.Fatalf("re-enable bot: %v",err)}
	recovered,err=service.GetToken(ctx,owner.User.TenantID,created.ID)
	if err!=nil || recovered!="222:token-two-abcdefghijklmnop"{t.Fatalf("rotated token = %q, %v",recovered,err)}
	if err:=service.Remove(ctx,owner.User.TenantID,owner.User.ID,created.ID);err!=nil{t.Fatalf("remove bot: %v",err)}
	var auditCount int
	if err:=db.Pool.QueryRow(ctx,"SELECT count(*) FROM audit_logs WHERE tenant_id=$1 AND actor_user_id=$2 AND resource_type='telegram_bot'",owner.User.TenantID,owner.User.ID).Scan(&auditCount);err!=nil{t.Fatal(err)}
	if auditCount!=5{t.Fatalf("Bot audit records = %d, want 5",auditCount)}
}

func encodeKey(key []byte)string{return base64.StdEncoding.EncodeToString(key)}

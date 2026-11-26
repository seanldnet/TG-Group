package telegramruntime

import ("testing"; "encoding/json"; "github.com/seanldnet/tg-group/internal/bots")

func TestRenderNotice(t *testing.T) {
	member:=telegramSender{ID:42,Username:"ada",FirstName:"Ada",LastName:"Lovelace"}
	got:=renderNotice("欢迎 {user}（{username}）加入 {group}，ID={id}",member,"TG 群")
	want:="欢迎 Ada Lovelace（@ada）加入 TG 群，ID={id}"
	if got!=want { t.Fatalf("renderNotice() = %q, want %q",got,want) }
}

func TestRenderNoticeWithoutUsername(t *testing.T) {
	member:=telegramSender{ID:987654,FirstName:"Guest"}
	got:=renderNotice("欢迎 {user} {username}",member,"")
	if got!="欢迎 Guest " { t.Fatalf("renderNotice() = %q",got) }
}

func TestRenderNoticeTruncatesToTelegramLimit(t *testing.T) {
	member:=telegramSender{ID:1,FirstName:"A"}
	got:=renderNotice(string(make([]rune,0)),member,"")
	_ = got
	long:=renderNotice(repeatRune('中',5000),member,"")
	if len([]rune(long))!=4096 { t.Fatalf("notice rune length = %d, want 4096",len([]rune(long))) }
}

func repeatRune(value rune,count int) string {
	result:=make([]rune,count)
	for i:=range result { result[i]=value }
	return string(result)
}

func TestJoinChallengeOptionsAreDistinctAndContainAnswer(t *testing.T) {
	for answer:=int64(2);answer<=18;answer++ {
		options:=joinChallengeOptions(answer)
		if len(options)!=3 { t.Fatalf("answer %d has %d options",answer,len(options)) }
		seen:=map[int64]bool{}
		for _,option:=range options {
			if option<2 || option>18 { t.Fatalf("answer %d generated out-of-range option %d",answer,option) }
			if seen[option] { t.Fatalf("answer %d generated duplicate option %d",answer,option) }
			seen[option]=true
		}
		if !seen[answer] { t.Fatalf("answer %d missing from options %v",answer,options) }
	}
}

func TestJoinChallengeOptionsRejectsInvalidAnswer(t *testing.T) {
	if options:=joinChallengeOptions(1);options!=nil { t.Fatalf("invalid answer options = %v",options) }
	if options:=joinChallengeOptions(19);options!=nil { t.Fatalf("invalid answer options = %v",options) }
}

func TestGroupOwnerAccessUsesLinkedTenantIdentity(t *testing.T) {
	tests:=[]struct{name string;hasLinkedOwner,isOwner bool;want string}{
		{name:"no account linked",hasLinkedOwner:false,isOwner:false,want:"unbound"},
		{name:"different Telegram account",hasLinkedOwner:true,isOwner:false,want:"not_owner"},
		{name:"linked account",hasLinkedOwner:true,isOwner:true,want:"owner"},
	}
	for _,test:=range tests {
		t.Run(test.name,func(t *testing.T){
			if got:=groupOwnerAccess(test.hasLinkedOwner,test.isOwner);got!=test.want {
				t.Fatalf("groupOwnerAccess()=%q, want %q",got,test.want)
			}
		})
	}
}

func TestValidRuleUUID(t *testing.T) {
    tests:=[]struct{value string;want bool}{
        {"00000000-0000-4000-8000-000000000001",true},
        {"not-a-uuid",false},
        {"00000000000040008000000000000001",false},
        {"00000000-0000-4000-8000-00000000000Z",false},
        {"",false},
    }
    for _,test:=range tests {
        if got:=validRuleUUID(test.value);got!=test.want {
            t.Errorf("validRuleUUID(%q)=%v want %v",test.value,got,test.want)
        }
    }
}

func TestParseGroupMemberCallback(t *testing.T) {
	tests := []struct {
		name string
		data string
		stage string
		action string
		targetID int64
		wantOK bool
	}{
		{name: "ask mute", data: "group:member:ask:mute:123456", stage: "ask", action: "mute", targetID: 123456, wantOK: true},
		{name: "confirm ban", data: "group:member:confirm:ban:987654", stage: "confirm", action: "ban", targetID: 987654, wantOK: true},
		{name: "ask unmute", data: "group:member:ask:unmute:123", stage: "ask", action: "unmute", targetID: 123, wantOK: true},
		{name: "confirm unban", data: "group:member:confirm:unban:456", stage: "confirm", action: "unban", targetID: 456, wantOK: true},
		{name: "reject zero id", data: "group:member:confirm:ban:0", wantOK: false},
		{name: "reject negative id", data: "group:member:ask:mute:-1", wantOK: false},
		{name: "reject unsupported action", data: "group:member:ask:kick:123", wantOK: false},
		{name: "reject malformed payload", data: "group:member:ask:mute:not-an-id", wantOK: false},
		{name: "reject unrelated callback", data: "group:rules", wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stage, action, targetID, ok := parseGroupMemberCallback(test.data)
			if ok != test.wantOK || stage != test.stage || action != test.action || targetID != test.targetID {
				t.Fatalf("parseGroupMemberCallback(%q) = (%q, %q, %d, %v)", test.data, stage, action, targetID, ok)
			}
		})
	}
}

func TestParseGroupRuleDeleteCallback(t *testing.T) {
	const ruleID = "00000000-0000-4000-8000-000000000001"
	tests := []struct {
		name string
		data string
		stage string
		wantOK bool
	}{
		{name:"ask",data:"group:delete:ask:"+ruleID+":rule",stage:"ask",wantOK:true},
		{name:"confirm",data:"group:delete:confirm:"+ruleID+":rule",stage:"confirm",wantOK:true},
		{name:"invalid uuid",data:"group:delete:confirm:not-a-uuid:rule"},
		{name:"invalid stage",data:"group:delete:remove:"+ruleID+":rule"},
		{name:"invalid suffix",data:"group:delete:confirm:"+ruleID+":other"},
		{name:"unrelated callback",data:"group:rules"},
	}
	for _,test:=range tests {
		t.Run(test.name,func(t *testing.T){
			stage,id,ok:=parseGroupRuleDeleteCallback(test.data)
			if ok!=test.wantOK { t.Fatalf("ok=%v want %v",ok,test.wantOK) }
			if ok && (stage!=test.stage || id!=ruleID) { t.Fatalf("got (%q,%q), want (%q,%q)",stage,id,test.stage,ruleID) }
		})
	}
}

func TestParseGroupRuleEditCallback(t *testing.T) {
	const id="00000000-0000-4000-8000-000000000001"
	tests:=[]struct{data,stage,preset string;want bool}{
		{"group:ed:"+id,"open","",true},
		{"group:es:"+id+":w1","set","w1",true},
		{"group:es:"+id+":v3","set","v3",true},
		{"group:es:"+id+":unknown","","",false},
		{"group:ed:not-a-uuid","","",false},
		{"group:es:"+id+":w1:extra","","",false},
	}
	for _,tt:=range tests {
		stage,_,preset,ok:=parseGroupRuleEditCallback(tt.data)
		if ok!=tt.want || stage!=tt.stage || preset!=tt.preset { t.Errorf("parseGroupRuleEditCallback(%q)=(%q,%q,%v)",tt.data,stage,preset,ok) }
	}
}

func TestGroupRuleEditConfigProducesValidTypeSpecificConfig(t *testing.T) {
	tests:=[]struct{kind,preset string}{
		{"welcome","w1"},{"leave_notice","l2"},{"keyword_filter","k1"},
		{"sensitive_word","k2"},{"anti_ad","a1"},{"anti_spam","s3"},
		{"join_verification","v2"},{"join_verification","vr"},
	}
	for _,tt:=range tests {
		config:=groupRuleEditConfig(tt.kind,tt.preset)
		if len(config)==0 { t.Fatalf("missing config for %s/%s",tt.kind,tt.preset) }
		if err:=bots.ValidateGroupRule(bots.GroupRuleInput{RuleType:tt.kind,Name:"test",Enabled:true,Priority:10,Config:json.RawMessage(config)});err!=nil { t.Errorf("%s/%s config invalid: %v",tt.kind,tt.preset,err) }
	}
	if config:=groupRuleEditConfig("welcome","k1");config!=nil { t.Fatal("mismatched preset should be rejected") }
}

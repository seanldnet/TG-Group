package telegramruntime

import ("testing"; "encoding/json"; "strconv"; "strings"; "github.com/seanldnet/tg-group/internal/bots")

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
		{"sensitive_word","k2"},{"anti_ad","a1"},{"anti_spam","s3"},{"media_restriction","m1"},
		{"join_verification","v2"},{"join_verification","vr"},
	}
	for _,tt:=range tests {
		config:=groupRuleEditConfig(tt.kind,tt.preset)
		if len(config)==0 { t.Fatalf("missing config for %s/%s",tt.kind,tt.preset) }
		if err:=bots.ValidateGroupRule(bots.GroupRuleInput{RuleType:tt.kind,Name:"test",Enabled:true,Priority:10,Config:json.RawMessage(config)});err!=nil { t.Errorf("%s/%s config invalid: %v",tt.kind,tt.preset,err) }
	}
	if config:=groupRuleEditConfig("welcome","k1");config!=nil { t.Fatal("mismatched preset should be rejected") }
}

func TestParseGroupMemberDetailCallback(t *testing.T) {
	const prefix="group:member:detail:"
	tests:=[]struct{value string;want int64;ok bool}{
		{prefix+"123",123,true},{prefix+"0",0,false},{prefix+"-1",0,false},
		{prefix+"abc",0,false},{prefix+"9223372036854775808",0,false},{"group:members",0,false},
	}
	for _,tt:=range tests {
		got,ok:=parseGroupMemberDetailCallback(tt.value)
		if ok!=tt.ok || (ok && got!=tt.want) { t.Errorf("parseGroupMemberDetailCallback(%q) = (%d,%v), want (%d,%v)",tt.value,got,ok,tt.want,tt.ok) }
	}
}

func TestParseGroupMemberMuteConfirm(t *testing.T) {
	const id="123456"
	for _,minutes:=range []string{"10","30","60","1440"} {
		stage,action,gotMinutes,gotID,ok:=parseGroupMemberMuteConfirm("group:member:confirm:mute:"+minutes+":"+id)
		if !ok || stage!="confirm" || action!="mute" || gotID!=123456 { t.Fatalf("unexpected parse result for %s",minutes) }
		want,_:=strconv.Atoi(minutes);if gotMinutes!=want { t.Fatalf("minutes=%d want %d",gotMinutes,want) }
	}
	for _,value:=range []string{"group:member:confirm:mute:1:123","group:member:confirm:mute:999999:123","group:member:confirm:mute:30:0","group:member:confirm:ban:30:123"} {
		if _,_,_,_,ok:=parseGroupMemberMuteConfirm(value);ok { t.Errorf("accepted invalid callback %q",value) }
	}
}

func TestParseGroupMessageActionCallback(t *testing.T) {
	valid:=[]string{"group:msg:delete:ask:100","group:msg:delete:confirm:100","group:msg:pin:run:100"}
	for _,value:=range valid { if _,_,id,ok:=parseGroupMessageActionCallback(value);!ok || id!=100 { t.Errorf("rejected %q",value) } }
	invalid:=[]string{"group:msg:delete:run:100","group:msg:pin:confirm:100","group:msg:delete:ask:-1","group:msg:delete:ask:0","group:msg:other:ask:100"}
	for _,value:=range invalid { if _,_,_,ok:=parseGroupMessageActionCallback(value);ok { t.Errorf("accepted %q",value) } }
}

func TestParseGroupRuleCustomEditCallback(t *testing.T) {
 const id="00000000-0000-4000-8000-000000000001"
 for _,field:=range []string{"text","words","patterns","exceptions","domains","allowed_domains","invite_links","allowed_bots","media_types","channels"} { if got,gotField,ok:=parseGroupRuleCustomEditCallback("group:ec:"+id+":"+field); !ok||got!=id||gotField!=field { t.Errorf("rejected field %s",field) } }
 for _,value:=range []string{"group:ec:bad:text","group:ec:"+id+":unknown","group:ec:"+id+":words:extra"} { if _,_,ok:=parseGroupRuleCustomEditCallback(value);ok { t.Errorf("accepted %q",value) } }
}
func TestSplitRuleTerms(t *testing.T) {
 got:=splitRuleTerms("广告, 引流，推广\n广告；私聊")
 want:=[]string{"广告","引流","推广","私聊"}
 if len(got)!=len(want) { t.Fatalf("terms=%v want %v",got,want) }
 for i:=range want { if got[i]!=want[i] { t.Fatalf("terms=%v want %v",got,want) } }
}

func TestSplitRuleTermsDeduplicatesAndTrims(t *testing.T) { got:=splitRuleTerms(" @news, @News\n -1001234567890 "); if len(got)!=2 || got[0]!="@news" || got[1]!="-1001234567890" { t.Fatalf("unexpected terms: %#v",got) } }



func TestResolveGroupMemberListPolicyPrecedence(t *testing.T) {
	tests := []struct {
		name string
		lists []string
		blacklisted bool
		bypassVerification bool
	}{
		{name: "no entry"},
		{name: "blacklist", lists: []string{"blacklist"}, blacklisted: true},
		{name: "whitelist bypasses verification", lists: []string{"whitelist"}, bypassVerification: true},
		{name: "exempt bypasses verification", lists: []string{"exempt"}, bypassVerification: true},
		{name: "blacklist overrides whitelist", lists: []string{"whitelist", "blacklist"}, blacklisted: true, bypassVerification: true},
		{name: "blacklist overrides exempt", lists: []string{"exempt", "blacklist"}, blacklisted: true, bypassVerification: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveGroupMemberListPolicy(tt.lists)
			if got.blacklisted != tt.blacklisted || got.bypassVerification != tt.bypassVerification {
				t.Fatalf("policy = %+v, want blacklisted=%v bypassVerification=%v", got, tt.blacklisted, tt.bypassVerification)
			}
		})
	}
}


func TestMatchesQARule(t *testing.T) {
	tests := []struct { name,message string; triggers []string; mode string; caseSensitive,want bool }{
		{name:"exact",message:"群规",triggers:[]string{"群规"},mode:"exact",want:true},
		{name:"exact rejects suffix",message:"群规在哪里",triggers:[]string{"群规"},mode:"exact"},
		{name:"contains",message:"请问群规在哪里",triggers:[]string{"群规"},mode:"contains",want:true},
		{name:"prefix",message:"帮助一下",triggers:[]string{"帮助"},mode:"prefix",want:true},
		{name:"case insensitive",message:"HELP",triggers:[]string{"help"},mode:"exact",want:true},
		{name:"case sensitive",message:"HELP",triggers:[]string{"help"},mode:"exact",caseSensitive:true},
	}
	for _, tt := range tests {
		t.Run(tt.name,func(t *testing.T){
			if got:=matchesQARule(tt.message,tt.triggers,tt.mode,tt.caseSensitive);got!=tt.want { t.Fatalf("got %v want %v",got,tt.want) }
		})
	}
}

func TestValidGroupCategory(t *testing.T) {
	valid:=[]string{"qa","review","notices","basic","spam","ads","retract","admins","lists","verify","warnings","profile","schedule","polls","stats","settings","media"}
	for _,category:=range valid { if !validGroupCategory(category) { t.Errorf("category %q should be valid",category) } }
	for _,category:=range []string{"","unknown","members","group:settings","../admins"} { if validGroupCategory(category) { t.Errorf("category %q should be rejected",category) } }
}

func TestGroupRuleCategoryKeepsFeatureIsolation(t *testing.T) {
	tests:=[]struct{ruleType,want string}{
		{"welcome","notices"},{"leave_notice","notices"},
		{"anti_spam","spam"},
		{"anti_ad","ads"},{"keyword_filter","ads"},{"sensitive_word","ads"},{"media_restriction","media"},{"anti_bot","protection"},
		{"join_verification","verify"},
		{"unknown","all"},
	}
	for _,tt:=range tests {
		if got:=groupRuleCategory(tt.ruleType);got!=tt.want {
			t.Errorf("groupRuleCategory(%q)=%q want %q",tt.ruleType,got,tt.want)
		}
	}
}

func TestFormatJoinVerificationSummary(t *testing.T) {
	got:=formatJoinVerificationSummary([]byte(`{"challenge_type":"math","timeout_seconds":180,"failure_action":"kick","required_channels":["@news","-1001234567890"]}`))
	for _,part:=range []string{"数字加法选择验证","180 秒","移出群组","@news","-1001234567890"} {
		if !strings.Contains(got,part) { t.Errorf("summary %q does not contain %q",got,part) }
	}
	if got:=formatJoinVerificationSummary([]byte(`{invalid}`));got!="验证配置无法读取" { t.Errorf("invalid config summary=%q",got) }
}

func TestGenerateJoinChallengeModes(t *testing.T) {
	for _, mode := range []string{"math", "letters", "alphanumeric"} {
		answer, prompt, options, numeric, err := generateJoinChallenge(mode)
		if err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		if answer == "" || prompt == "" {
			t.Fatalf("%s returned empty challenge", mode)
		}
		if mode == "math" {
			if len(options) != 3 || numeric < 2 || numeric > 18 {
				t.Fatalf("math challenge options=%v numeric=%d", options, numeric)
			}
			continue
		}
		if len(answer) != 4 || len(options) != 0 || numeric != 2 || !strings.Contains(prompt, "tg-spoiler") {
			t.Fatalf("%s challenge answer=%q prompt=%q options=%v numeric=%d", mode, answer, prompt, options, numeric)
		}
		for _, r := range answer {
			if mode == "letters" && !strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ", r) {
				t.Fatalf("letters challenge contains invalid rune %q", r)
			}
			if mode == "alphanumeric" && !strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789", r) {
				t.Fatalf("alphanumeric challenge contains invalid rune %q", r)
			}
		}
	}
	if _, _, _, _, err := generateJoinChallenge("captcha"); err == nil {
		t.Fatal("unsupported CAPTCHA mode should fail closed")
	}
}

func TestTelegramMessageMediaType(t *testing.T) {
	tests := []struct {
		name string
		message *telegramMessage
		want string
	}{
		{name: "photo", message: &telegramMessage{Photo: []json.RawMessage{json.RawMessage(`{}`)}}, want: "photo"},
		{name: "video", message: &telegramMessage{Video: json.RawMessage(`{}`)}, want: "video"},
		{name: "animation", message: &telegramMessage{Animation: json.RawMessage(`{}`)}, want: "animation"},
		{name: "sticker", message: &telegramMessage{Sticker: json.RawMessage(`{}`)}, want: "sticker"},
		{name: "voice", message: &telegramMessage{Voice: json.RawMessage(`{}`)}, want: "voice"},
		{name: "audio", message: &telegramMessage{Audio: json.RawMessage(`{}`)}, want: "audio"},
		{name: "document", message: &telegramMessage{Document: json.RawMessage(`{}`)}, want: "document"},
		{name: "video note", message: &telegramMessage{VideoNote: json.RawMessage(`{}`)}, want: "video_note"},
		{name: "poll", message: &telegramMessage{Poll: json.RawMessage(`{}`)}, want: "poll"},
		{name: "contact", message: &telegramMessage{Contact: json.RawMessage(`{}`)}, want: "contact"},
		{name: "location", message: &telegramMessage{Location: json.RawMessage(`{}`)}, want: "location"},
		{name: "venue", message: &telegramMessage{Venue: json.RawMessage(`{}`)}, want: "venue"},
		{name: "text only", message: &telegramMessage{Text: "hello"}, want: ""},
		{name: "nil", message: nil, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := telegramMessageMediaType(tt.message); got != tt.want {
				t.Fatalf("media type = %q, want %q", got, tt.want)
			}
		})
	}
}


func TestMatchesBlockedLink(t *testing.T) {
	tests:=[]struct{name,message string;blocked,allowed []string;invites,want bool}{
		{name:"exact blocked domain",message:"visit https://spam.example/path",blocked:[]string{"spam.example"},want:true},
		{name:"blocked subdomain",message:"visit ads.spam.example now",blocked:[]string{"spam.example"},want:true},
		{name:"allowed domain exception",message:"visit https://safe.spam.example",blocked:[]string{"spam.example"},allowed:[]string{"safe.spam.example"},want:false},
		{name:"allowed parent domain",message:"visit https://news.safe.example",blocked:[]string{"example"},allowed:[]string{"safe.example"},want:false},
		{name:"Telegram invite",message:"join https://t.me/+AbCdEf",invites:true,want:true},
		{name:"Telegram public channel not invite",message:"https://t.me/channelname",invites:true,want:false},
		{name:"bare blocked domain",message:"spam.example",blocked:[]string{"spam.example"},want:true},
		{name:"ordinary text",message:"this is ordinary text",blocked:[]string{"spam.example"},want:false},
	}
	for _,test:=range tests { t.Run(test.name,func(t *testing.T){if got:=matchesBlockedLink(test.message,test.blocked,test.allowed,test.invites);got!=test.want{t.Fatalf("matchesBlockedLink()=%v want %v",got,test.want)}}) }
}


func TestIsAllowedBotUsername(t *testing.T) {
	if !isAllowedBotUsername("@Trusted_Bot",[]string{"trusted_bot"}) { t.Fatal("expected case-insensitive username allowlist match") }
	if isAllowedBotUsername("untrusted_bot",[]string{"trusted_bot"}) { t.Fatal("unexpected allowlist match") }
	if isAllowedBotUsername("",[]string{"trusted_bot"}) { t.Fatal("empty username must not match") }
}


func TestSelectSpamPenaltyLevel(t *testing.T) {
	levels:=[]spamRulePenalty{
		{Threshold:4,Action:"mute",DurationMinutes:10},
		{Threshold:12,Action:"ban"},
	}
	tests:=[]struct{count int;wantAction string;wantThreshold int}{
		{count:2,wantAction:"",wantThreshold:0},
		{count:4,wantAction:"mute",wantThreshold:4},
		{count:11,wantAction:"mute",wantThreshold:4},
		{count:12,wantAction:"ban",wantThreshold:12},
		{count:30,wantAction:"ban",wantThreshold:12},
	}
	for _,test:=range tests {
		got:=selectSpamPenaltyLevel(levels,test.count)
		if test.wantAction=="" { if got!=nil { t.Fatalf("count %d unexpectedly selected %+v",test.count,got) }; continue }
		if got==nil || got.Action!=test.wantAction || got.Threshold!=test.wantThreshold { t.Errorf("count %d selected %+v, want action=%s threshold=%d",test.count,got,test.wantAction,test.wantThreshold) }
	}
}

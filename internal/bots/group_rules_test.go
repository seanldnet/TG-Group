package bots

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateGroupRule(t *testing.T) {
	tests := []struct {
		name string
		input GroupRuleInput
		valid bool
	}{
		{"keyword", GroupRuleInput{RuleType:"keyword_filter",Name:"关键词过滤",Enabled:true,Priority:10,Config:json.RawMessage(`{"words":["广告","spam"],"action":"delete"}`)},true},
		{"sensitive", GroupRuleInput{RuleType:"sensitive_word",Name:"敏感词",Config:json.RawMessage(`{"words":["bad"],"action":"delete"}`)},true},
		{"spam", GroupRuleInput{RuleType:"anti_spam",Name:"刷屏",Config:json.RawMessage(`{"max_messages":5,"window_seconds":10,"action":"delete"}`)},true},
		{"spam duplicate threshold", GroupRuleInput{RuleType:"anti_spam",Name:"重复消息",Config:json.RawMessage(`{"max_messages":5,"window_seconds":10,"duplicate_count":3,"action":"delete"}`)},true},
		{"spam duplicate threshold too low", GroupRuleInput{RuleType:"anti_spam",Name:"重复消息",Config:json.RawMessage(`{"max_messages":5,"window_seconds":10,"duplicate_count":1,"action":"delete"}`)},false},
		{"spam duplicate threshold too high", GroupRuleInput{RuleType:"anti_spam",Name:"重复消息",Config:json.RawMessage(`{"max_messages":5,"window_seconds":10,"duplicate_count":101,"action":"delete"}`)},false},
		{"anti bot", GroupRuleInput{RuleType:"anti_bot",Name:"防机器人",Config:json.RawMessage(`{"allowed_usernames":["trusted_bot"],"action":"ban"}`)},true},
		{"anti bot invalid username", GroupRuleInput{RuleType:"anti_bot",Name:"防机器人",Config:json.RawMessage(`{"allowed_usernames":["bad"],"action":"ban"}`)},false},
		{"anti bot duplicate username", GroupRuleInput{RuleType:"anti_bot",Name:"防机器人",Config:json.RawMessage(`{"allowed_usernames":["Trusted_Bot","trusted_bot"],"action":"ban"}`)},false},
		{"anti bot unsupported action", GroupRuleInput{RuleType:"anti_bot",Name:"防机器人",Config:json.RawMessage(`{"action":"kick"}`)},false},
		{"ad", GroupRuleInput{RuleType:"anti_ad",Name:"广告",Config:json.RawMessage(`{"patterns":["t.me/"],"action":"delete"}`)},true},
		{"ad exceptions", GroupRuleInput{RuleType:"anti_ad",Name:"广告白名单",Config:json.RawMessage(`{"patterns":["t.me/"],"exceptions":["official.example"],"action":"delete"}`)},true},
		{"ad blocked domains", GroupRuleInput{RuleType:"anti_ad",Name:"域名拦截",Config:json.RawMessage(`{"patterns":["广告"],"blocked_domains":["spam.example","bad.example"],"block_invite_links":true,"action":"delete"}`)},true},
		{"ad invalid domain", GroupRuleInput{RuleType:"anti_ad",Name:"非法域名",Config:json.RawMessage(`{"patterns":["广告"],"blocked_domains":["https://spam.example"],"action":"delete"}`)},false},
		{"ad duplicate domains", GroupRuleInput{RuleType:"anti_ad",Name:"重复域名",Config:json.RawMessage(`{"patterns":["广告"],"blocked_domains":["spam.example","spam.example"],"action":"delete"}`)},false},
		{"ad invalid exceptions", GroupRuleInput{RuleType:"anti_ad",Name:"广告白名单",Config:json.RawMessage(`{"patterns":["t.me/"],"exceptions":[""],"action":"delete"}`)},false},
		{"media restriction", GroupRuleInput{RuleType:"media_restriction",Name:"媒体限制",Config:json.RawMessage(`{"types":["photo","video","document"],"action":"delete"}`)},true},
		{"media restriction invalid type", GroupRuleInput{RuleType:"media_restriction",Name:"媒体限制",Config:json.RawMessage(`{"types":["executable"],"action":"delete"}`)},false},
		{"media restriction duplicate type", GroupRuleInput{RuleType:"media_restriction",Name:"媒体限制",Config:json.RawMessage(`{"types":["photo","photo"],"action":"delete"}`)},false},
		{"welcome", GroupRuleInput{RuleType:"welcome",Name:"欢迎",Config:json.RawMessage(`{"text":"欢迎加入"}`)},true},
		{"leave", GroupRuleInput{RuleType:"leave_notice",Name:"离群",Config:json.RawMessage(`{"text":"成员离开"}`)},true},
		{"join verification", GroupRuleInput{RuleType:"join_verification",Name:"入群验证",Config:json.RawMessage(`{"challenge_type":"math","timeout_seconds":120,"failure_action":"kick"}`)},true},
		{"join verification letters", GroupRuleInput{RuleType:"join_verification",Name:"字母验证",Config:json.RawMessage(`{"challenge_type":"letters","timeout_seconds":120,"failure_action":"restrict"}`)},true},
		{"join verification alphanumeric", GroupRuleInput{RuleType:"join_verification",Name:"组合验证",Config:json.RawMessage(`{"challenge_type":"alphanumeric","timeout_seconds":120,"failure_action":"restrict"}`)},true},
		{"join verification unsupported captcha", GroupRuleInput{RuleType:"join_verification",Name:"图片验证",Config:json.RawMessage(`{"challenge_type":"captcha","timeout_seconds":120,"failure_action":"restrict"}`)},false},
		{"join verification required channels", GroupRuleInput{RuleType:"join_verification",Name:"频道验证",Config:json.RawMessage(`{"challenge_type":"math","timeout_seconds":120,"failure_action":"kick","required_channels":["@tg_group_news","-1001234567890"]}`)},true},
		{"join verification invalid channel", GroupRuleInput{RuleType:"join_verification",Name:"频道验证",Config:json.RawMessage(`{"challenge_type":"math","timeout_seconds":120,"failure_action":"kick","required_channels":["https://t.me/not-a-chat"]}`)},false},
		{"join verification duplicate channels", GroupRuleInput{RuleType:"join_verification",Name:"频道验证",Config:json.RawMessage(`{"challenge_type":"math","timeout_seconds":120,"failure_action":"kick","required_channels":["@tg_group_news","@tg_group_news"]}`)},false},
		{"join verification timeout too short", GroupRuleInput{RuleType:"join_verification",Name:"入群验证",Config:json.RawMessage(`{"challenge_type":"math","timeout_seconds":5,"failure_action":"kick"}`)},false},
		{"join verification unsafe failure action", GroupRuleInput{RuleType:"join_verification",Name:"入群验证",Config:json.RawMessage(`{"challenge_type":"math","timeout_seconds":120,"failure_action":"ban"}`)},false},
		{"unknown type", GroupRuleInput{RuleType:"arbitrary",Name:"x",Config:json.RawMessage(`{}`)},false},
		{"unknown config field", GroupRuleInput{RuleType:"welcome",Name:"x",Config:json.RawMessage(`{"text":"hi","unsafe":true}`)},false},
		{"empty words", GroupRuleInput{RuleType:"keyword_filter",Name:"x",Config:json.RawMessage(`{"words":[],"action":"delete"}`)},false},
		{"unsafe action", GroupRuleInput{RuleType:"keyword_filter",Name:"x",Config:json.RawMessage(`{"words":["x"],"action":"ban"}`)},false},
		{"invalid spam window", GroupRuleInput{RuleType:"anti_spam",Name:"x",Config:json.RawMessage(`{"max_messages":1,"window_seconds":10,"action":"delete"}`)},false},
		{"empty notice", GroupRuleInput{RuleType:"welcome",Name:"x",Config:json.RawMessage(`{"text":" "}`)},false},
		{"valid warning escalation", GroupRuleInput{RuleType:"warning_escalation",Name:"警告递进",Config:json.RawMessage(`{"levels":[{"threshold":3,"action":"mute","duration_minutes":30},{"threshold":5,"action":"ban"}]}`)},true},
		{"warning escalation unsorted", GroupRuleInput{RuleType:"warning_escalation",Name:"警告递进",Config:json.RawMessage(`{"levels":[{"threshold":5,"action":"ban"},{"threshold":3,"action":"mute","duration_minutes":30}]}`)},false},
		{"warning escalation unsafe duration", GroupRuleInput{RuleType:"warning_escalation",Name:"警告递进",Config:json.RawMessage(`{"levels":[{"threshold":2,"action":"mute","duration_minutes":0}]}`)},false},
		{"warning escalation unknown action", GroupRuleInput{RuleType:"warning_escalation",Name:"警告递进",Config:json.RawMessage(`{"levels":[{"threshold":2,"action":"delete"}]}`)},false},
		{"valid QA exact", GroupRuleInput{RuleType:"qa",Name:"faq",Config:json.RawMessage(`{"triggers":["群规"],"match_mode":"exact","response":"请看公告","case_sensitive":false}`)},true},
		{"valid QA contains", GroupRuleInput{RuleType:"qa",Name:"faq",Config:json.RawMessage(`{"triggers":["群规","规则"],"match_mode":"contains","response":"请看公告","case_sensitive":false}`)},true},
		{"invalid QA match mode", GroupRuleInput{RuleType:"qa",Name:"faq",Config:json.RawMessage(`{"triggers":["群规"],"match_mode":"regex","response":"请看公告","case_sensitive":false}`)},false},
		{"empty QA response", GroupRuleInput{RuleType:"qa",Name:"faq",Config:json.RawMessage(`{"triggers":["群规"],"match_mode":"exact","response":" ","case_sensitive":false}`)},false},
		{"priority out of range", GroupRuleInput{RuleType:"welcome",Name:"x",Priority:1001,Config:json.RawMessage(`{"text":"hi"}`)},false},
	}
	for _,tt:=range tests {
		t.Run(tt.name,func(t *testing.T){
			err:=ValidateGroupRule(tt.input)
			if tt.valid && err!=nil { t.Fatalf("unexpected error: %v",err) }
			if !tt.valid && !errors.Is(err,ErrInvalidGroupRule) { t.Fatalf("error = %v, want ErrInvalidGroupRule",err) }
		})
	}
}

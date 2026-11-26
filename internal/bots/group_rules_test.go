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
		{"ad", GroupRuleInput{RuleType:"anti_ad",Name:"广告",Config:json.RawMessage(`{"patterns":["t.me/"],"action":"delete"}`)},true},
		{"welcome", GroupRuleInput{RuleType:"welcome",Name:"欢迎",Config:json.RawMessage(`{"text":"欢迎加入"}`)},true},
		{"leave", GroupRuleInput{RuleType:"leave_notice",Name:"离群",Config:json.RawMessage(`{"text":"成员离开"}`)},true},
		{"join verification", GroupRuleInput{RuleType:"join_verification",Name:"入群验证",Config:json.RawMessage(`{"challenge_type":"math","timeout_seconds":120,"failure_action":"kick"}`)},true},
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

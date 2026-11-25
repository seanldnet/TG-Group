package telegramruntime

import "testing"

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

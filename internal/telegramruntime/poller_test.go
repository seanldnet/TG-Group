package telegramruntime

import "testing"

func TestChatFromUpdate(t *testing.T) {
	group := chat{ID:-100123,Type:"supergroup",Title:"TG Group",IsForum:true}
	item := update{Message:&struct{Chat chat `json:"chat"`}{Chat:group}}
	got:=chatFromUpdate(item)
	if got==nil || got.ID!=group.ID || got.Title!=group.Title { t.Fatalf("chatFromUpdate() = %#v, want group %#v",got,group) }
}

func TestIsManagedChat(t *testing.T) {
	tests:=[]struct{name string; value *chat; want bool}{
		{name:"nil",value:nil,want:false},
		{name:"private",value:&chat{ID:1,Type:"private"},want:false},
		{name:"group",value:&chat{ID:-1,Type:"group"},want:true},
		{name:"supergroup",value:&chat{ID:-100,Type:"supergroup"},want:true},
		{name:"channel",value:&chat{ID:-200,Type:"channel"},want:true},
		{name:"zero id",value:&chat{ID:0,Type:"group"},want:false},
	}
	for _,test:=range tests { t.Run(test.name,func(t *testing.T){if got:=isManagedChat(test.value);got!=test.want{t.Fatalf("isManagedChat() = %v, want %v",got,test.want)}}) }
}

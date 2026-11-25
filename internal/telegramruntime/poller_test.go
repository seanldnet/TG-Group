package telegramruntime

import ("testing"; "encoding/json")

func TestChatFromUpdate(t *testing.T) {
	group := chat{ID:-100123,Type:"supergroup",Title:"TG Group",IsForum:true}
	item := update{Message:&telegramMessage{Chat:group}}
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

func TestGroupStatusFromMembershipUpdate(t *testing.T) {
	tests:=[]struct{name,status,want string}{
		{name:"member",status:"member",want:"active"},
		{name:"administrator",status:"administrator",want:"active"},
		{name:"left",status:"left",want:"left"},
		{name:"kicked",status:"kicked",want:"kicked"},
	}
	for _,test:=range tests {
		t.Run(test.name,func(t *testing.T){
			item:=update{MyChatMember:&struct{
				Chat chat `json:"chat"`
				NewChatMember struct{Status string `json:"status"`} `json:"new_chat_member"`
			}{NewChatMember:struct{Status string `json:"status"`}{Status:test.status}}}
			if got:=groupStatus(item);got!=test.want{t.Fatalf("groupStatus() = %q, want %q",got,test.want)}
		})
	}
}

func TestMessageType(t *testing.T) {
	tests:=[]struct{name string; message telegramMessage; want string}{
		{name:"text",message:telegramMessage{Text:"hello"},want:"text"},
		{name:"photo",message:telegramMessage{Photo:[]json.RawMessage{json.RawMessage("{}")}},want:"photo"},
		{name:"document",message:telegramMessage{Document:json.RawMessage("{}")},want:"document"},
		{name:"caption only",message:telegramMessage{Caption:"caption"},want:"other"},
	}
	for _,test:=range tests{t.Run(test.name,func(t *testing.T){if got:=messageType(test.message);got!=test.want{t.Fatalf("messageType() = %q, want %q",got,test.want)}})}
}

func TestSenderDisplayName(t *testing.T) {
	tests:=[]struct{name string; sender *telegramSender; want string}{
		{name:"full",sender:&telegramSender{FirstName:"Ada",LastName:"Lovelace"},want:"Ada Lovelace"},
		{name:"first",sender:&telegramSender{FirstName:"Ada"},want:"Ada"},
		{name:"nil",sender:nil,want:""},
	}
	for _,test:=range tests{t.Run(test.name,func(t *testing.T){if got:=senderDisplayName(test.sender);got!=test.want{t.Fatalf("senderDisplayName() = %q, want %q",got,test.want)}})}
}

func TestChatMemberUpdateDecoding(t *testing.T) {
	raw := `{"update_id":12,"chat_member":{"chat":{"id":-100123,"type":"supergroup","title":"Test"},"date":1700000000,"old_chat_member":{"user":{"id":42,"is_bot":false,"first_name":"Ada"},"status":"left"},"new_chat_member":{"user":{"id":42,"is_bot":false,"first_name":"Ada","username":"ada"},"status":"administrator","can_delete_messages":true,"can_restrict_members":true}}}`
	var item update
	if err := json.Unmarshal([]byte(raw), &item); err != nil { t.Fatalf("decode chat_member update: %v", err) }
	if got := chatFromUpdate(item); got == nil || got.ID != -100123 { t.Fatalf("chatFromUpdate() = %#v", got) }
	if item.ChatMember == nil { t.Fatal("chat_member update was not decoded") }
	member := item.ChatMember.NewChatMember
	if member.User.ID != 42 || member.Status != "administrator" || !member.CanDeleteMessages || !member.CanRestrictMembers {
		t.Fatalf("unexpected member update: %+v", member)
	}
	if item.ChatMember.Date != 1700000000 { t.Fatalf("date = %d", item.ChatMember.Date) }
}

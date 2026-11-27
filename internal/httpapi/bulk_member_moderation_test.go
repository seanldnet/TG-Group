package httpapi

import "testing"

func TestValidateBulkMemberAction(t *testing.T) {
	tests := []struct {
		name string
		input bulkMemberActionRequest
		wantErr bool
	}{
		{name:"valid ban",input:bulkMemberActionRequest{UserIDs:[]int64{10,20},Action:"ban"}},
		{name:"valid mute",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"mute",DurationMinutes:60}},
		{name:"empty selection",input:bulkMemberActionRequest{Action:"ban"},wantErr:true},
		{name:"too many members",input:bulkMemberActionRequest{UserIDs:makeIDs(51),Action:"ban"},wantErr:true},
		{name:"zero user id",input:bulkMemberActionRequest{UserIDs:[]int64{0},Action:"ban"},wantErr:true},
		{name:"negative user id",input:bulkMemberActionRequest{UserIDs:[]int64{-1},Action:"ban"},wantErr:true},
		{name:"duplicate user id",input:bulkMemberActionRequest{UserIDs:[]int64{10,10},Action:"ban"},wantErr:true},
		{name:"unknown action",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"kick"},wantErr:true},
		{name:"mute duration missing",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"mute"},wantErr:true},
		{name:"mute duration too long",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"mute",DurationMinutes:525601},wantErr:true},
		{name:"duration on ban",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"ban",DurationMinutes:10},wantErr:true},
	}
	for _,tt:=range tests {
		t.Run(tt.name,func(t *testing.T){
			err:=validateBulkMemberAction(tt.input)
			if (err!=nil)!=tt.wantErr { t.Fatalf("validateBulkMemberAction() error = %v, wantErr %v",err,tt.wantErr) }
		})
	}
}

func makeIDs(n int) []int64 {
	ids:=make([]int64,n)
	for i:=range ids { ids[i]=int64(i+1) }
	return ids
}

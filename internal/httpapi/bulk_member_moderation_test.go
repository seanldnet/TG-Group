package httpapi

import (
	"errors"
	"reflect"
	"testing"
)

func TestExecuteBulkMemberActionContinuesAfterFailures(t *testing.T) {
	calls:=make([]int64,0,4)
	results,succeeded:=executeBulkMemberAction([]int64{11,22,33,44},func(id int64) error {
		calls=append(calls,id)
		if id==22 || id==44 { return errors.New("telegram rejected action") }
		return nil
	})
	if !reflect.DeepEqual(calls,[]int64{11,22,33,44}) { t.Fatalf("calls = %v",calls) }
	if succeeded!=2 { t.Fatalf("succeeded = %d, want 2",succeeded) }
	if len(results)!=4 { t.Fatalf("results length = %d, want 4",len(results)) }
	if !results[0].OK || results[0].Error!="" { t.Fatalf("first result = %+v",results[0]) }
	if results[1].OK || results[1].Error=="" { t.Fatalf("second result = %+v",results[1]) }
	if !results[2].OK || results[3].OK || results[3].Error=="" { t.Fatalf("last results = %+v / %+v",results[2],results[3]) }
}

func TestValidateBulkMemberAction(t *testing.T) {
	tests := []struct {
		name string
		input bulkMemberActionRequest
		wantErr bool
	}{
		{name:"valid ban",input:bulkMemberActionRequest{UserIDs:[]int64{10,20},Action:"ban"}},
		{name:"valid unban",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"unban"}},
		{name:"valid mute",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"mute",DurationMinutes:60}},
		{name:"valid unmute",input:bulkMemberActionRequest{UserIDs:[]int64{10},Action:"unmute"}},
		{name:"valid maximum batch",input:bulkMemberActionRequest{UserIDs:makeIDs(50),Action:"ban"}},
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

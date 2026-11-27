package payments

import "testing"

func TestParseIntegerUSDT(t *testing.T){
	tests:=[]struct{value string;want int64;wantErr bool}{
		{value:"1",want:1},
		{value:"100.000000",want:100},
		{value:"00042",want:42},
		{value:"0",wantErr:true},
		{value:"0.5",wantErr:true},
		{value:"14.29",wantErr:true},
		{value:"-1",wantErr:true},
		{value:"9223372036854775808",wantErr:true},
		{value:"not-a-number",wantErr:true},
		{value:"",wantErr:true},
	}
	for _,tt:=range tests{t.Run(tt.value,func(t *testing.T){got,err:=ParseIntegerUSDT(tt.value);if (err!=nil)!=tt.wantErr{t.Fatalf("ParseIntegerUSDT() error=%v wantErr=%v",err,tt.wantErr)};if err==nil&&got!=tt.want{t.Fatalf("ParseIntegerUSDT()=%d want=%d",got,tt.want)}})}
}

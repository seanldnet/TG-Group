package auth

import "testing"

func TestValidatePlan(t *testing.T) {
	tests:=[]struct{name,code,planName,description string;price int64;days,bots,groups,sort int32;wantErr bool}{
		{name:"valid",code:"starter",planName:"入门套餐",price:100,days:30,bots:2,groups:5,sort:1},
		{name:"zero price",code:"free",planName:"免费套餐",price:0,days:365,bots:0,groups:0,sort:0},
		{name:"negative price",code:"bad",planName:"套餐",price:-1,days:30,wantErr:true},
		{name:"zero duration",code:"bad",planName:"套餐",price:1,days:0,wantErr:true},
		{name:"too long duration",code:"bad",planName:"套餐",price:1,days:36501,wantErr:true},
		{name:"negative bot limit",code:"bad",planName:"套餐",price:1,days:30,bots:-1,wantErr:true},
		{name:"negative group limit",code:"bad",planName:"套餐",price:1,days:30,groups:-1,wantErr:true},
		{name:"empty name",code:"good",planName:"  ",price:1,days:30,wantErr:true},
		{name:"invalid code",code:"bad code",planName:"套餐",price:1,days:30,wantErr:true},
		{name:"description too long",code:"long",planName:"套餐",description:string(make([]byte,2001)),price:1,days:30,wantErr:true},
	}
	for _,tt:=range tests { t.Run(tt.name,func(t *testing.T){err:=validatePlan(tt.code,tt.planName,tt.description,tt.price,tt.days,tt.bots,tt.groups,tt.sort);if (err!=nil)!=tt.wantErr{t.Fatalf("validatePlan() error = %v, wantErr %v",err,tt.wantErr)}}) }
}

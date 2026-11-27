package payments

import (
	"strconv"
	"testing"
)

func TestVerifyGMPayCallback(t *testing.T){
	params:=map[string]string{"pid":"1000","trade_id":"trade-1","order_id":"RC123","amount":"25","actual_amount":"25.000","token":"USDT","block_transaction_id":"0xabc","status":"2"}
	params["signature"]=GMPaySignature(params,"secret")
	got,err:=VerifyGMPayCallback(params,"secret","1000","RC123",25)
	if err!=nil{t.Fatal(err)}
	if got.CreditedUSDT!=25||got.TransactionID!="0xabc"{t.Fatalf("unexpected callback: %#v",got)}
}

func TestVerifyGMPayCallbackRejectsInvalidSettlements(t *testing.T){
	base:=map[string]string{"pid":"1000","trade_id":"trade-1","order_id":"RC123","amount":"25","actual_amount":"25","token":"USDT","block_transaction_id":"0xabc","status":"2"}
	tests:=[]struct{name string;change func(map[string]string)}{
		{name:"fractional actual amount",change:func(p map[string]string){p["actual_amount"]="24.99"}},
		{name:"amount mismatch",change:func(p map[string]string){p["amount"]="26"}},
		{name:"wrong PID",change:func(p map[string]string){p["pid"]="1001"}},
		{name:"wrong order",change:func(p map[string]string){p["order_id"]="other"}},
		{name:"wrong token",change:func(p map[string]string){p["token"]="TRX"}},
		{name:"not paid",change:func(p map[string]string){p["status"]="5"}},
		{name:"missing transaction",change:func(p map[string]string){p["block_transaction_id"]=""}},
		{name:"tampered after signing",change:func(p map[string]string){p["actual_amount"]="999"}},
	}
	for _,tt:=range tests{t.Run(tt.name,func(t *testing.T){p:=map[string]string{};for k,v:=range base{p[k]=v};p["signature"]=GMPaySignature(p,"secret");tt.change(p);if _,err:=VerifyGMPayCallback(p,"secret","1000","RC123",25);err==nil{t.Fatal("invalid callback accepted")}})}
}

func TestVerifyGMPayCallbackRejectsIntegerOverflow(t *testing.T){
	p:=map[string]string{"pid":"1000","trade_id":"trade-1","order_id":"RC123","amount":"25","actual_amount":strconv.FormatUint(^uint64(0),10),"token":"USDT","block_transaction_id":"0xabc","status":"2"}
	p["signature"]=GMPaySignature(p,"secret")
	if _,err:=VerifyGMPayCallback(p,"secret","1000","RC123",25);err==nil{t.Fatal("overflow settlement accepted")}
}


func TestVerifyGMPayCallbackSeparatesFractionalSettlementForReview(t *testing.T){
	params:=map[string]string{"pid":"1000","trade_id":"trade-1","order_id":"RC123","amount":"25","actual_amount":"24.99","token":"USDT","block_transaction_id":"0xabc","status":"2"}
	params["signature"]=GMPaySignature(params,"secret")
	verified,err:=VerifyGMPayCallback(params,"secret","1000","RC123",25)
	if err!=ErrNonIntegerUSDT{t.Fatalf("error=%v, want ErrNonIntegerUSDT",err)}
	if verified.TradeID!="trade-1"||verified.TransactionID!="0xabc"{t.Fatalf("signed fractional callback metadata lost: %#v",verified)}
}

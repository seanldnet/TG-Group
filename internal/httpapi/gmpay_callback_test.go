package httpapi

import "testing"

func TestDecodeGMPayCallbackParamsCanonicalizesJSONNumbers(t *testing.T){
	params,err:=decodeGMPayCallbackParams([]byte(`{"pid":"1000","amount":100.00,"actual_amount":14.2900,"status":2,"nullable":null}`))
	if err!=nil{t.Fatal(err)}
	if params["amount"]!="100"||params["actual_amount"]!="14.29"||params["status"]!="2"{t.Fatalf("unexpected canonical callback params: %#v",params)}
	if _,ok:=params["nullable"];ok{t.Fatal("null field should be omitted")}
}

func TestDecodeGMPayCallbackParamsRejectsUnsupportedAndTrailingJSON(t *testing.T){
	for _,raw:=range []string{`{"amount":[1,2]}`,`{"amount":1}{"amount":2}`,`[]`,`{"amount":NaN}`}{
		if _,err:=decodeGMPayCallbackParams([]byte(raw));err==nil{t.Errorf("accepted invalid callback JSON %s",raw)}
	}
}

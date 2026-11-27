package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestGMPayClientCreateOrderSignsRequestAndParsesResponse(t *testing.T){
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.URL.Path!="/payments/gmpay/v1/order/create-transaction"{t.Fatalf("unexpected path %s",r.URL.Path)}
		var body map[string]json.RawMessage
		if err:=json.NewDecoder(r.Body).Decode(&body);err!=nil{t.Fatal(err)}
		params:=map[string]string{}
		for key,value:=range body{if key=="signature"{continue};var s string;if err:=json.Unmarshal(value,&s);err==nil{params[key]=s}else{params[key]=string(value)}}
		var signature string
		if err:=json.Unmarshal(body["signature"],&signature);err!=nil{t.Fatal(err)}
		if signature!=GMPaySignature(params,"0123456789abcdef"){t.Fatal("request signature mismatch")}
		if params["amount"]!=strconv.Itoa(25)||params["currency"]!="usd"{t.Fatalf("unexpected amount/currency: %#v",params)}
		w.Header().Set("Content-Type","application/json")
		_,_=w.Write([]byte(`{"status_code":200,"message":"success","data":{"trade_id":"trade-1","order_id":"RC123","actual_amount":25,"token":"USDT","status":1,"expiration_time":1779530812,"payment_url":"https://checkout.example/pay/1","receive_address":"TAddress"}}`))
	}))
	defer server.Close()
	channel:=ChannelConfig{Code:"epusdt_gmpay",BaseURL:server.URL,MerchantPID:"1000",SecretKey:"0123456789abcdef",Currency:"usd",Token:"usdt",Network:"tron"}
	channel.BaseURL="https://provider.example"
	// httptest uses HTTP; the test client overrides transport only while preserving the HTTPS endpoint.
	channel.BaseURL="https://provider.example"
	client:=GMPayClient{HTTP:&http.Client{Transport:rewriteHostTransport{target:server.URL}}}
	got,err:=client.CreateOrder(context.Background(),channel,"RC123",25,"https://merchant.example/notify","")
	if err!=nil{t.Fatal(err)}
	if got.TradeID!="trade-1"||got.ActualAmount!="25"||got.Status!=1{t.Fatalf("unexpected provider result: %#v",got)}
}

type rewriteHostTransport struct{target string}
func (t rewriteHostTransport) RoundTrip(r *http.Request)(*http.Response,error){
	clone:=r.Clone(r.Context())
	clone.URL.Scheme="http"
	clone.URL.Host=strings.TrimPrefix(t.target,"http://")
	return http.DefaultTransport.RoundTrip(clone)
}

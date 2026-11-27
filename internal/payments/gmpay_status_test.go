package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGMPayClientCheckOrderStatus(t *testing.T){
	server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.URL.Path!="/pay/check-status/trade-123"{t.Fatalf("unexpected path %s",r.URL.Path)}
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code":200,"data":map[string]any{"trade_id":"trade-123","status":5}})
	}))
	defer server.Close()
	client:=GMPayClient{HTTP:server.Client()}
	status,err:=client.CheckOrderStatus(context.Background(),server.URL,"trade-123")
	if err!=nil{t.Fatal(err)}
	if status!=5{t.Fatalf("status=%d want 5",status)}
}

func TestGMPayClientCheckOrderStatusRejectsMismatchedTrade(t *testing.T){
	server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code":200,"data":map[string]any{"trade_id":"other","status":2}})
	}))
	defer server.Close()
	client:=GMPayClient{HTTP:server.Client()}
	if _,err:=client.CheckOrderStatus(context.Background(),server.URL,"trade-123");err==nil{t.Fatal("mismatched trade status accepted")}
}

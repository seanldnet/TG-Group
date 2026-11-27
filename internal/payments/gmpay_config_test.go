package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGMPayClientReadsSupportedAssets(t *testing.T){
	server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Method!=http.MethodGet||r.URL.Path!="/payments/gmpay/v1/config"{t.Fatalf("unexpected request %s %s",r.Method,r.URL.Path)}
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code":200,"data":map[string]any{"supported_assets":[]any{map[string]any{"network":"tron","display_name":"TRON","tokens":[]string{"TRX","USDT"}}}}})
	}))
	defer server.Close()
	client:=GMPayClient{HTTP:server.Client()}
	config,err:=client.GetPublicConfig(context.Background(),server.URL)
	if err!=nil{t.Fatal(err)}
	if len(config.SupportedAssets)!=1||config.SupportedAssets[0].Network!="tron"{t.Fatalf("unexpected supported assets: %#v",config.SupportedAssets)}
}

func TestGMPayClientRejectsInsecurePublicConfigURL(t *testing.T){
	if _,err:=(GMPayClient{}).GetPublicConfig(context.Background(),"http://provider.example");err==nil{t.Fatal("HTTP config URL accepted")}
}

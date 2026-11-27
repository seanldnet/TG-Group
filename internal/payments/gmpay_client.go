package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrProviderRejected = errors.New("EPUSDT rejected payment order")

type GMPayOrderRequest struct {
	PID string `json:"pid"`
	OrderID string `json:"order_id"`
	Currency string `json:"currency"`
	Token string `json:"token,omitempty"`
	Network string `json:"network,omitempty"`
	Amount int64 `json:"amount"`
	NotifyURL string `json:"notify_url"`
	RedirectURL string `json:"redirect_url,omitempty"`
	Name string `json:"name,omitempty"`
	Signature string `json:"signature"`
}

type GMPayOrderResult struct {
	TradeID string
	OrderID string
	ActualAmount string
	Token string
	Status int
	ExpirationTime int64
	PaymentURL string
	ReceiveAddress string
}

type gmpayEnvelope struct {
	StatusCode int `json:"status_code"`
	Message string `json:"message"`
	Data struct {
		TradeID string `json:"trade_id"`
		OrderID string `json:"order_id"`
		ActualAmount json.Number `json:"actual_amount"`
		Token string `json:"token"`
		Status int `json:"status"`
		ExpirationTime int64 `json:"expiration_time"`
		PaymentURL string `json:"payment_url"`
		ReceiveAddress string `json:"receive_address"`
	} `json:"data"`
}

type GMPayClient struct { HTTP *http.Client }

func (c GMPayClient) CreateOrder(ctx context.Context,channel ChannelConfig,orderID string,amountUSDT int64,notifyURL,redirectURL string)(GMPayOrderResult,error){
	if channel.Code!="epusdt_gmpay" || !channel.Enabled || channel.Currency!="usd" || amountUSDT<=0 || len(orderID)>32 || strings.TrimSpace(orderID)=="" {
		return GMPayOrderResult{},ErrInvalidChannelConfig
	}
	notify,err:=url.Parse(notifyURL)
	if err!=nil || notify.Scheme!="https" || notify.Host=="" { return GMPayOrderResult{},ErrInvalidChannelConfig }
	if redirectURL!="" { u,e:=url.Parse(redirectURL);if e!=nil||u.Scheme!="https"||u.Host==""{return GMPayOrderResult{},ErrInvalidChannelConfig} }
	reqData:=GMPayOrderRequest{PID:channel.MerchantPID,OrderID:orderID,Currency:channel.Currency,Token:channel.Token,Network:channel.Network,Amount:amountUSDT,NotifyURL:notifyURL,RedirectURL:redirectURL,Name:"USDT wallet recharge"}
	params:=map[string]string{"pid":reqData.PID,"order_id":reqData.OrderID,"currency":reqData.Currency,"token":reqData.Token,"network":reqData.Network,"amount":strconv.FormatInt(reqData.Amount,10),"notify_url":reqData.NotifyURL,"name":reqData.Name}
	if reqData.RedirectURL!="" { params["redirect_url"]=reqData.RedirectURL }
	reqData.Signature=GMPaySignature(params,channel.SecretKey)
	body,err:=json.Marshal(reqData);if err!=nil{return GMPayOrderResult{},fmt.Errorf("encode GMPay request: %w",err)}
	endpoint:=strings.TrimRight(channel.BaseURL,"/")+"/payments/gmpay/v1/order/create-transaction"
	httpReq,err:=http.NewRequestWithContext(ctx,http.MethodPost,endpoint,bytes.NewReader(body));if err!=nil{return GMPayOrderResult{},fmt.Errorf("build GMPay request: %w",err)}
	httpReq.Header.Set("Content-Type","application/json")
	client:=c.HTTP;if client==nil{client=&http.Client{Timeout:12*time.Second}}
	response,err:=client.Do(httpReq);if err!=nil{return GMPayOrderResult{},fmt.Errorf("create GMPay order: %w",err)}
	defer response.Body.Close()
	payload,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return GMPayOrderResult{},fmt.Errorf("read GMPay response: %w",err)}
	var envelope gmpayEnvelope
	decoder:=json.NewDecoder(bytes.NewReader(payload));decoder.UseNumber()
	if err:=decoder.Decode(&envelope);err!=nil{return GMPayOrderResult{},fmt.Errorf("decode GMPay response: %w",err)}
	if response.StatusCode!=http.StatusOK||envelope.StatusCode!=200||envelope.Data.TradeID==""||envelope.Data.OrderID!=orderID {
		return GMPayOrderResult{},fmt.Errorf("%w: HTTP %d: %s",ErrProviderRejected,response.StatusCode,envelope.Message)
	}
	paymentURL,err:=url.Parse(envelope.Data.PaymentURL)
	if err!=nil||paymentURL.Scheme!="https"||paymentURL.Host==""||paymentURL.User!=nil { return GMPayOrderResult{},errors.New("EPUSDT returned an invalid payment URL") }
	return GMPayOrderResult{TradeID:envelope.Data.TradeID,OrderID:envelope.Data.OrderID,ActualAmount:envelope.Data.ActualAmount.String(),Token:envelope.Data.Token,Status:envelope.Data.Status,ExpirationTime:envelope.Data.ExpirationTime,PaymentURL:envelope.Data.PaymentURL,ReceiveAddress:envelope.Data.ReceiveAddress},nil
}


type SupportedAsset struct {
	Network string `json:"network"`
	DisplayName string `json:"display_name"`
	Tokens []string `json:"tokens"`
}

type PublicConfig struct {
	SupportedAssets []SupportedAsset `json:"supported_assets"`
}

func (c GMPayClient) GetPublicConfig(ctx context.Context,baseURL string)(PublicConfig,error){
	u,err:=url.Parse(strings.TrimRight(baseURL,"/")+"/payments/gmpay/v1/config")
	if err!=nil||u.Scheme!="https"||u.Host==""||u.User!=nil{return PublicConfig{},ErrInvalidChannelConfig}
	httpReq,err:=http.NewRequestWithContext(ctx,http.MethodGet,u.String(),nil)
	if err!=nil{return PublicConfig{},fmt.Errorf("build GMPay config request: %w",err)}
	client:=c.HTTP;if client==nil{client=&http.Client{Timeout:8*time.Second}}
	response,err:=client.Do(httpReq);if err!=nil{return PublicConfig{},fmt.Errorf("read GMPay public config: %w",err)}
	defer response.Body.Close()
	payload,err:=io.ReadAll(io.LimitReader(response.Body,1<<20));if err!=nil{return PublicConfig{},fmt.Errorf("read GMPay config response: %w",err)}
	var envelope struct{StatusCode int `json:"status_code"`;Message string `json:"message"`;Data PublicConfig `json:"data"`}
	if err:=json.Unmarshal(payload,&envelope);err!=nil{return PublicConfig{},fmt.Errorf("decode GMPay config: %w",err)}
	if response.StatusCode!=http.StatusOK||envelope.StatusCode!=200{return PublicConfig{},fmt.Errorf("%w: public config HTTP %d: %s",ErrProviderRejected,response.StatusCode,envelope.Message)}
	if envelope.Data.SupportedAssets==nil{envelope.Data.SupportedAssets=make([]SupportedAsset,0)}
	return envelope.Data,nil
}

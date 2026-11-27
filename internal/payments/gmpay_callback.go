package payments

import (
	"errors"
	"strconv"
	"strings"
)

var ErrInvalidGMPayCallback = errors.New("invalid GMPay callback")

type VerifiedGMPayCallback struct {
	PID string
	TradeID string
	OrderID string
	AmountUSDT int64
	CreditedUSDT int64
	Token string
	TransactionID string
}

func VerifyGMPayCallback(params map[string]string,secret,expectedPID,expectedOrderID string,expectedAmountUSDT int64)(VerifiedGMPayCallback,error){
	if !VerifyGMPaySignature(params,secret){return VerifiedGMPayCallback{},ErrInvalidGMPayCallback}
	amount,err:=ParseIntegerUSDT(params["amount"])
	if err!=nil||amount!=expectedAmountUSDT{return VerifiedGMPayCallback{},ErrInvalidGMPayCallback}
	status,err:=strconv.Atoi(params["status"])
	if err!=nil||status!=2{return VerifiedGMPayCallback{},ErrInvalidGMPayCallback}
	pid:=strings.TrimSpace(params["pid"])
	orderID:=strings.TrimSpace(params["order_id"])
	tradeID:=strings.TrimSpace(params["trade_id"])
	txID:=strings.TrimSpace(params["block_transaction_id"])
	if pid!=expectedPID||orderID!=expectedOrderID||tradeID==""||txID==""||!strings.EqualFold(strings.TrimSpace(params["token"]),"USDT"){
		return VerifiedGMPayCallback{},ErrInvalidGMPayCallback
	}
	credited,err:=ParseIntegerUSDT(params["actual_amount"])
	if err!=nil{return VerifiedGMPayCallback{PID:pid,TradeID:tradeID,OrderID:orderID,AmountUSDT:amount,Token:"USDT",TransactionID:txID},ErrNonIntegerUSDT}
	return VerifiedGMPayCallback{PID:pid,TradeID:tradeID,OrderID:orderID,AmountUSDT:amount,CreditedUSDT:credited,Token:"USDT",TransactionID:txID},nil
}

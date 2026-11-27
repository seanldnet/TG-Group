package payments

import (
	"errors"
	"math/big"
	"strings"
)

var ErrNonIntegerUSDT = errors.New("settled USDT amount must be a positive integer")

// ParseIntegerUSDT parses a decimal quantity without floating-point conversion.
// It rejects fractional quantities and values outside PostgreSQL BIGINT range.
func ParseIntegerUSDT(value string) (int64,error) {
	value=strings.TrimSpace(value)
	if value=="" { return 0,ErrNonIntegerUSDT }
	rat,ok:=new(big.Rat).SetString(value)
	if !ok || rat.Sign()<=0 || !rat.IsInt() || !rat.Num().IsInt64() { return 0,ErrNonIntegerUSDT }
	return rat.Num().Int64(),nil
}

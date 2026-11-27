package payments

import (
	"errors"
	"strconv"
)

// NormalizeJSONNumber matches Go's shortest decimal representation for JSON
// numeric values used by EPUSDT's signature canonicalization. It is never used
// to perform wallet arithmetic.
func NormalizeJSONNumber(value string)(string,error){
	number,err:=strconv.ParseFloat(value,64)
	if err!=nil{return "",err}
	if number!=number||number>1.7976931348623157e308||number< -1.7976931348623157e308{return "",errors.New("invalid JSON number")}
	return strconv.FormatFloat(number,'f',-1,64),nil
}

package payments

import (
	"errors"
	"strconv"
	"strings"
)

// NormalizeJSONNumber converts a valid JSON decimal to a shortest plain
// decimal string without floating-point rounding. It is used for signature
// canonicalization; wallet arithmetic still uses ParseIntegerUSDT.
func NormalizeJSONNumber(value string)(string,error){
	value=strings.TrimSpace(value)
	if value==""{return "",errors.New("empty JSON number")}
	sign:=""
	if value[0]=='-' {sign="-";value=value[1:]}
	exponent:=0
	if index:=strings.IndexAny(value,"eE");index>=0{
		parsed,err:=strconv.Atoi(value[index+1:])
		if err!=nil||parsed>1000||parsed< -1000{return "",errors.New("invalid JSON number exponent")}
		exponent=parsed
		value=value[:index]
	}
	parts:=strings.Split(value,".")
	if len(parts)>2||len(parts)==0||parts[0]==""{return "",errors.New("invalid JSON number")}
	integerPart:=parts[0]
	fractionPart:=""
	if len(parts)==2{fractionPart=parts[1]}
	for _,part:=range []string{integerPart,fractionPart}{
		for _,r:=range part{if r<'0'||r>'9'{return "",errors.New("invalid JSON number digits")}}
	}
	if len(parts)==2&&fractionPart==""{return "",errors.New("invalid JSON fraction")}
	digits:=integerPart+fractionPart
	decimalPosition:=len(integerPart)+exponent
	if decimalPosition>1000||decimalPosition< -1000{return "",errors.New("JSON number magnitude is out of range")}
	var whole,fraction string
	switch{
	case decimalPosition<=0:
		whole="0"
		fraction=strings.Repeat("0",-decimalPosition)+digits
	case decimalPosition>=len(digits):
		whole=digits+strings.Repeat("0",decimalPosition-len(digits))
	default:
		whole=digits[:decimalPosition]
		fraction=digits[decimalPosition:]
	}
	whole=strings.TrimLeft(whole,"0")
	if whole==""{whole="0"}
	fraction=strings.TrimRight(fraction,"0")
	if fraction!=""{return sign+whole+"."+fraction,nil}
	return sign+whole,nil
}

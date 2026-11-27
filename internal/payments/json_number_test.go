package payments

import "testing"

func TestNormalizeJSONNumber(t *testing.T){
	for input,want:=range map[string]string{"100.00":"100","14.2900":"14.29","1e+3":"1000","0.000001":"0.000001","9007199254740993":"9007199254740993","1.20e-2":"0.012"}{
		got,err:=NormalizeJSONNumber(input);if err!=nil{t.Fatalf("%s: %v",input,err)};if got!=want{t.Fatalf("NormalizeJSONNumber(%q)=%q want %q",input,got,want)}
	}
	if _,err:=NormalizeJSONNumber("not-number");err==nil{t.Fatal("invalid number accepted")}
}

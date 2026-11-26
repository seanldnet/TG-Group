package platformsettings

import "testing"

func TestNormalizeUsername(t *testing.T) {
	for _,tc:=range []struct{input,want string;valid bool}{
		{"@ExampleBot","ExampleBot",true},
		{"group_manager_bot","group_manager_bot",true},
		{"badname","",false},
		{"ab bot","",false},
		{"中文bot","",false},
	} {
		got,err:=normalizeUsername(tc.input)
		if tc.valid && (err!=nil||got!=tc.want) {t.Errorf("normalizeUsername(%q)=(%q,%v), want %q",tc.input,got,err,tc.want)}
		if !tc.valid && err==nil {t.Errorf("normalizeUsername(%q) unexpectedly valid",tc.input)}
	}
}

func TestValidTokenShape(t *testing.T) {
	for _,tc:=range []struct{token string;valid bool}{
		{"123456:abcdefghijklmnopqrstuvwxyz1234567890",true},
		{"123:short",false},
		{"abcde:abcdefghijklmnopqrstuvwxyz1234567890",false},
		{"123456:contains spaces abcdefghijklmnopqrstuvwxyz",false},
	} {
		if got:=validTokenShape(tc.token);got!=tc.valid {t.Errorf("validTokenShape(%q)=%v, want %v",tc.token,got,tc.valid)}
	}
}

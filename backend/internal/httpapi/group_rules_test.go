package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestGroupRulesID(t *testing.T) {
	for _, tc := range []struct{ path string; valid bool }{
		{"/groups/12/rules", true},
		{"/groups/0/rules", false},
		{"/groups/not-a-number/rules", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			req.SetPathValue("id", req.URL.Path[len("/groups/") : len(req.URL.Path)-len("/rules")])
			_, ok := groupRulesID(httptest.NewRecorder(), req)
			if ok != tc.valid { t.Fatalf("valid=%v, got %v", tc.valid, ok) }
		})
	}
}

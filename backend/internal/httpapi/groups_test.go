package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseGroupListQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/groups?botId=9&q=community&page=3&pageSize=40", nil)
	q, ok := parseGroupListQuery(req)
	if !ok { t.Fatal("expected valid query") }
	if q.BotID != 9 || q.Search != "community" || q.Page != 3 || q.PageSize != 40 {
		t.Fatalf("unexpected query: %#v", q)
	}
}

func TestParseGroupListQueryDefaults(t *testing.T) {
	q, ok := parseGroupListQuery(httptest.NewRequest("GET", "/groups", nil))
	if !ok { t.Fatal("expected defaults to be valid") }
	if q.BotID != 0 || q.Page != 1 || q.PageSize != 20 { t.Fatalf("unexpected defaults: %#v", q) }
}

func TestParseGroupListQueryRejectsInvalidInput(t *testing.T) {
	for _, target := range []string{"/groups?botId=0", "/groups?botId=x", "/groups?page=0", "/groups?pageSize=101", "/groups?q="+strings.Repeat("x", 101)} {
		t.Run(target, func(t *testing.T) {
			if _, ok := parseGroupListQuery(httptest.NewRequest("GET", target, nil)); ok {
				t.Fatal("expected invalid query to be rejected")
			}
		})
	}
}

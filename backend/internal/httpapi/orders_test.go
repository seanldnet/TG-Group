package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestParseOrderListQuery(t *testing.T) {
	request := httptest.NewRequest("GET", "/orders?botId=12&status=paid&q=TG-100&page=2&pageSize=50", nil)
	query, ok := parseOrderListQuery(request)
	if !ok {
		t.Fatal("expected valid order query")
	}
	if query.BotID != int64(12) || query.Status != "paid" || query.Search != "TG-100" {
		t.Fatalf("unexpected filters: %#v", query)
	}
	if query.Page != 2 || query.PageSize != 50 {
		t.Fatalf("unexpected pagination: %#v", query)
	}
}

func TestParseOrderListQueryDefaults(t *testing.T) {
	request := httptest.NewRequest("GET", "/orders", nil)
	query, ok := parseOrderListQuery(request)
	if !ok {
		t.Fatal("expected default query to be valid")
	}
	if query.BotID != nil || query.Status != "" || query.Page != 1 || query.PageSize != 20 {
		t.Fatalf("unexpected defaults: %#v", query)
	}
}

func TestParseOrderListQueryRejectsInvalidFilters(t *testing.T) {
	tests := []string{
		"/orders?botId=0",
		"/orders?botId=not-a-number",
		"/orders?status=unknown",
		"/orders?page=0",
		"/orders?pageSize=101",
		"/orders?q=" + string(make([]byte, 101)),
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest("GET", target, nil)
			if _, ok := parseOrderListQuery(request); ok {
				t.Fatal("expected invalid query to be rejected")
			}
		})
	}
}

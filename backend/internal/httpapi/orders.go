package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderHandler struct {
	DB *pgxpool.Pool
}

type orderListQuery struct {
	BotID    any
	Status   string
	Search   string
	Page     int
	PageSize int
}

type orderListItem struct {
	OrderNo        string    `json:"orderNo"`
	BotName        string    `json:"botName"`
	TelegramUserID *int64    `json:"telegramUserId"`
	AmountMinor    int64     `json:"amountMinor"`
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (h OrderHandler) Register(mux *http.ServeMux, auth AuthHandler) {
	mux.Handle("GET /orders", auth.requireSession(http.HandlerFunc(h.list)))
}

func (h OrderHandler) list(w http.ResponseWriter, r *http.Request) {
	query, ok := parseOrderListQuery(r)
	if !ok {
		apiError(w, http.StatusBadRequest, "invalid order filters or pagination")
		return
	}

	var total int64
	err := h.DB.QueryRow(r.Context(), `
		SELECT COUNT(*)
		FROM orders o
		WHERE ($1::bigint IS NULL OR o.bot_id = $1)
		  AND ($2 = '' OR o.status = $2)
		  AND ($3 = '' OR o.order_no ILIKE '%' || $3 || '%'
		       OR COALESCE(o.telegram_user_id::text, '') LIKE '%' || $3 || '%')
	`, query.BotID, query.Status, query.Search).Scan(&total)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not count orders")
		return
	}

	rows, err := h.DB.Query(r.Context(), `
		SELECT o.order_no, COALESCE(b.name, ''), o.telegram_user_id,
		       o.amount_minor, o.currency, o.status, o.created_at
		FROM orders o
		LEFT JOIN telegram_bots b ON b.id = o.bot_id
		WHERE ($1::bigint IS NULL OR o.bot_id = $1)
		  AND ($2 = '' OR o.status = $2)
		  AND ($3 = '' OR o.order_no ILIKE '%' || $3 || '%'
		       OR COALESCE(o.telegram_user_id::text, '') LIKE '%' || $3 || '%')
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT $4 OFFSET $5
	`, query.BotID, query.Status, query.Search, query.PageSize, (query.Page-1)*query.PageSize)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load orders")
		return
	}
	defer rows.Close()

	items := make([]orderListItem, 0, query.PageSize)
	for rows.Next() {
		var item orderListItem
		if err := rows.Scan(&item.OrderNo, &item.BotName, &item.TelegramUserID,
			&item.AmountMinor, &item.Currency, &item.Status, &item.CreatedAt); err != nil {
			apiError(w, http.StatusInternalServerError, "could not load orders")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		apiError(w, http.StatusInternalServerError, "could not load orders")
		return
	}

	apiOK(w, http.StatusOK, map[string]any{
		"items": items, "total": total, "page": query.Page, "pageSize": query.PageSize,
	})
}

func parseOrderListQuery(r *http.Request) (orderListQuery, bool) {
	values := r.URL.Query()
	query := orderListQuery{Page: 1, PageSize: 20}
	if raw := strings.TrimSpace(values.Get("botId")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return query, false
		}
		query.BotID = id
	}
	query.Status = strings.TrimSpace(values.Get("status"))
	switch query.Status {
	case "", "pending", "paid", "failed", "cancelled", "refunded":
	default:
		return query, false
	}
	query.Search = strings.TrimSpace(values.Get("q"))
	if len(query.Search) > 100 {
		return query, false
	}
	if raw := values.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 || page > 1_000_000 {
			return query, false
		}
		query.Page = page
	}
	if raw := values.Get("pageSize"); raw != "" {
		size, err := strconv.Atoi(raw)
		if err != nil || size < 1 || size > 100 {
			return query, false
		}
		query.PageSize = size
	}
	return query, true
}

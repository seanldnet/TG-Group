package httpapi

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardHandler struct {
	DB *pgxpool.Pool
}

type dashboardOrder struct {
	OrderNo       string    `json:"orderNo"`
	BotName       string    `json:"botName"`
	TelegramUserID *int64   `json:"telegramUserId"`
	AmountMinor   int64     `json:"amountMinor"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (h DashboardHandler) Register(mux *http.ServeMux, auth AuthHandler) {
	mux.Handle("GET /dashboard/summary", auth.requireSession(http.HandlerFunc(h.summary)))
}

func (h DashboardHandler) summary(w http.ResponseWriter, r *http.Request) {
	var totalBots, activeBots, totalGroups, totalMembers int64
	err := h.DB.QueryRow(r.Context(), `
		SELECT
			(SELECT COUNT(*) FROM telegram_bots),
			(SELECT COUNT(*) FROM telegram_bots WHERE status = 'active'),
			(SELECT COUNT(*) FROM telegram_groups),
			(SELECT COUNT(*) FROM telegram_members)
	`).Scan(&totalBots, &activeBots, &totalGroups, &totalMembers)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load dashboard metrics")
		return
	}

	rows, err := h.DB.Query(r.Context(), `
		SELECT o.order_no, COALESCE(b.name, ''), o.telegram_user_id,
		       o.amount_minor, o.currency, o.status, o.created_at
		FROM orders o
		LEFT JOIN telegram_bots b ON b.id = o.bot_id
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT 10
	`)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load recent orders")
		return
	}
	defer rows.Close()

	recentOrders := make([]dashboardOrder, 0, 10)
	for rows.Next() {
		var item dashboardOrder
		if err := rows.Scan(&item.OrderNo, &item.BotName, &item.TelegramUserID,
			&item.AmountMinor, &item.Currency, &item.Status, &item.CreatedAt); err != nil {
			apiError(w, http.StatusInternalServerError, "could not load recent orders")
			return
		}
		recentOrders = append(recentOrders, item)
	}
	if err := rows.Err(); err != nil {
		apiError(w, http.StatusInternalServerError, "could not load recent orders")
		return
	}

	apiOK(w, http.StatusOK, map[string]any{
		"totalBots": totalBots,
		"activeBots": activeBots,
		"totalGroups": totalGroups,
		"totalMembers": totalMembers,
		"recentOrders": recentOrders,
	})
}

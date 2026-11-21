package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupHandler struct { DB *pgxpool.Pool }

type groupListQuery struct {
	BotID int64
	Search string
	Page int
	PageSize int
}

type groupListItem struct {
	ID int64 `json:"id"`
	BotID int64 `json:"botId"`
	BotName string `json:"botName"`
	TelegramChatID int64 `json:"telegramChatId"`
	Title string `json:"title"`
	MemberCount int64 `json:"memberCount"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h GroupHandler) Register(mux *http.ServeMux, auth AuthHandler) {
	mux.Handle("GET /groups", auth.requireSession(http.HandlerFunc(h.list)))
}

func (h GroupHandler) list(w http.ResponseWriter, r *http.Request) {
	q, ok := parseGroupListQuery(r)
	if !ok {
		apiError(w, http.StatusBadRequest, "invalid group filters or pagination")
		return
	}
	var total int64
	err := h.DB.QueryRow(r.Context(), `
		SELECT COUNT(*)
		FROM telegram_groups g
		WHERE ($1::bigint = 0 OR g.bot_id = $1)
		  AND ($2 = '' OR g.title ILIKE '%' || $2 || '%'
		       OR g.telegram_chat_id::text LIKE '%' || $2 || '%')
	`, q.BotID, q.Search).Scan(&total)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not count groups")
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT g.id, g.bot_id, COALESCE(b.name, ''), g.telegram_chat_id,
		       g.title, COUNT(m.id), g.created_at
		FROM telegram_groups g
		LEFT JOIN telegram_bots b ON b.id = g.bot_id
		LEFT JOIN telegram_members m ON m.group_id = g.id
		WHERE ($1::bigint = 0 OR g.bot_id = $1)
		  AND ($2 = '' OR g.title ILIKE '%' || $2 || '%'
		       OR g.telegram_chat_id::text LIKE '%' || $2 || '%')
		GROUP BY g.id, b.name
		ORDER BY g.created_at DESC, g.id DESC
		LIMIT $3 OFFSET $4
	`, q.BotID, q.Search, q.PageSize, (q.Page-1)*q.PageSize)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "could not load groups")
		return
	}
	defer rows.Close()
	items := make([]groupListItem, 0, q.PageSize)
	for rows.Next() {
		var item groupListItem
		if err := rows.Scan(&item.ID, &item.BotID, &item.BotName, &item.TelegramChatID,
			&item.Title, &item.MemberCount, &item.CreatedAt); err != nil {
			apiError(w, http.StatusInternalServerError, "could not load groups")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		apiError(w, http.StatusInternalServerError, "could not load groups")
		return
	}
	apiOK(w, http.StatusOK, map[string]any{
		"items": items, "total": total, "page": q.Page, "pageSize": q.PageSize,
	})
}

func parseGroupListQuery(r *http.Request) (groupListQuery, bool) {
	values := r.URL.Query()
	q := groupListQuery{Page: 1, PageSize: 20}
	if raw := strings.TrimSpace(values.Get("botId")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 { return q, false }
		q.BotID = id
	}
	q.Search = strings.TrimSpace(values.Get("q"))
	if len(q.Search) > 100 { return q, false }
	if raw := values.Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil || page < 1 || page > 1_000_000 { return q, false }
		q.Page = page
	}
	if raw := values.Get("pageSize"); raw != "" {
		size, err := strconv.Atoi(raw)
		if err != nil || size < 1 || size > 100 { return q, false }
		q.PageSize = size
	}
	return q, true
}

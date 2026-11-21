package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRulesHandler struct { DB *pgxpool.Pool }

type groupRules struct {
	GroupID int64 `json:"groupId"`
	BlockLinks bool `json:"blockLinks"`
	BlockedKeywords []string `json:"blockedKeywords"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

func (h GroupRulesHandler) Register(mux *http.ServeMux, auth AuthHandler) {
	mux.Handle("GET /groups/{id}/rules", auth.requireSession(http.HandlerFunc(h.get)))
	mux.Handle("PUT /groups/{id}/rules", auth.requireSession(http.HandlerFunc(h.put)))
}

func (h GroupRulesHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := groupRulesID(w, r)
	if !ok { return }
	var rules groupRules
	err := h.DB.QueryRow(r.Context(), `
		SELECT group_id, block_links, blocked_keywords, updated_at
		FROM group_moderation_rules WHERE group_id = $1
	`, id).Scan(&rules.GroupID, &rules.BlockLinks, &rules.BlockedKeywords, &rules.UpdatedAt)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			apiError(w, http.StatusInternalServerError, "could not load group rules")
			return
		}
		// A group without an explicit policy uses the safe, disabled defaults.
		var exists bool
		if checkErr := h.DB.QueryRow(r.Context(), "SELECT EXISTS (SELECT 1 FROM telegram_groups WHERE id = $1)", id).Scan(&exists); checkErr != nil {
			apiError(w, http.StatusInternalServerError, "could not load group rules")
			return
		}
		if !exists { apiError(w, http.StatusNotFound, "group not found"); return }
		apiOK(w, http.StatusOK, groupRules{GroupID:id, BlockedKeywords:[]string{}})
		return
	}
	if rules.BlockedKeywords == nil { rules.BlockedKeywords = []string{} }
	apiOK(w, http.StatusOK, rules)
}

func (h GroupRulesHandler) put(w http.ResponseWriter, r *http.Request) {
	id, ok := groupRulesID(w, r)
	if !ok { return }
	var input struct {
		BlockLinks bool `json:"blockLinks"`
		BlockedKeywords []string `json:"blockedKeywords"`
	}
	if !decodeJSON(w, r, &input) { return }
	if len(input.BlockedKeywords) > 100 {
		apiError(w, http.StatusBadRequest, "at most 100 blocked keywords are allowed")
		return
	}
	keywords := make([]string, 0, len(input.BlockedKeywords))
	seen := make(map[string]struct{})
	for _, keyword := range input.BlockedKeywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" { continue }
		if len(keyword) > 64 {
			apiError(w, http.StatusBadRequest, "each blocked keyword must be at most 64 characters")
			return
		}
		key := strings.ToLower(keyword)
		if _, exists := seen[key]; exists { continue }
		seen[key] = struct{}{}
		keywords = append(keywords, keyword)
	}
	encoded, err := json.Marshal(keywords)
	if err != nil { apiError(w, http.StatusBadRequest, "invalid blocked keywords"); return }
	tag, err := h.DB.Exec(r.Context(), `
		INSERT INTO group_moderation_rules (group_id, block_links, blocked_keywords, updated_at)
		SELECT id, $2, $3::jsonb, NOW() FROM telegram_groups WHERE id = $1
		ON CONFLICT (group_id) DO UPDATE SET
			block_links = EXCLUDED.block_links,
			blocked_keywords = EXCLUDED.blocked_keywords,
			updated_at = NOW()
	`, id, input.BlockLinks, string(encoded))
	if err != nil { apiError(w, http.StatusInternalServerError, "could not save group rules"); return }
	if tag.RowsAffected() == 0 { apiError(w, http.StatusNotFound, "group not found"); return }
	apiOK(w, http.StatusOK, map[string]any{"groupId":id,"blockLinks":input.BlockLinks,"blockedKeywords":keywords})
}

func groupRulesID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		apiError(w, http.StatusBadRequest, "invalid group id")
		return 0, false
	}
	return id, true
}

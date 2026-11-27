package httpapi

import (
	"net/http"
	"strconv"

	"github.com/seanldnet/tg-group/internal/bots"
)

func (h *Handler) listWarningEscalations(w http.ResponseWriter, r *http.Request) {
	if !h.botServicesReady(w) { return }
	user, ok := h.currentUser(w,r)
	if !ok { return }
	userID, err := strconv.ParseInt(r.PathValue("userID"),10,64)
	if err != nil || userID <= 0 { writeError(w,http.StatusBadRequest,"invalid Telegram user ID"); return }
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 1 || value > 100 { writeError(w,http.StatusBadRequest,"limit must be between 1 and 100"); return }
		limit = value
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		value, parseErr := strconv.Atoi(raw)
		if parseErr != nil || value < 0 || value > 100000 { writeError(w,http.StatusBadRequest,"offset must be between 0 and 100000"); return }
		offset = value
	}
	items, total, err := h.bots.ListWarningEscalationsPage(r.Context(),user.TenantID,r.PathValue("groupID"),userID,limit,offset)
	if err != nil { h.writeGroupRuleError(w,err); return }
	writeJSON(w,http.StatusOK,map[string]any{"items":items,"total":total,"limit":limit,"offset":offset})
}

func (h *Handler) retryWarningEscalation(w http.ResponseWriter, r *http.Request) {
	if !h.botServicesReady(w) { return }
	user, ok := h.currentUser(w,r)
	if !ok { return }
	userID, err := strconv.ParseInt(r.PathValue("userID"),10,64)
	if err != nil || userID <= 0 { writeError(w,http.StatusBadRequest,"invalid Telegram user ID"); return }
	executionID, err := strconv.ParseInt(r.PathValue("executionID"),10,64)
	if err != nil || executionID <= 0 { writeError(w,http.StatusBadRequest,"invalid execution ID"); return }
	item, err := h.bots.RetryWarningEscalation(r.Context(),user.TenantID,user.ID,r.PathValue("groupID"),userID,executionID)
	if err != nil {
		if err == bots.ErrWarningEscalationNotRetryable { writeError(w,http.StatusConflict,"该执行记录不存在、已处理或当前不可重试"); return }
		h.writeGroupRuleError(w,err)
		return
	}
	writeJSON(w,http.StatusOK,item)
}

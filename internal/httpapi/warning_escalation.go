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
	items, err := h.bots.ListWarningEscalations(r.Context(),user.TenantID,r.PathValue("groupID"),userID,limit)
	if err != nil { h.writeGroupRuleError(w,err); return }
	writeJSON(w,http.StatusOK,map[string]any{"items":items})
}

func (h *Handler) retryWarningEscalation(w http.ResponseWriter, r *http.Request) {
	if !h.botServicesReady(w) { return }
	user, ok := h.currentUser(w,r)
	if !ok { return }
	userID, err := strconv.ParseInt(r.PathValue("userID"),10,64)
	if err != nil || userID <= 0 { writeError(w,http.StatusBadRequest,"invalid Telegram user ID"); return }
	executionID, err := strconv.ParseInt(r.PathValue("executionID"),10,64)
	if err != nil || executionID <= 0 { writeError(w,http.StatusBadRequest,"invalid execution ID"); return }
	item, err := h.bots.RetryWarningEscalation(r.Context(),user.TenantID,user.ID,r.PathValue("groupID"),executionID)
	if err != nil {
		if err == bots.ErrWarningEscalationNotRetryable { writeError(w,http.StatusConflict,"该执行记录不存在、已处理或当前不可重试"); return }
		h.writeGroupRuleError(w,err)
		return
	}
	if item.TelegramUserID != userID { writeError(w,http.StatusNotFound,"warning escalation execution not found"); return }
	writeJSON(w,http.StatusOK,item)
}

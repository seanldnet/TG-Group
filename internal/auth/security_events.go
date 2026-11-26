package auth

import (
	"context"
	"fmt"
	"time"
)

type SecurityEvent struct {
	ID int64 `json:"id"`
	Action string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID string `json:"resource_id"`
	IPAddress string `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}

// ListSecurityEvents returns only authentication-related audit events owned by
// the authenticated user; tenant peers cannot inspect another user's history.
func (s *Service) ListSecurityEvents(ctx context.Context, userID string, limit int) ([]SecurityEvent, error) {
	if limit < 1 || limit > 100 { limit=20 }
	rows,err:=s.pool.Query(ctx,`SELECT id,action,resource_type,resource_id,COALESCE(ip_address::text,''),created_at
		FROM audit_logs
		WHERE actor_user_id=$1 AND action LIKE 'auth.%'
		ORDER BY created_at DESC,id DESC
		LIMIT $2`,userID,limit)
	if err!=nil { return nil,fmt.Errorf("list account security events: %w",err) }
	defer rows.Close()
	events:=make([]SecurityEvent,0)
	for rows.Next() {
		var event SecurityEvent
		if err:=rows.Scan(&event.ID,&event.Action,&event.ResourceType,&event.ResourceID,&event.IPAddress,&event.CreatedAt);err!=nil {
			return nil,fmt.Errorf("scan account security event: %w",err)
		}
		events=append(events,event)
	}
	if err:=rows.Err();err!=nil { return nil,fmt.Errorf("iterate account security events: %w",err) }
	return events,nil
}

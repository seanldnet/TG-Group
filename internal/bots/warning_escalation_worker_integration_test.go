package bots

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/seanldnet/tg-group/internal/auth"
	"github.com/seanldnet/tg-group/internal/database"
)

type warningRetryFixture struct {
	db *database.DB
	service *Service
	tenantID string
	otherTenantID string
	groupID string
}

func newWarningRetryFixture(t *testing.T) warningRetryFixture {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := database.Open(ctx, databaseURL)
	if err != nil { t.Fatalf("open database: %v", err) }
	if err := db.Migrate(ctx); err != nil { db.Close(); t.Fatalf("migrate: %v", err) }
	t.Cleanup(db.Close)

	suffix := time.Now().UnixNano()
	authService := auth.NewService(db.Pool)
	owner, err := authService.Register(ctx, fmt.Sprintf("warning-retry-%d@example.test", suffix), "correct-horse-battery", "Retry Owner", "", "")
	if err != nil { t.Fatalf("register owner: %v", err) }
	other, err := authService.Register(ctx, fmt.Sprintf("warning-retry-other-%d@example.test", suffix), "correct-horse-battery", "Other Owner", "", "")
	if err != nil { t.Fatalf("register other tenant: %v", err) }

	var botID string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO telegram_bots(
		tenant_id,created_by,telegram_bot_id,token_ciphertext,token_nonce,token_fingerprint
	) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`,
		owner.User.TenantID, owner.User.ID, suffix, []byte(fmt.Sprintf("cipher-%d", suffix)),
		[]byte(fmt.Sprintf("nonce-%d", suffix)), []byte(fmt.Sprintf("fingerprint-%d", suffix)),
	).Scan(&botID); err != nil { t.Fatalf("create bot fixture: %v", err) }

	var groupID string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO telegram_groups(tenant_id,bot_id,telegram_chat_id,title,chat_type)
		VALUES($1,$2,$3,'Warning Retry Fixture','supergroup') RETURNING id::text`,
		owner.User.TenantID, botID, -suffix,
	).Scan(&groupID); err != nil { t.Fatalf("create group fixture: %v", err) }

	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	service, err := NewService(db.Pool, key)
	if err != nil { t.Fatalf("create Bot service: %v", err) }
	return warningRetryFixture{db: db, service: service, tenantID: owner.User.TenantID, otherTenantID: other.User.TenantID, groupID: groupID}
}

func insertWarningRetryFixture(t *testing.T, f warningRetryFixture, tenantID, groupID string, userID int64, status string, attempt int, due time.Time) int64 {
	t.Helper()
	ctx := context.Background()
	var id int64
	err := f.db.Pool.QueryRow(ctx, `INSERT INTO telegram_group_warning_escalations(
		tenant_id,group_id,rule_id,telegram_user_id,warning_count,action,duration_minutes,status,attempt_count,next_retry_at,processing_started_at
	) VALUES($1,$2,gen_random_uuid(),$3,1,'ban',0,$4,$5,
		CASE WHEN $4='failed' THEN $6 ELSE NULL END,
		CASE WHEN $4='processing' THEN $6 ELSE NULL END
	) RETURNING id`, tenantID, groupID, userID, status, attempt, due).Scan(&id)
	if err != nil { t.Fatalf("insert retry fixture: %v", err) }
	return id
}

func TestWarningEscalationClaimSkipsMismatchedTenantGroupPostgres(t *testing.T) {
	f := newWarningRetryFixture(t)
	now := time.Now().UTC()
	// This invalid row is older and must not block the valid row behind it.
	insertWarningRetryFixture(t, f, f.otherTenantID, f.groupID, 61001, "failed", 0, now.Add(-2*time.Minute))
	validID := insertWarningRetryFixture(t, f, f.tenantID, f.groupID, 61002, "failed", 0, now.Add(-time.Minute))

	item, err := f.service.claimWarningEscalation(context.Background())
	if err != nil { t.Fatalf("claim valid retry: %v", err) }
	if item.ID != validID || item.TenantID != f.tenantID || item.GroupID != f.groupID {
		t.Fatalf("claimed mismatched retry: %+v, want valid execution %d", item, validID)
	}
}

func TestWarningEscalationClaimRecoversExpiredLeasePostgres(t *testing.T) {
	f := newWarningRetryFixture(t)
	id := insertWarningRetryFixture(t, f, f.tenantID, f.groupID, 62001, "processing", 1, time.Now().UTC().Add(-warningEscalationLease-time.Minute))
	item, err := f.service.claimWarningEscalation(context.Background())
	if err != nil { t.Fatalf("claim expired lease: %v", err) }
	if item.ID != id || item.AttemptCount != 2 {
		t.Fatalf("recovered item = %+v, want id=%d attempt=2", item, id)
	}
}

func TestWarningEscalationConcurrentClaimsAreDistinctPostgres(t *testing.T) {
	f := newWarningRetryFixture(t)
	now := time.Now().UTC()
	insertWarningRetryFixture(t, f, f.tenantID, f.groupID, 63001, "failed", 0, now.Add(-time.Minute))
	insertWarningRetryFixture(t, f, f.tenantID, f.groupID, 63002, "failed", 0, now.Add(-time.Minute))

	var wg sync.WaitGroup
	items := make([]pendingWarningEscalation, 2)
	errs := make([]error, 2)
	for i := range items {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			items[index], errs[index] = f.service.claimWarningEscalation(context.Background())
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil { t.Fatalf("claim %d: %v", i, err) }
	}
	if items[0].ID == items[1].ID {
		t.Fatalf("concurrent workers claimed the same execution %d", items[0].ID)
	}
}

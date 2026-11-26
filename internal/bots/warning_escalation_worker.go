package bots

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	warningEscalationMaxAttempts = 8
	warningEscalationLease       = 5 * time.Minute
)

type pendingWarningEscalation struct {
	ID             int64
	TenantID       string
	GroupID        string
	UserID         int64
	Action         string
	Duration       int
	AttemptCount   int
}

// WarningEscalationRetryDelay applies bounded exponential backoff. The first
// automatic retry waits 30 seconds; the delay is capped at one hour.
func WarningEscalationRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := 30 * time.Second * time.Duration(math.Pow(2, float64(attempt-1)))
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

// RunWarningEscalationRetries continuously recovers due failures and expired
// processing leases. The database is the queue, so restarts do not lose work.
func (s *Service) RunWarningEscalationRetries(ctx context.Context, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		s.processDueWarningEscalations(ctx, logger)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) processDueWarningEscalations(ctx context.Context, logger *slog.Logger) {
	for ctx.Err() == nil {
		item, err := s.claimWarningEscalation(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return
		}
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("claim warning escalation retry", "error", err)
			}
			return
		}

		actionErr := s.ModerateGroupMember(ctx, item.TenantID, "", item.GroupID, item.UserID, item.Action, item.Duration)
		if err := s.finishWarningEscalation(ctx, item, actionErr); err != nil {
			logger.Error("save warning escalation retry result", "execution_id", item.ID, "error", err)
			return
		}
		if actionErr != nil {
			logger.Warn("warning escalation retry failed", "execution_id", item.ID, "attempt", item.AttemptCount, "error", actionErr)
		} else {
			logger.Info("warning escalation retry succeeded", "execution_id", item.ID, "attempt", item.AttemptCount)
		}
	}
}

func (s *Service) claimWarningEscalation(ctx context.Context) (pendingWarningEscalation, error) {
	var item pendingWarningEscalation
	err := s.pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT id
			FROM telegram_group_warning_escalations
			WHERE (
				status = 'failed'
				AND next_retry_at IS NOT NULL
				AND next_retry_at <= now()
				AND attempt_count < $1
			) OR (
				status = 'processing'
				AND processing_started_at IS NOT NULL
				AND processing_started_at <= now() - $2::interval
				AND attempt_count < $1
			)
			ORDER BY COALESCE(next_retry_at, processing_started_at), id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE telegram_group_warning_escalations e
		SET status = 'processing',
			attempt_count = e.attempt_count + 1,
			next_retry_at = NULL,
			processing_started_at = now(),
			detail = 'automatic retry attempt ' || (e.attempt_count + 1)::text,
			updated_at = now()
		FROM candidate
		JOIN telegram_groups g ON g.id = e.group_id
		WHERE e.id = candidate.id
			AND g.tenant_id = e.tenant_id
		RETURNING e.id, e.tenant_id::text, e.group_id::text,
			e.telegram_user_id, e.action, e.duration_minutes, e.attempt_count
	`, warningEscalationMaxAttempts, warningEscalationLease.String()).Scan(
		&item.ID, &item.TenantID, &item.GroupID, &item.UserID,
		&item.Action, &item.Duration, &item.AttemptCount,
	)
	return item, err
}

func (s *Service) finishWarningEscalation(ctx context.Context, item pendingWarningEscalation, actionErr error) error {
	status := "succeeded"
	detail := "automatic warning escalation retry applied"
	var nextRetry any
	if actionErr != nil {
		status = "failed"
		detail = actionErr.Error()
		if len([]rune(detail)) > 500 {
			detail = string([]rune(detail)[:500])
		}
		if item.AttemptCount < warningEscalationMaxAttempts {
			nextRetry = time.Now().UTC().Add(WarningEscalationRetryDelay(item.AttemptCount))
		} else {
			detail = "automatic retry limit reached: " + detail
		}
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE telegram_group_warning_escalations
		SET status = $2, detail = $3, next_retry_at = $4,
			processing_started_at = NULL, updated_at = now()
		WHERE id = $1 AND tenant_id = $5 AND group_id = $6
	`, item.ID, status, detail, nextRetry, item.TenantID, item.GroupID)
	return err
}

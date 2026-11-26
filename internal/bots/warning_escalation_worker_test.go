package bots

import (
	"testing"
	"time"
)

func TestWarningEscalationRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: -1, want: 30 * time.Second},
		{attempt: 0, want: 30 * time.Second},
		{attempt: 1, want: 30 * time.Second},
		{attempt: 2, want: time.Minute},
		{attempt: 3, want: 2 * time.Minute},
		{attempt: 4, want: 4 * time.Minute},
		{attempt: 8, want: 64 * time.Minute},
		{attempt: 9, want: time.Hour},
		{attempt: 20, want: time.Hour},
	}
	for _, test := range tests {
		if got := WarningEscalationRetryDelay(test.attempt); got != test.want {
			t.Errorf("WarningEscalationRetryDelay(%d) = %s, want %s", test.attempt, got, test.want)
		}
	}
}

func TestWarningEscalationRetryLimit(t *testing.T) {
	if warningEscalationMaxAttempts != 8 {
		t.Fatalf("warningEscalationMaxAttempts = %d, want 8", warningEscalationMaxAttempts)
	}
	if warningEscalationLease != 5*time.Minute {
		t.Fatalf("warningEscalationLease = %s, want 5m", warningEscalationLease)
	}
}

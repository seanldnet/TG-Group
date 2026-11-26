package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

const (
	loginFailureLimit = 5
	loginFailureWindow = "15 minutes"
)

type loginIdentifier struct {
	kind string
	hash []byte
	lockKey string
}

func loginIdentifiers(email, ip string) []loginIdentifier {
	emailHash := sha256.Sum256([]byte("email:" + email))
	identifiers := []loginIdentifier{{kind:"email",hash:emailHash[:],lockKey:"email:"+hex.EncodeToString(emailHash[:])}}
	if ip != "" {
		ipHash := sha256.Sum256([]byte("ip:" + ip))
		identifiers = append(identifiers,loginIdentifier{kind:"ip",hash:ipHash[:],lockKey:"ip:"+hex.EncodeToString(ipHash[:])})
	}
	return identifiers
}

// lockLoginIdentifiers serializes attempts for the same email and source IP,
// so concurrent requests cannot bypass the configured failure threshold.
func lockLoginIdentifiers(ctx context.Context, tx pgx.Tx, identifiers []loginIdentifier) error {
	keys := make([]string,0,len(identifiers))
	for _, identifier := range identifiers { keys=append(keys,identifier.lockKey) }
	sort.Strings(keys)
	for _, key := range keys {
		if _,err:=tx.Exec(ctx,"SELECT pg_advisory_xact_lock(hashtextextended($1,0))",key);err!=nil {
			return fmt.Errorf("lock login rate-limit key: %w",err)
		}
	}
	return nil
}

func loginRateLimited(ctx context.Context, tx pgx.Tx, identifiers []loginIdentifier) (bool,error) {
	for _,identifier:=range identifiers {
		var count int
		if err:=tx.QueryRow(ctx,`SELECT count(*) FROM auth_login_failures
			WHERE identifier_type=$1 AND identifier_hash=$2 AND failed_at>now()-interval '15 minutes'`,
			identifier.kind,identifier.hash).Scan(&count);err!=nil {
			return false,fmt.Errorf("count login failures: %w",err)
		}
		if count>=loginFailureLimit { return true,nil }
	}
	return false,nil
}

func recordLoginFailure(ctx context.Context, tx pgx.Tx, identifiers []loginIdentifier) error {
	for _,identifier:=range identifiers {
		if _,err:=tx.Exec(ctx,`INSERT INTO auth_login_failures(identifier_type,identifier_hash) VALUES($1,$2)`,
			identifier.kind,identifier.hash);err!=nil {
			return fmt.Errorf("record login failure: %w",err)
		}
	}
	// Bound table growth without requiring a background worker.
	if _,err:=tx.Exec(ctx,`DELETE FROM auth_login_failures
		WHERE id IN (SELECT id FROM auth_login_failures
			WHERE failed_at<now()-interval '24 hours' ORDER BY failed_at LIMIT 1000)`);err!=nil {
		return fmt.Errorf("prune old login failures: %w",err)
	}
	return nil
}

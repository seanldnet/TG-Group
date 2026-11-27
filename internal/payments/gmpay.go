package payments

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// GMPaySignature returns the lowercase HMAC-SHA256 signature defined by the
// EPUSDT merchant API. Empty values and the signature field are excluded.
func GMPaySignature(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "signature" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for i, key := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(key)
		canonical.WriteByte('=')
		canonical.WriteString(params[key])
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyGMPaySignature compares a supplied merchant signature in constant time.
func VerifyGMPaySignature(params map[string]string, secret string) bool {
	provided := strings.ToLower(strings.TrimSpace(params["signature"]))
	if len(provided) != sha256.Size*2 {
		return false
	}
	expected := GMPaySignature(params, secret)
	return hmac.Equal([]byte(provided), []byte(expected))
}


// EPaySignature implements EPUSDT's legacy EPay-compatible MD5 signature.
// New integrations should use GMPaySignature (HMAC-SHA256) instead.
func EPaySignature(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || key == "sign_type" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for i, key := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(key)
		canonical.WriteByte('=')
		canonical.WriteString(params[key])
	}
	canonical.WriteString(secret)
	sum := md5.Sum([]byte(canonical.String()))
	return hex.EncodeToString(sum[:])
}

// VerifyEPaySignature compares a supplied legacy EPay signature in constant time.
func VerifyEPaySignature(params map[string]string, secret string) bool {
	provided := strings.ToLower(strings.TrimSpace(params["sign"]))
	if len(provided) != md5.Size*2 {
		return false
	}
	expected := EPaySignature(params, secret)
	return hmac.Equal([]byte(provided), []byte(expected))
}

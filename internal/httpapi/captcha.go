package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net"
	"net/url"
	"strings"
	"time"
)

var ErrCaptchaRejected = errors.New("captcha verification rejected")

type CaptchaVerifier struct {
	endpoint string
	secret string
	client *http.Client
}

type captchaSiteVerifyResponse struct {
	Success bool `json:"success"`
}

func NewCaptchaVerifier(baseURL, siteKey, secret string) (*CaptchaVerifier, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	siteKey = strings.TrimSpace(siteKey)
	secret = strings.TrimSpace(secret)
	if baseURL == "" && siteKey == "" && secret == "" {
		return nil, nil
	}
	if baseURL == "" || siteKey == "" || secret == "" {
		return nil, errors.New("CAP_BASE_URL, CAP_SITE_KEY and CAP_SECRET must be configured together")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()))) || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("CAP_BASE_URL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	endpoint := baseURL + "/" + url.PathEscape(siteKey) + "/"
	return &CaptchaVerifier{endpoint: endpoint, secret: secret, client: &http.Client{Timeout: 6 * time.Second}}, nil
}

func (v *CaptchaVerifier) Enabled() bool { return v != nil }

func (v *CaptchaVerifier) PublicConfig() map[string]any {
	if v == nil { return map[string]any{"enabled": false} }
	return map[string]any{"enabled": true, "endpoint": v.endpoint}
}

func (v *CaptchaVerifier) Verify(ctx context.Context, token string) error {
	if v == nil { return nil }
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 8192 { return ErrCaptchaRejected }
	body, err := json.Marshal(map[string]string{"secret": v.secret, "response": token})
	if err != nil { return fmt.Errorf("encode CAP verification: %w", err) }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint+"siteverify", bytes.NewReader(body))
	if err != nil { return fmt.Errorf("create CAP verification request: %w", err) }
	req.Header.Set("Content-Type", "application/json")
	response, err := v.client.Do(req)
	if err != nil { return fmt.Errorf("CAP verification service unavailable: %w", err) }
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return ErrCaptchaRejected
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil { return fmt.Errorf("read CAP verification response: %w", err) }
	var result captchaSiteVerifyResponse
	if err := json.Unmarshal(responseBody, &result); err != nil { return ErrCaptchaRejected }
	if !result.Success { return ErrCaptchaRejected }
	return nil
}


func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") { return true }
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

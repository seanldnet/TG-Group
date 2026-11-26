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
	widgetURL string
	client *http.Client
}

type captchaSiteVerifyResponse struct {
	Success bool `json:"success"`
}

func NewCaptchaVerifier(baseURL, siteKey, secret string, widgetURLs ...string) (*CaptchaVerifier, error) {
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
	widgetURL := "https://cdn.jsdelivr.net/npm/cap-widget@0.1.58"
	if len(widgetURLs) > 0 && strings.TrimSpace(widgetURLs[0]) != "" { widgetURL = strings.TrimSpace(widgetURLs[0]) }
	widgetParsed, widgetErr := url.Parse(widgetURL)
	if widgetErr != nil || widgetParsed.Host == "" || (widgetParsed.Scheme != "https" && !(widgetParsed.Scheme == "http" && isLoopbackHost(widgetParsed.Hostname()))) || widgetParsed.User != nil || widgetParsed.RawQuery != "" || widgetParsed.Fragment != "" {
		return nil, errors.New("CAP_WIDGET_URL must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	return &CaptchaVerifier{endpoint: endpoint, secret: secret, widgetURL: widgetURL, client: &http.Client{Timeout: 6 * time.Second}}, nil
}

func (v *CaptchaVerifier) Enabled() bool { return v != nil }

func (v *CaptchaVerifier) PublicConfig() map[string]any {
	if v == nil { return map[string]any{"enabled": false} }
	return map[string]any{"enabled": true, "endpoint": v.endpoint, "widget_url": v.widgetURL}
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
	if response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return fmt.Errorf("CAP verification service returned HTTP %d", response.StatusCode)
	}
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

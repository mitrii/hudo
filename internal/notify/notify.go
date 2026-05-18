// Package notify sends text notifications via a configurable webhook URL.
package notify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Send posts text to webhookURL.
// If the URL contains the placeholder {text} it is replaced with the
// URL-encoded message and a GET request is made (Telegram Bot API compatible).
// Otherwise the text is sent as a form field via POST.
func Send(webhookURL, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		req *http.Request
		err error
	)

	if strings.Contains(webhookURL, "{text}") {
		finalURL := strings.ReplaceAll(webhookURL, "{text}", url.QueryEscape(text))
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, finalURL, nil)
	} else {
		body := url.Values{"text": {text}}
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, strings.NewReader(body.Encode()))
	}

	if err == nil && !strings.Contains(webhookURL, "{text}") {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

package unosend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/robiuzzaman4/daily-durood-api/internal/application/reminder"
)

const (
	defaultBaseURL = "https://api.unosend.com"
	sendPath       = "/v1/emails"
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) Send(ctx context.Context, email reminder.OutboundEmail) error {
	payload := map[string]any{
		"from":    email.From,
		"to":      []string{email.To},
		"subject": email.Subject,
		"text":    email.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+sendPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build email request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute email request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	responseBody, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	return fmt.Errorf("unosend api responded with %d: %s", res.StatusCode, strings.TrimSpace(string(responseBody)))
}

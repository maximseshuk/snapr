package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/maximseshuk/snapr/internal/config"
)

type Webhook struct {
	cfg    config.NotifierConfig
	client *http.Client
}

func NewWebhook(c config.NotifierConfig) (*Webhook, error) {
	return &Webhook{
		cfg:    c,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (w *Webhook) GetType() string { return "webhook" }

func (w *Webhook) Notify(ctx context.Context, ev Event) error {
	payload := map[string]any{
		"job":      ev.JobName,
		"success":  ev.Success,
		"duration": ev.Duration,
		"error":    ev.Error,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	method := w.cfg.Method
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequestWithContext(ctx, method, w.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("webhook status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

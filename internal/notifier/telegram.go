package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/maximseshuk/snapr/internal/config"
)

type Telegram struct {
	cfg    config.NotifierConfig
	client *http.Client
}

func NewTelegram(c config.NotifierConfig) (*Telegram, error) {
	return &Telegram{
		cfg:    c,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *Telegram) GetType() string { return "telegram" }

func (t *Telegram) Notify(ctx context.Context, ev Event) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.cfg.BotToken)

	form := url.Values{}
	form.Set("chat_id", t.cfg.ChatID)
	form.Set("text", buildMessage(ev))
	form.Set("parse_mode", "HTML")
	form.Set("link_preview_options", `{"is_disabled":true}`)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		var apiErr struct {
			Description string `json:"description"`
		}
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Description != "" {
			return fmt.Errorf("telegram: %s", apiErr.Description)
		}
		return fmt.Errorf("telegram status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func buildMessage(ev Event) string {
	var b strings.Builder
	if ev.Success {
		b.WriteString("✅ <b>Backup OK</b>\n")
	} else {
		b.WriteString("❌ <b>Backup FAILED</b>\n")
	}
	fmt.Fprintf(&b, "<b>Job:</b> <code>%s</code>\n", html.EscapeString(ev.JobName))
	fmt.Fprintf(&b, "<b>Duration:</b> %s\n", html.EscapeString(ev.Duration))
	if ev.Error != "" {
		fmt.Fprintf(&b, "\n<b>Error:</b>\n<pre>%s</pre>", html.EscapeString(ev.Error))
	}
	return b.String()
}

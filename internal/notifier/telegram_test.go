package notifier

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestTelegram_GetType(t *testing.T) {
	tg, _ := NewTelegram(config.NotifierConfig{})
	assert.Equal(t, "telegram", tg.GetType())
}

func TestBuildMessage_Success(t *testing.T) {
	msg := buildMessage(Event{JobName: "db", Success: true, Duration: "5s"})
	assert.Contains(t, msg, "✅")
	assert.Contains(t, msg, "<b>Backup OK</b>")
	assert.Contains(t, msg, "<code>db</code>")
	assert.Contains(t, msg, "<b>Duration:</b> 5s")
	assert.NotContains(t, msg, "<b>Error:</b>")
}

func TestBuildMessage_Failure(t *testing.T) {
	msg := buildMessage(Event{JobName: "db", Success: false, Duration: "1s", Error: "disk full"})
	assert.Contains(t, msg, "❌")
	assert.Contains(t, msg, "<b>Backup FAILED</b>")
	assert.Contains(t, msg, "<pre>disk full</pre>")
}

func TestBuildMessage_EscapesHTML(t *testing.T) {
	msg := buildMessage(Event{
		JobName:  "<script>alert(1)</script>",
		Success:  false,
		Duration: "1s",
		Error:    "5 < 10 & 3 > 1",
	})
	assert.NotContains(t, msg, "<script>")
	assert.Contains(t, msg, "&lt;script&gt;")
	assert.Contains(t, msg, "5 &lt; 10 &amp; 3 &gt; 1")
}

// Telegram builds the bot URL from cfg.BotToken; redirect the http.Client
// transport at an httptest server to capture the request.
func telegramWithCaptureServer(t *testing.T) (*Telegram, *httptest.Server, *captured) {
	t.Helper()
	cap := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.path = r.URL.Path
		cap.contentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		cap.form, _ = url.ParseQuery(string(body))
		w.WriteHeader(cap.status)
		if cap.respBody != "" {
			_, _ = w.Write([]byte(cap.respBody))
		}
	}))
	cap.status = http.StatusOK

	tg, _ := NewTelegram(config.NotifierConfig{BotToken: "TEST_TOKEN", ChatID: "12345"})
	tg.client.Transport = redirectTransport{base: srv.URL}
	return tg, srv, cap
}

type captured struct {
	path        string
	contentType string
	form        url.Values
	status      int
	respBody    string
}

type redirectTransport struct{ base string }

func (rt redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := url.Parse(rt.base)
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	return http.DefaultTransport.RoundTrip(req)
}

func TestTelegram_PostsFormEncodedRequest(t *testing.T) {
	tg, srv, cap := telegramWithCaptureServer(t)
	defer srv.Close()

	require.NoError(t, tg.Notify(context.Background(), Event{JobName: "db", Success: true, Duration: "3s"}))

	assert.Equal(t, "/botTEST_TOKEN/sendMessage", cap.path)
	assert.Equal(t, "application/x-www-form-urlencoded", cap.contentType)
	assert.Equal(t, "12345", cap.form.Get("chat_id"))
	assert.Equal(t, "HTML", cap.form.Get("parse_mode"))
	assert.Contains(t, cap.form.Get("text"), "Backup OK")
	assert.Contains(t, cap.form.Get("link_preview_options"), `"is_disabled":true`)
}

func TestTelegram_PropagatesAPIError(t *testing.T) {
	tg, srv, cap := telegramWithCaptureServer(t)
	defer srv.Close()

	cap.status = http.StatusBadRequest
	cap.respBody = `{"ok":false,"error_code":400,"description":"chat not found"}`

	err := tg.Notify(context.Background(), Event{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat not found")
}

func TestTelegram_FallbackErrorWhenBodyOpaque(t *testing.T) {
	tg, srv, cap := telegramWithCaptureServer(t)
	defer srv.Close()

	cap.status = http.StatusBadGateway
	cap.respBody = "upstream down"

	err := tg.Notify(context.Background(), Event{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
	assert.Contains(t, err.Error(), "upstream down")
	_ = strings.HasPrefix // silence unused import false-positive guard
}

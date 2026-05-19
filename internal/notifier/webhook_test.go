package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestWebhook_GetType(t *testing.T) {
	w, _ := NewWebhook(config.NotifierConfig{})
	assert.Equal(t, "webhook", w.GetType())
}

func TestWebhook_PostsJSONPayload(t *testing.T) {
	var captured struct {
		method      string
		contentType string
		auth        string
		body        []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.contentType = r.Header.Get("Content-Type")
		captured.auth = r.Header.Get("Authorization")
		captured.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w, _ := NewWebhook(config.NotifierConfig{
		URL:     srv.URL,
		Headers: map[string]string{"Authorization": "Bearer xyz"},
	})
	require.NoError(t, w.Notify(context.Background(), Event{
		JobName:  "db-backup",
		Success:  true,
		Duration: "12s",
		Error:    "",
	}))

	assert.Equal(t, "POST", captured.method, "default method must be POST")
	assert.Equal(t, "application/json", captured.contentType)
	assert.Equal(t, "Bearer xyz", captured.auth, "custom header must be forwarded")

	var payload map[string]any
	require.NoError(t, json.Unmarshal(captured.body, &payload))
	assert.Equal(t, "db-backup", payload["job"])
	assert.Equal(t, true, payload["success"])
	assert.Equal(t, "12s", payload["duration"])
	assert.Equal(t, "", payload["error"])
}

func TestWebhook_CustomMethod(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method
	}))
	defer srv.Close()

	w, _ := NewWebhook(config.NotifierConfig{URL: srv.URL, Method: "PUT"})
	require.NoError(t, w.Notify(context.Background(), Event{}))
	assert.Equal(t, "PUT", got)
}

func TestWebhook_Non2xxReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server boom"))
	}))
	defer srv.Close()

	w, _ := NewWebhook(config.NotifierConfig{URL: srv.URL})
	err := w.Notify(context.Background(), Event{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "server boom")
}

func TestWebhook_RequestErrorPropagates(t *testing.T) {
	w, _ := NewWebhook(config.NotifierConfig{URL: "http://127.0.0.1:1/no-listener"})
	err := w.Notify(context.Background(), Event{})
	assert.Error(t, err)
}

func TestWebhook_ContextCancellation(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w, _ := NewWebhook(config.NotifierConfig{URL: srv.URL})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := w.Notify(ctx, Event{})
	require.Error(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&hits), "server must not receive request after pre-cancelled context")
}

package notifier

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestEmail_GetType(t *testing.T) {
	e, _ := NewEmail(config.NotifierConfig{})
	assert.Equal(t, "email", e.GetType())
}

func parseMIME(t *testing.T, raw string) (headers map[string]string, decodedBody string) {
	t.Helper()
	parts := strings.SplitN(raw, "\r\n\r\n", 2)
	require.Len(t, parts, 2, "MIME must have header/body separator")

	headers = map[string]string{}
	for _, line := range strings.Split(parts[0], "\r\n") {
		if i := strings.Index(line, ": "); i > 0 {
			headers[line[:i]] = line[i+2:]
		}
	}

	body, err := base64.StdEncoding.DecodeString(parts[1])
	require.NoError(t, err, "body must be valid base64")
	return headers, string(body)
}

func TestBuildMIME_Success(t *testing.T) {
	raw := buildMIME("snapr@example.com", []string{"ops@example.com", "alerts@example.com"}, Event{
		JobName:  "db-backup",
		Success:  true,
		Duration: "12s",
	})

	headers, body := parseMIME(t, raw)
	assert.Equal(t, "snapr@example.com", headers["From"])
	assert.Equal(t, "ops@example.com, alerts@example.com", headers["To"])
	assert.Contains(t, headers["Subject"], "✅")
	assert.Contains(t, headers["Subject"], "OK")
	assert.Contains(t, headers["Subject"], "db-backup")
	assert.Equal(t, "1.0", headers["MIME-Version"])
	assert.Equal(t, "text/plain; charset=utf-8", headers["Content-Type"])
	assert.Equal(t, "base64", headers["Content-Transfer-Encoding"])

	assert.Contains(t, body, "OK in 12s")
	assert.Contains(t, body, "Job:      db-backup")
	assert.Contains(t, body, "Duration: 12s")
	assert.NotContains(t, body, "Error:")
}

func TestBuildMIME_Failure(t *testing.T) {
	raw := buildMIME("snapr@example.com", []string{"ops@example.com"}, Event{
		JobName:  "db",
		Success:  false,
		Duration: "1s",
		Error:    "pg_dump exited 1",
	})

	headers, body := parseMIME(t, raw)
	assert.Contains(t, headers["Subject"], "❌")
	assert.Contains(t, headers["Subject"], "FAILED")
	assert.Contains(t, body, "FAILED in 1s")
	assert.Contains(t, body, "Error:\npg_dump exited 1")
}

func TestBuildMIME_HasCRLFLineEndings(t *testing.T) {
	raw := buildMIME("a@x", []string{"b@x"}, Event{JobName: "j", Success: true, Duration: "1s"})
	headerSection := strings.SplitN(raw, "\r\n\r\n", 2)[0]
	for _, line := range strings.Split(headerSection, "\r\n") {
		assert.NotContains(t, line, "\n", "headers must use CRLF only")
	}
}

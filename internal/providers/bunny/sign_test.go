package bunny

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimestamp(t *testing.T) {
	got, err := ParseTimestamp("2026-01-15T10:30:45.123")
	require.NoError(t, err)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 15, got.Day())
	assert.Equal(t, 10, got.Hour())

	_, err = ParseTimestamp("not a timestamp")
	assert.Error(t, err)
}

func TestSignURL_StructureAndExpires(t *testing.T) {
	before := time.Now().Unix()
	signed := SignURL(SignOptions{
		Hostname:    "myzone.b-cdn.net",
		SecurityKey: "secret-key",
		Path:        "/backup/file.tar.gz",
		TTL:         300,
	})
	after := time.Now().Unix()

	assert.True(t, strings.HasPrefix(signed, "https://myzone.b-cdn.net/backup/file.tar.gz?"), "got %q", signed)

	u, err := url.Parse(signed)
	require.NoError(t, err)
	assert.Equal(t, "myzone.b-cdn.net", u.Host)
	assert.Equal(t, "/backup/file.tar.gz", u.Path)

	token := u.Query().Get("token")
	assert.True(t, strings.HasPrefix(token, "HS256-"), "token must use HS256 prefix, got %q", token)

	expiresStr := u.Query().Get("expires")
	exp, err := strconv.ParseInt(expiresStr, 10, 64)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, exp, before+300)
	assert.LessOrEqual(t, exp, after+300)
}

func TestSignURL_DefaultTTL(t *testing.T) {
	before := time.Now().Unix()
	signed := SignURL(SignOptions{
		Hostname:    "z.b-cdn.net",
		SecurityKey: "k",
		Path:        "/f",
		TTL:         0,
	})
	u, _ := url.Parse(signed)
	exp, _ := strconv.ParseInt(u.Query().Get("expires"), 10, 64)
	assert.GreaterOrEqual(t, exp, before+3600, "default TTL must be 3600s")
}

func TestSignURL_DifferentKeysProduceDifferentTokens(t *testing.T) {
	a := SignURL(SignOptions{Hostname: "h", SecurityKey: "k1", Path: "/x", TTL: 60})
	b := SignURL(SignOptions{Hostname: "h", SecurityKey: "k2", Path: "/x", TTL: 60})
	tokA, _ := url.Parse(a)
	tokB, _ := url.Parse(b)
	assert.NotEqual(t, tokA.Query().Get("token"), tokB.Query().Get("token"))
}

func TestSignURL_NormalizesHost(t *testing.T) {
	cases := []string{"myzone.b-cdn.net", "https://myzone.b-cdn.net", "http://myzone.b-cdn.net/", "myzone.b-cdn.net/"}
	for _, host := range cases {
		signed := SignURL(SignOptions{Hostname: host, SecurityKey: "k", Path: "/f", TTL: 60})
		assert.True(t, strings.HasPrefix(signed, "https://myzone.b-cdn.net/f?"), "host %q produced %q", host, signed)
	}
}

func TestNormalizeHost(t *testing.T) {
	cases := map[string]string{
		"plain.host":    "plain.host",
		"https://host":  "host",
		"http://host":   "host",
		"host/":         "host",
		"https://host/": "host",
	}
	for in, want := range cases {
		assert.Equalf(t, want, normalizeHost(in), "normalizeHost(%q)", in)
	}
}

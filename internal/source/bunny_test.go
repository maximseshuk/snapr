package source

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/maximseshuk/snapr/internal/config"
	bunnyclient "github.com/maximseshuk/snapr/internal/providers/bunny"
)

func TestBunnySource_BuildStorageURL(t *testing.T) {
	b := NewBunnySource()
	cases := []struct {
		name   string
		source config.SourceConfig
		path   string
		want   string
	}{
		{
			"root path",
			config.SourceConfig{Endpoint: "https://storage.bunnycdn.com", ZoneName: "myzone"},
			"",
			"https://storage.bunnycdn.com/myzone",
		},
		{
			"endpoint with trailing slash",
			config.SourceConfig{Endpoint: "https://storage.bunnycdn.com/", ZoneName: "myzone"},
			"",
			"https://storage.bunnycdn.com/myzone",
		},
		{
			"subpath",
			config.SourceConfig{Endpoint: "https://storage.bunnycdn.com", ZoneName: "myzone"},
			"/backups/db",
			"https://storage.bunnycdn.com/myzone/backups/db",
		},
		{
			"slash path treated as root",
			config.SourceConfig{Endpoint: "https://storage.bunnycdn.com", ZoneName: "myzone"},
			"/",
			"https://storage.bunnycdn.com/myzone",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, b.buildStorageURL(c.source, c.path))
		})
	}
}

func TestBunnySource_BuildDownloadURL_DirectAccessKey(t *testing.T) {
	b := NewBunnySource()
	url, useAccessKey := b.buildDownloadURL(
		config.SourceConfig{
			Endpoint:  "https://storage.bunnycdn.com",
			ZoneName:  "myzone",
			AccessKey: "k",
		},
		bunnyclient.ListItem{Path: "/myzone/dir/", ObjectName: "file.bin"},
	)
	assert.True(t, useAccessKey, "direct download must require access key header")
	assert.Equal(t, "https://storage.bunnycdn.com/myzone/dir/file.bin", url)
}

func TestBunnySource_BuildDownloadURL_PullZoneSigned(t *testing.T) {
	b := NewBunnySource()
	url, useAccessKey := b.buildDownloadURL(
		config.SourceConfig{
			Endpoint:             "https://storage.bunnycdn.com",
			ZoneName:             "myzone",
			AccessKey:            "k",
			PullZoneHostname:     "myzone.b-cdn.net",
			PullZoneTokenAuthKey: "supersecret",
			PullZoneTokenTTL:     600,
		},
		bunnyclient.ListItem{Path: "/myzone/dir/", ObjectName: "file.bin"},
	)
	assert.False(t, useAccessKey, "signed URL must not use access key")
	assert.True(t, strings.HasPrefix(url, "https://myzone.b-cdn.net/dir/file.bin?"), "got %q", url)
	assert.Contains(t, url, "token=HS256-")
	assert.Contains(t, url, "expires=")
}

func TestBunnySource_GetRelativePath(t *testing.T) {
	b := NewBunnySource()
	cases := []struct {
		name   string
		file   bunnyclient.ListItem
		source config.SourceConfig
		want   string
	}{
		{
			"zone-rooted file",
			bunnyclient.ListItem{Path: "/myzone/", ObjectName: "a.bin"},
			config.SourceConfig{ZoneName: "myzone"},
			"a.bin",
		},
		{
			"with config path",
			bunnyclient.ListItem{Path: "/myzone/backups/", ObjectName: "a.bin"},
			config.SourceConfig{ZoneName: "myzone", Path: "/backups"},
			"a.bin",
		},
		{
			"nested under config path",
			bunnyclient.ListItem{Path: "/myzone/backups/2026/", ObjectName: "a.bin"},
			config.SourceConfig{ZoneName: "myzone", Path: "backups"},
			"2026/a.bin",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, b.getRelativePath(c.file, c.source))
		})
	}
}

func TestBunnySource_ShouldRetry(t *testing.T) {
	b := NewBunnySource()
	retryable := []string{"timeout while reading", "connection refused", "got 429", "502 bad gateway", "503", "504"}
	for _, msg := range retryable {
		assert.True(t, b.shouldRetry(&stringErr{msg}), "must retry %q", msg)
	}
	notRetryable := []string{"404 not found", "401 unauthorized", "json parse error"}
	for _, msg := range notRetryable {
		assert.False(t, b.shouldRetry(&stringErr{msg}), "must NOT retry %q", msg)
	}
}

type stringErr struct{ s string }

func (e *stringErr) Error() string { return e.s }

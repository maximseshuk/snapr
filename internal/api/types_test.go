package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestURLHost(t *testing.T) {
	cases := map[string]string{
		"":                           "",
		"not a url":                  "",
		"https://example.com/path":   "example.com",
		"https://example.com:8080/p": "example.com:8080",
		"http://localhost":           "localhost",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, urlHost(in))
		})
	}
}

func TestToJobDetail_BasicShape(t *testing.T) {
	cfg := &config.JobConfig{
		Name:           "myjob",
		Schedule:       "0 * * * *",
		DefaultStorage: "primary",
		Compression:    "tar.gz",
		Retention:      config.RetentionConfig{Last: 7},
		BeforeScript:   "echo hi",
		Sources: []config.SourceConfig{
			{Type: "postgresql", Host: "db", Port: 5432, Database: "app", Username: "u"},
		},
		Storages: []config.StorageConfig{
			{Name: "primary", Type: "local", Path: "/backups"},
		},
	}
	d := ToJobDetail(cfg)
	assert.Equal(t, "myjob", d.Name)
	assert.Equal(t, "0 * * * *", d.Schedule)
	assert.Equal(t, "primary", d.DefaultStorage)
	assert.Equal(t, "tar.gz", d.Compression)
	assert.True(t, d.HasBeforeScript)
	assert.False(t, d.HasAfterScript)
	assert.Equal(t, 7, d.Retention.Last)
	require.Len(t, d.Sources, 1)
	assert.Equal(t, "postgresql", d.Sources[0].Type)
	assert.Equal(t, 5432, d.Sources[0].Port)
	assert.Equal(t, "u", d.Sources[0].Username)
}

func TestToJobDetail_RedactsSecrets(t *testing.T) {
	cfg := &config.JobConfig{
		Name: "j", Retention: config.RetentionConfig{Last: 1},
		Sources: []config.SourceConfig{
			{Type: "mongodb", URI: "mongodb://u:secret@host/db"},
		},
		Storages: []config.StorageConfig{
			{Name: "s", Type: "sftp", Host: "h", Port: 22, Username: "u",
				PrivateKey: "-----BEGIN RSA PRIVATE KEY-----...", KnownHosts: "github.com ssh-rsa ..."},
			{Name: "w", Type: "webdav", URL: "https://user:pass@dav.example.com/path"},
		},
		Encryption: &config.EncryptionConfig{Type: "openssl", Cipher: "aes-256-cbc", Password: "topsecret"},
		Notifiers: []config.NotifierConfig{
			{Type: "webhook", URL: "https://hooks.example.com/abc?token=xxx"},
			{Type: "telegram", BotToken: "tok123", ChatID: "55"},
			{Type: "email", SMTPHost: "smtp.x", SMTPPass: "passwd", From: "f@x", To: []string{"t@x"}},
		},
	}
	d := ToJobDetail(cfg)

	// Source URI flagged but not exposed.
	assert.True(t, d.Sources[0].HasURI)

	// SFTP: key/known-hosts only as booleans, no value.
	assert.True(t, d.Storages[0].HasPrivateKey)
	assert.True(t, d.Storages[0].HasKnownHosts)

	// WebDAV: URL not exposed, only host.
	assert.True(t, d.Storages[1].HasURL)
	assert.Equal(t, "dav.example.com", d.Storages[1].URLHost)
	// Userinfo "user:pass@" gets parsed into URL.User which is NOT part of u.Host — verify not leaked.
	assert.NotContains(t, d.Storages[1].URLHost, "pass")
	assert.NotContains(t, d.Storages[1].URLHost, "user:")

	// Encryption: no password field in detail.
	require.NotNil(t, d.Encryption)
	assert.Equal(t, "openssl", d.Encryption.Type)
	assert.Equal(t, "aes-256-cbc", d.Encryption.Cipher)

	// Notifiers: only host for webhook, ChatID for telegram (no bot token), no SMTP pass for email.
	wh := d.Notifiers[0]
	assert.Equal(t, "webhook", wh.Type)
	assert.Equal(t, "hooks.example.com", wh.URLHost)

	tg := d.Notifiers[1]
	assert.Equal(t, "telegram", tg.Type)
	assert.Equal(t, "55", tg.ChatID)

	em := d.Notifiers[2]
	assert.Equal(t, "email", em.Type)
	assert.Equal(t, "smtp.x", em.SMTPHost)
	assert.Equal(t, []string{"t@x"}, em.To)
}

func TestToJobDetail_Split(t *testing.T) {
	cfg := &config.JobConfig{
		Name: "j", Retention: config.RetentionConfig{Last: 1},
		Split: &config.SplitConfig{ChunkSize: "100MB"},
	}
	d := ToJobDetail(cfg)
	require.NotNil(t, d.Split)
	assert.Equal(t, "100MB", d.Split.ChunkSize)
}

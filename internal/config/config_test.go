package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalJobYAML = `
jobs:
  - name: db
    schedule: "0 * * * *"
    sources:
      - type: local
        path: /data
    storages:
      - name: primary
        type: local
        path: /backups
    retention:
      last: 5
`

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "snapr.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestLoad_MinimalValid(t *testing.T) {
	path := writeConfig(t, minimalJobYAML)
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Len(t, cfg.Jobs, 1)
	assert.Equal(t, "db", cfg.Jobs[0].Name)
	assert.Equal(t, 5, cfg.Jobs[0].Retention.Last)

	assert.True(t, cfg.Server.Enabled, "server.enabled default")
	assert.Equal(t, "0.0.0.0:8080", cfg.Server.Address)
	assert.Equal(t, "en", cfg.Server.DefaultLanguage)
	assert.True(t, cfg.Server.UI.Enabled)
	assert.Equal(t, "./logs", cfg.Logs.Path)
	assert.Equal(t, 100, cfg.Logs.MaxSizeMB)
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoad_NoJobs(t *testing.T) {
	path := writeConfig(t, "jobs: []\n")
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoad_DuplicateStorageNames(t *testing.T) {
	yaml := `
jobs:
  - name: db
    schedule: "0 * * * *"
    sources:
      - type: local
        path: /data
    storages:
      - name: dup
        type: local
        path: /a
      - name: dup
        type: local
        path: /b
    retention:
      last: 1
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unique")
}

func TestLoad_InvalidSplitChunkSize(t *testing.T) {
	yaml := `
jobs:
  - name: db
    schedule: "0 * * * *"
    sources:
      - type: local
        path: /data
    storages:
      - name: s
        type: local
        path: /a
    retention:
      last: 1
    split:
      chunkSize: "not-a-size"
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestLoad_SignedDownloadOnlyForS3(t *testing.T) {
	yaml := `
jobs:
  - name: db
    schedule: "0 * * * *"
    sources:
      - type: local
        path: /data
    storages:
      - name: local-bad
        type: local
        path: /a
        downloadMode: signed
    retention:
      last: 1
`
	path := writeConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "downloadmode")
}

func TestLoad_EnvFromFlagPath(t *testing.T) {
	yaml := `
jobs:
  - name: db
    schedule: "0 * * * *"
    sources:
      - type: local
        path: /data
    storages:
      - name: s
        type: local
        path: /a
    retention:
      last: 1
    encryption:
      type: openssl
      password: env:SNAPR_TEST_ENC
`
	t.Setenv("SNAPR_TEST_ENC", "s3cret")
	path := writeConfig(t, yaml)
	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "s3cret", cfg.Jobs[0].Encryption.Password)
}

func TestLoad_EnvVarMissing(t *testing.T) {
	yaml := `
jobs:
  - name: db
    schedule: "0 * * * *"
    sources:
      - type: local
        path: /data
    storages:
      - name: s
        type: local
        path: /a
    retention:
      last: 1
    encryption:
      type: openssl
      password: env:SNAPR_DEFINITELY_UNSET_VAR
`
	_ = os.Unsetenv("SNAPR_DEFINITELY_UNSET_VAR")
	path := writeConfig(t, yaml)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SNAPR_DEFINITELY_UNSET_VAR")
}

func TestResolveEnvString(t *testing.T) {
	t.Setenv("SNAPR_TEST_X", "hello")

	got, err := resolveEnvString("plain")
	require.NoError(t, err)
	assert.Equal(t, "plain", got)

	got, err = resolveEnvString("env:SNAPR_TEST_X")
	require.NoError(t, err)
	assert.Equal(t, "hello", got)

	_, err = resolveEnvString("env:")
	assert.Error(t, err)

	_, err = resolveEnvString("env:SNAPR_UNSET_FOR_TEST")
	assert.Error(t, err)
}

func TestResolveEnvRefs_NestedStructAndMap(t *testing.T) {
	t.Setenv("SNAPR_TEST_PWD", "p@ss")
	t.Setenv("SNAPR_TEST_HDR", "Bearer x")

	cfg := Config{
		Jobs: []JobConfig{{
			Encryption: &EncryptionConfig{Password: "env:SNAPR_TEST_PWD"},
			Notifiers: []NotifierConfig{{
				Type:    "webhook",
				URL:     "https://example.com",
				Headers: map[string]string{"Authorization": "env:SNAPR_TEST_HDR"},
			}},
		}},
	}
	require.NoError(t, resolveEnvRefs(reflect.ValueOf(&cfg).Elem()))
	assert.Equal(t, "p@ss", cfg.Jobs[0].Encryption.Password)
	assert.Equal(t, "Bearer x", cfg.Jobs[0].Notifiers[0].Headers["Authorization"])
}

func TestLoad_FromEnvVarPath(t *testing.T) {
	path := writeConfig(t, minimalJobYAML)
	t.Setenv("SNAPR_CONFIG_FILE", path)
	cfg, err := Load("")
	require.NoError(t, err)
	assert.Len(t, cfg.Jobs, 1)
}

func TestLoad_EnvVarPathNotFound(t *testing.T) {
	t.Setenv("SNAPR_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))
	_, err := Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

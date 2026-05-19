package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

func newLocalFixture(t *testing.T) (*LocalStorage, *pkgconfig.JobConfig, pkgconfig.StorageConfig, string) {
	t.Helper()
	dir := t.TempDir()
	return NewLocalStorage(),
		&pkgconfig.JobConfig{Name: "myjob"},
		pkgconfig.StorageConfig{Name: "primary", Type: "local", Path: dir},
		dir
}

func writeArchive(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestLocalStorage_EnsureJobDir(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	require.NoError(t, s.EnsureJobDir(context.Background(), job, cfg))
	_, err := os.Stat(filepath.Join(base, "myjob"))
	assert.NoError(t, err)
}

func TestLocalStorage_UploadInto(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	archive := writeArchive(t, "payload-bytes")

	require.NoError(t, s.UploadInto(context.Background(), archive, job, "", cfg))

	got, err := os.ReadFile(filepath.Join(base, "myjob", "archive.tar.gz"))
	require.NoError(t, err)
	assert.Equal(t, "payload-bytes", string(got))

	// no .tmp leftover
	entries, _ := os.ReadDir(filepath.Join(base, "myjob"))
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp")
	}
}

func TestLocalStorage_UploadInto_WrapperDir(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	archive := writeArchive(t, "p")

	wrapper := SplitWrapperName("archive.tar.gz", 1, 1)
	require.NoError(t, s.UploadInto(context.Background(), archive, job, wrapper, cfg))

	_, err := os.Stat(filepath.Join(base, "myjob", wrapper, "archive.tar.gz"))
	assert.NoError(t, err)
}

func TestLocalStorage_ListFiles(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	jobDir := filepath.Join(base, "myjob")
	require.NoError(t, os.MkdirAll(jobDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(jobDir, "a.tar.gz"), []byte("aa"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(jobDir, "b.tar.gz"), []byte("bbbb"), 0644))

	wrapper := SplitWrapperName("c.tar.gz", 2, 100)
	require.NoError(t, os.MkdirAll(filepath.Join(jobDir, wrapper), 0755))

	// Non-wrapper directory must be skipped.
	require.NoError(t, os.MkdirAll(filepath.Join(jobDir, "random-dir"), 0755))

	got, err := s.ListFiles(context.Background(), job, cfg)
	require.NoError(t, err)

	names := map[string]FileInfo{}
	for _, f := range got {
		names[f.Name] = f
	}
	require.Len(t, names, 3)
	assert.Equal(t, int64(2), names["a.tar.gz"].Size)
	assert.Equal(t, int64(4), names["b.tar.gz"].Size)
	assert.True(t, names[wrapper].Wrapper)
	assert.False(t, names["a.tar.gz"].Wrapper)
}

func TestLocalStorage_ListFiles_MissingDirReturnsEmpty(t *testing.T) {
	s, job, cfg, _ := newLocalFixture(t)
	got, err := s.ListFiles(context.Background(), job, cfg)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestLocalStorage_ListWrapperParts(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	wrapper := SplitWrapperName("c.tar.gz", 2, 100)
	dir := filepath.Join(base, "myjob", wrapper)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.tar.gz.part-aaa"), []byte("xx"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.tar.gz.part-aab"), []byte("yyy"), 0644))

	parts, err := s.ListWrapperParts(context.Background(), job, wrapper, cfg)
	require.NoError(t, err)
	assert.Len(t, parts, 2)
}

func TestLocalStorage_Delete(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	jobDir := filepath.Join(base, "myjob")
	require.NoError(t, os.MkdirAll(jobDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(jobDir, "a.tar.gz"), []byte("x"), 0644))

	require.NoError(t, s.DeleteFile(context.Background(), job, "a.tar.gz", cfg))
	_, err := os.Stat(filepath.Join(jobDir, "a.tar.gz"))
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_DeleteWrapper(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	wrapper := SplitWrapperName("c.tar.gz", 1, 5)
	dir := filepath.Join(base, "myjob", wrapper)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0644))

	require.NoError(t, s.DeleteWrapper(context.Background(), job, wrapper, cfg))
	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Download(t *testing.T) {
	s, job, cfg, base := newLocalFixture(t)
	jobDir := filepath.Join(base, "myjob")
	require.NoError(t, os.MkdirAll(jobDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(jobDir, "a.tar.gz"), []byte("payload"), 0644))

	res, err := s.Download(context.Background(), job, "", "a.tar.gz", cfg)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, int64(7), res.Size)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(body))
}

func TestLocalStorage_Download_MissingErrors(t *testing.T) {
	s, job, cfg, _ := newLocalFixture(t)
	_, err := s.Download(context.Background(), job, "", "nope", cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

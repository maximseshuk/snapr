package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLocalUpToDate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.bin")
	data := []byte("hello world")
	require.NoError(t, os.WriteFile(path, data, 0644))

	stat, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, IsLocalUpToDate(path, int64(len(data)), time.Time{}), "matching size, zero mtime")
	assert.True(t, IsLocalUpToDate(path, int64(len(data)), stat.ModTime().Add(-time.Hour)), "remote older than local")
	assert.False(t, IsLocalUpToDate(path, int64(len(data)), stat.ModTime().Add(time.Hour)), "remote newer than local")
	assert.False(t, IsLocalUpToDate(path, int64(len(data))+1, time.Time{}), "size mismatch")
	assert.False(t, IsLocalUpToDate(filepath.Join(dir, "missing"), 0, time.Time{}), "missing file")
}

func TestSafeJoin(t *testing.T) {
	base := t.TempDir()

	t.Run("normal relative", func(t *testing.T) {
		got, err := SafeJoin(base, "sub/file.txt")
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(got, base))
	})

	t.Run("empty rel returns base", func(t *testing.T) {
		got, err := SafeJoin(base, "")
		require.NoError(t, err)
		abs, _ := filepath.Abs(base)
		assert.Equal(t, abs, got)
	})

	t.Run("traversal blocked", func(t *testing.T) {
		_, err := SafeJoin(base, "../etc/passwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path traversal")
	})

	t.Run("absolute rel joined under base", func(t *testing.T) {
		got, err := SafeJoin(base, "/etc/passwd")
		require.NoError(t, err)
		abs, _ := filepath.Abs(base)
		assert.True(t, strings.HasPrefix(got, abs))
	})
}

func TestIsExcluded(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		excludes []string
		want     bool
	}{
		{"empty excludes", "foo/bar", nil, false},
		{"empty pattern skipped", "foo/bar", []string{""}, false},
		{"exact filename", "a/b/test.log", []string{"test.log"}, true},
		{"glob filename", "a/b/x.tmp", []string{"*.tmp"}, true},
		{"glob mismatch", "a/b/x.txt", []string{"*.tmp"}, false},
		{"directory trailing slash exact", "node_modules/foo", []string{"node_modules/"}, true},
		{"plain literal segment match", "a/node_modules/x", []string{"node_modules"}, true},
		{"deep prefix **/", "src/foo/bar.log", []string{"**/bar.log"}, true},
		{"double-star within path", "src/foo/bar/baz.log", []string{"**/bar/**"}, true},
		{"no match", "a/b/c", []string{"x"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, IsExcluded(c.path, c.excludes))
		})
	}
}

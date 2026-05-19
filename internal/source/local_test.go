package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestLocalSource_SymlinksWhenNoExcludes(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0644))

	dest := filepath.Join(t.TempDir(), "dest")
	err := NewLocalSource().Backup(context.Background(), dest, config.SourceConfig{Path: src})
	require.NoError(t, err)

	info, err := os.Lstat(dest)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "dest must be a symlink when no excludes")

	got, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(got))
}

func TestLocalSource_CopiesWhenExcludes(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "keep.txt"), []byte("k"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "skip.log"), []byte("s"), 0644))

	dest := filepath.Join(t.TempDir(), "dest")
	err := NewLocalSource().Backup(context.Background(), dest, config.SourceConfig{
		Path:     src,
		Excludes: []string{"*.log"},
	})
	require.NoError(t, err)

	info, err := os.Lstat(dest)
	require.NoError(t, err)
	assert.False(t, info.Mode()&os.ModeSymlink != 0, "dest must be a real dir when excludes set")

	_, err = os.Stat(filepath.Join(dest, "keep.txt"))
	assert.NoError(t, err, "keep.txt must be copied")
	_, err = os.Stat(filepath.Join(dest, "skip.log"))
	assert.True(t, os.IsNotExist(err), "skip.log must be excluded")
}

func TestLocalSource_MissingSourceErrors(t *testing.T) {
	err := NewLocalSource().Backup(context.Background(), t.TempDir(), config.SourceConfig{
		Path: "/does/not/exist/anywhere",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestLocalSource_SourceIsFileErrors(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))
	err := NewLocalSource().Backup(context.Background(), t.TempDir(), config.SourceConfig{Path: f})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

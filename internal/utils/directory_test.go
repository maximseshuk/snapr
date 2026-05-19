package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveEmptyDirs(t *testing.T) {
	root := t.TempDir()
	emptyA := filepath.Join(root, "emptyA")
	emptyB := filepath.Join(root, "nested", "emptyB")
	withFile := filepath.Join(root, "withFile")

	require.NoError(t, os.MkdirAll(emptyA, 0755))
	require.NoError(t, os.MkdirAll(emptyB, 0755))
	require.NoError(t, os.MkdirAll(withFile, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(withFile, "f.txt"), []byte("x"), 0644))

	RemoveEmptyDirs(root)

	_, err := os.Stat(emptyA)
	assert.True(t, os.IsNotExist(err), "emptyA should be removed")
	_, err = os.Stat(emptyB)
	assert.True(t, os.IsNotExist(err), "emptyB should be removed")
	_, err = os.Stat(withFile)
	assert.NoError(t, err, "withFile should remain")

	_, err = os.Stat(root)
	assert.NoError(t, err, "root must remain even if empty after walk")
}

func TestCalculateDirectorySize(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.bin"), make([]byte, 100), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "sub", "b.bin"), make([]byte, 250), 0644))

	got := CalculateDirectorySize(root)
	assert.Equal(t, int64(350), got)
}

func TestCalculateDirectorySize_Empty(t *testing.T) {
	root := t.TempDir()
	assert.Equal(t, int64(0), CalculateDirectorySize(root))
}

func TestCalculateDirectorySize_SymlinkToFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.bin")
	require.NoError(t, os.WriteFile(target, make([]byte, 200), 0644))

	link := filepath.Join(root, "link.bin")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}

	got := CalculateDirectorySize(root)
	assert.Equal(t, int64(200), got, "hardlink/symlink to same inode counted once")
}

package compression

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZipCompressorMetadata(t *testing.T) {
	c := NewZipCompressor()
	assert.Equal(t, "zip", c.GetType())
	assert.Equal(t, ".zip", c.GetExtension())
}

func TestZipCompressRoundTrip(t *testing.T) {
	if !lookPathOK("zip") {
		t.Skip("zip not on PATH")
	}

	sourcesDir := t.TempDir()
	tmpDir := t.TempDir()
	payload := []byte("hello snapr\n")
	require.NoError(t, os.WriteFile(filepath.Join(sourcesDir, "file.txt"), payload, 0o600))

	out, err := NewZipCompressor().Compress(context.Background(), sourcesDir, tmpDir, "snap")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(out, ".zip"), "output %q should end with .zip", out)

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.NotZero(t, info.Size(), "archive must not be empty")
}

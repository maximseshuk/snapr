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

func TestTarCompressorMetadata(t *testing.T) {
	cases := []struct {
		new      func() *TarCompressor
		wantType string
		wantExt  string
	}{
		{NewTarCompressor, "tar", ".tar"},
		{NewTarGzCompressor, "tar.gz", ".tar.gz"},
		{NewTarZstCompressor, "tar.zst", ".tar.zst"},
		{NewTarXzCompressor, "tar.xz", ".tar.xz"},
	}
	for _, tc := range cases {
		c := tc.new()
		assert.Equal(t, tc.wantType, c.GetType())
		assert.Equal(t, tc.wantExt, c.GetExtension())
	}
}

func TestTarBuildArgsPlain(t *testing.T) {
	args, err := NewTarCompressor().buildArgs("/tmp/x.tar", "/data")
	require.NoError(t, err)
	assert.Equal(t, []string{"-chf", "/tmp/x.tar", "-C", "/data", "."}, args)
	for _, a := range args {
		assert.NotContains(t, a, "--use-compress-program", "plain tar must not pipe through a compressor")
	}
}

func TestTarBuildArgsParallel(t *testing.T) {
	cases := []struct {
		ctor       func() *TarCompressor
		binaryHint string
		wantFlag   string
	}{
		{NewTarZstCompressor, "zstd", "--use-compress-program=zstd -T0"},
		{NewTarXzCompressor, "xz", "--use-compress-program=xz -T0"},
	}
	for _, tc := range cases {
		if !lookPathOK(tc.binaryHint) {
			t.Logf("skipping %s: %s not on PATH", tc.ctor().format, tc.binaryHint)
			continue
		}
		c := tc.ctor()
		args, err := c.buildArgs("/tmp/x"+c.GetExtension(), "/data")
		require.NoError(t, err)
		assert.Contains(t, args, tc.wantFlag)
	}
}

func TestTarMissingBinaryError(t *testing.T) {
	t.Setenv("PATH", "")
	_, err := NewTarZstCompressor().compressProgram()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zstd")
	assert.Contains(t, err.Error(), "tar.zst")
}

func TestTarUnknownFormatError(t *testing.T) {
	_, err := (&TarCompressor{format: "tar.bogus"}).compressProgram()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tar format")
}

// tar.gz must prefer pigz and fall back to gzip.
func TestTarGzFallback(t *testing.T) {
	hasGzip := lookPathOK("gzip")
	hasPigz := lookPathOK("pigz")
	if !hasGzip && !hasPigz {
		t.Skip("neither gzip nor pigz on PATH")
	}

	prog, err := NewTarGzCompressor().compressProgram()
	require.NoError(t, err)

	switch {
	case hasPigz:
		assert.Equal(t, "pigz", prog)
	case hasGzip:
		assert.Equal(t, "gzip", prog)
	}
}

func TestTarCompressRoundTrip(t *testing.T) {
	if !lookPathOK("tar") {
		t.Skip("tar not on PATH")
	}

	sourcesDir := t.TempDir()
	tmpDir := t.TempDir()
	payload := []byte("hello snapr\n")
	require.NoError(t, os.WriteFile(filepath.Join(sourcesDir, "file.txt"), payload, 0o600))

	out, err := NewTarCompressor().Compress(context.Background(), sourcesDir, tmpDir, "snap")
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(out, ".tar"), "output %q should end with .tar", out)

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, info.Size(), int64(len(payload)), "archive smaller than payload")
}

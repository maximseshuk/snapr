package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeLines(t *testing.T, count int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.log")
	var buf bytes.Buffer
	for i := 0; i < count; i++ {
		buf.WriteString("line-")
		buf.WriteString(string(rune('a' + (i % 26))))
		buf.WriteByte('\n')
	}
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))
	return path
}

func TestTailFile_LastN(t *testing.T) {
	path := writeLines(t, 10)
	lines, err := TailFile(path, 3)
	require.NoError(t, err)
	require.Len(t, lines, 3)
	assert.Equal(t, "line-h", string(lines[0]))
	assert.Equal(t, "line-i", string(lines[1]))
	assert.Equal(t, "line-j", string(lines[2]))
}

func TestTailFile_NRequestedExceedsAvailable(t *testing.T) {
	path := writeLines(t, 3)
	lines, err := TailFile(path, 100)
	require.NoError(t, err)
	assert.Len(t, lines, 3)
}

func TestTailFile_NZeroOrNegative(t *testing.T) {
	path := writeLines(t, 5)
	for _, n := range []int{0, -1} {
		lines, err := TailFile(path, n)
		require.NoError(t, err)
		assert.Nil(t, lines)
	}
}

func TestTailFile_MissingReturnsNil(t *testing.T) {
	lines, err := TailFile(filepath.Join(t.TempDir(), "nope.log"), 5)
	require.NoError(t, err)
	assert.Nil(t, lines)
}

func TestTailFile_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.log")
	require.NoError(t, os.WriteFile(path, nil, 0644))

	lines, err := TailFile(path, 5)
	require.NoError(t, err)
	assert.Nil(t, lines)
}

func TestTailFile_NoTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-nl.log")
	require.NoError(t, os.WriteFile(path, []byte("a\nb\nc"), 0644))

	lines, err := TailFile(path, 2)
	require.NoError(t, err)
	require.Len(t, lines, 2)
	assert.Equal(t, "b", string(lines[0]))
	assert.Equal(t, "c", string(lines[1]))
}

func TestTailFile_LargeFileMultiChunk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	var buf bytes.Buffer
	const total = 5000
	for i := 0; i < total; i++ {
		buf.WriteString(strings.Repeat("x", 30))
		buf.WriteByte('\n')
	}
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0644))

	lines, err := TailFile(path, 50)
	require.NoError(t, err)
	assert.Len(t, lines, 50)
	for _, l := range lines {
		assert.Equal(t, 30, len(l))
	}
}

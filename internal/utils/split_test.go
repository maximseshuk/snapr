package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSize(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"1024", 1024, false},
		{"1KB", 1024, false},
		{"1 KiB", 1024, false},
		{"100MB", 100 * 1024 * 1024, false},
		{"1.5GB", int64(1.5 * 1024 * 1024 * 1024), false},
		{"2 GIB", 2 * 1024 * 1024 * 1024, false},
		{"", 0, true},
		{"abc", 0, true},
		{"-5MB", 0, true},
		{"5XB", 0, true},
		{"0", 0, true},
	}
	for _, c := range cases {
		got, err := ParseSize(c.in)
		if c.wantErr {
			assert.Errorf(t, err, "ParseSize(%q)", c.in)
			continue
		}
		require.NoErrorf(t, err, "ParseSize(%q)", c.in)
		assert.Equalf(t, c.want, got, "ParseSize(%q)", c.in)
	}
}

func TestPartSuffix(t *testing.T) {
	cases := map[int]string{
		0:     "aaa",
		1:     "aab",
		25:    "aaz",
		26:    "aba",
		676:   "baa",
		17575: "zzz",
	}
	for idx, want := range cases {
		got, err := PartSuffix(idx)
		require.NoErrorf(t, err, "PartSuffix(%d)", idx)
		assert.Equalf(t, want, got, "PartSuffix(%d)", idx)
	}
	_, err := PartSuffix(17576)
	assert.Error(t, err, "PartSuffix(17576) should overflow")
}

func TestSplitFile(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "archive.tar.gz")
	data := bytes.Repeat([]byte("ABCDE"), 1000) // 5000 bytes
	require.NoError(t, os.WriteFile(srcPath, data, 0644))

	parts, err := SplitFile(srcPath, 2000)
	require.NoError(t, err)
	require.Len(t, parts, 3)

	wantSuffixes := []string{".part-aaa", ".part-aab", ".part-aac"}
	for i, p := range parts {
		assert.Equal(t, "archive.tar.gz"+wantSuffixes[i], filepath.Base(p))
	}

	_, err = os.Stat(srcPath)
	assert.True(t, os.IsNotExist(err), "source must be removed after split")

	var got []byte
	for _, p := range parts {
		b, err := os.ReadFile(p)
		require.NoErrorf(t, err, "read part %s", p)
		got = append(got, b...)
	}
	assert.Equal(t, data, got, "concatenated parts must equal source")

	wantSizes := []int64{2000, 2000, 1000}
	for i, p := range parts {
		fi, err := os.Stat(p)
		require.NoError(t, err)
		assert.Equalf(t, wantSizes[i], fi.Size(), "part %d size", i)
	}
}

func TestSplitFile_ChunkLargerThanSource(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "small.bin")
	data := []byte("hello")
	require.NoError(t, os.WriteFile(srcPath, data, 0644))

	parts, err := SplitFile(srcPath, 10000)
	require.NoError(t, err)
	require.Len(t, parts, 1)

	got, err := os.ReadFile(parts[0])
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestIsPartName(t *testing.T) {
	cases := []struct {
		in       string
		wantBase string
		wantOK   bool
	}{
		{"archive.tar.gz.part-aaa", "archive.tar.gz", true},
		{"archive.tar.gz.part-zzz", "archive.tar.gz", true},
		{"archive.tar.gz", "", false},
		{"archive.tar.gz.part-aa", "", false},
		{"archive.tar.gz.part-aaaa", "", false},
		{"archive.tar.gz.part-AAA", "", false},
		{"archive.tar.gz.part-a1a", "", false},
	}
	for _, c := range cases {
		base, ok := IsPartName(c.in)
		assert.Equalf(t, c.wantOK, ok, "IsPartName(%q) ok", c.in)
		assert.Equalf(t, c.wantBase, base, "IsPartName(%q) base", c.in)
	}
}

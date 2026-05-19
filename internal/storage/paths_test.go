package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobDirPosix(t *testing.T) {
	cases := map[string]struct{ base, job, want string }{
		"empty base": {"", "myjob", "myjob"},
		"plain":      {"backups", "myjob", "backups/myjob"},
		"trailing":   {"backups/", "myjob", "backups/myjob"},
		"deep":       {"a/b/c", "myjob", "a/b/c/myjob"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.want, JobDirPosix(c.base, c.job))
		})
	}
}

func TestSplitWrapperRoundtrip(t *testing.T) {
	name := SplitWrapperName("backup-20260101.tar.gz", 5, 12345)
	assert.Equal(t, "backup-20260101.tar.gz.parts-5-12345", name)

	base, parts, size, ok := ParseSplitWrapper(name)
	assert.True(t, ok)
	assert.Equal(t, "backup-20260101.tar.gz", base)
	assert.Equal(t, 5, parts)
	assert.Equal(t, int64(12345), size)
}

func TestParseSplitWrapper_Invalid(t *testing.T) {
	cases := []string{
		"plain-archive.tar.gz",
		"file.parts-",
		"file.parts-abc-100",
		"file.parts-5-abc",
		"file.parts-0-100",
		"file.parts-5--1",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, _, _, ok := ParseSplitWrapper(in)
			assert.False(t, ok, "must reject %q", in)
		})
	}
}

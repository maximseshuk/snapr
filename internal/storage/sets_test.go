package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupBackupSets(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)

	splitWrapper := SplitWrapperName("mybackup-20260102-100000.tar.gz", 3, 500)

	files := []FileInfo{
		{Name: "other-job-20260101.tar.gz", Size: 100, LastModified: t1},
		{Name: "mybackup-20260101-100000.tar.gz", Size: 500, LastModified: t1},
		{Name: splitWrapper, Size: 500, LastModified: t2, Wrapper: true},
		{Name: "mybackup-20260103-100000.tar.gz", Size: 500, LastModified: t3},
	}

	sets := GroupBackupSets(files, "mybackup")
	require.Len(t, sets, 4)

	// Newest first; ties broken by ID.
	assert.Equal(t, "mybackup-20260103-100000.tar.gz", sets[0].ID)
	assert.False(t, sets[0].IsSplit)
	assert.Len(t, sets[0].Files, 1)

	assert.Equal(t, "mybackup-20260102-100000.tar.gz", sets[1].ID)
	assert.True(t, sets[1].IsSplit)
	assert.Equal(t, 3, sets[1].PartsCount)
	assert.Equal(t, int64(500), sets[1].TotalSize)
	assert.Equal(t, splitWrapper, sets[1].WrapperName)
	assert.Empty(t, sets[1].Files, "Files must be nil for split sets")

	assert.Equal(t, "mybackup-20260101-100000.tar.gz", sets[2].ID)
	assert.Equal(t, "other-job-20260101.tar.gz", sets[3].ID)
}

func TestSetCacheInvalidate(t *testing.T) {
	c := newSetCache()
	c.put("k", []BackupSet{{ID: "x"}}, time.Minute)
	sets, ok := c.get("k")
	require.True(t, ok)
	assert.Len(t, sets, 1)

	c.invalidate("k")
	_, ok = c.get("k")
	assert.False(t, ok, "expected cache miss after invalidate")
}

func TestSetCacheTTL(t *testing.T) {
	c := newSetCache()
	c.put("k", []BackupSet{{ID: "x"}}, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	_, ok := c.get("k")
	assert.False(t, ok, "expected cache miss after TTL")
}

package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/storage"
)

func TestFindJob(t *testing.T) {
	cfg := &config.Config{
		Jobs: []config.JobConfig{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		},
	}
	j := findJob(cfg, "b")
	require.NotNil(t, j)
	assert.Equal(t, "b", j.Name)

	assert.Nil(t, findJob(cfg, "missing"))
	assert.Nil(t, findJob(&config.Config{}, "any"))
}

func TestManager_ResolveStorage(t *testing.T) {
	bm := NewManager(&config.Config{})

	t.Run("no storages", func(t *testing.T) {
		_, ok := bm.resolveStorage(&config.JobConfig{})
		assert.False(t, ok)
	})

	t.Run("first by default", func(t *testing.T) {
		job := &config.JobConfig{Storages: []config.StorageConfig{
			{Name: "a", Type: "local"},
			{Name: "b", Type: "s3"},
		}}
		s, ok := bm.resolveStorage(job)
		require.True(t, ok)
		assert.Equal(t, "a", s.Name)
	})

	t.Run("default by name", func(t *testing.T) {
		job := &config.JobConfig{
			DefaultStorage: "b",
			Storages: []config.StorageConfig{
				{Name: "a", Type: "local"},
				{Name: "b", Type: "s3"},
			},
		}
		s, ok := bm.resolveStorage(job)
		require.True(t, ok)
		assert.Equal(t, "b", s.Name)
	})

	t.Run("default missing falls back to first", func(t *testing.T) {
		job := &config.JobConfig{
			DefaultStorage: "nonexistent",
			Storages: []config.StorageConfig{
				{Name: "a", Type: "local"},
				{Name: "b", Type: "s3"},
			},
		}
		s, ok := bm.resolveStorage(job)
		require.True(t, ok)
		assert.Equal(t, "a", s.Name)
	})
}

func TestBackupPath(t *testing.T) {
	t.Run("local single", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "local", Path: "/srv/backups"},
			"myjob",
			&storage.BackupSet{ID: "myjob-20260101.tar.gz"},
		)
		assert.Equal(t, "/srv/backups/myjob/myjob-20260101.tar.gz", got)
	})

	t.Run("local split", func(t *testing.T) {
		wrapper := storage.SplitWrapperName("myjob-20260101.tar.gz", 3, 500)
		got := backupPath(
			config.StorageConfig{Type: "local", Path: "/srv"},
			"myjob",
			&storage.BackupSet{ID: "myjob-20260101.tar.gz", IsSplit: true, WrapperName: wrapper},
		)
		assert.Equal(t, "/srv/myjob/"+wrapper, got)
	})

	t.Run("s3 with path", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "s3", Bucket: "mybucket", Path: "backups/"},
			"myjob",
			&storage.BackupSet{ID: "f.tar.gz"},
		)
		assert.Equal(t, "s3://mybucket/backups/myjob/f.tar.gz", got)
	})

	t.Run("s3 no path", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "s3", Bucket: "b"},
			"j",
			&storage.BackupSet{ID: "f"},
		)
		assert.Equal(t, "s3://b/j/f", got)
	})

	t.Run("other storage fallback", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "bunny"},
			"job",
			&storage.BackupSet{ID: "file"},
		)
		assert.Equal(t, "job/file", got)
	})

	noJobName := false

	t.Run("s3 includeJobName false drops job segment", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "s3", Bucket: "mybucket", Path: "video", IncludeJobName: &noJobName},
			"video",
			&storage.BackupSet{ID: "video-20260101.tar"},
		)
		assert.Equal(t, "s3://mybucket/video/video-20260101.tar", got)
	})

	t.Run("s3 includeJobName false no path", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "s3", Bucket: "b", IncludeJobName: &noJobName},
			"j",
			&storage.BackupSet{ID: "f"},
		)
		assert.Equal(t, "s3://b/f", got)
	})

	t.Run("local includeJobName false drops job segment", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "local", Path: "/srv/backups", IncludeJobName: &noJobName},
			"myjob",
			&storage.BackupSet{ID: "myjob-20260101.tar.gz"},
		)
		assert.Equal(t, "/srv/backups/myjob-20260101.tar.gz", got)
	})

	t.Run("fallback includeJobName false drops job segment", func(t *testing.T) {
		got := backupPath(
			config.StorageConfig{Type: "bunny", IncludeJobName: &noJobName},
			"job",
			&storage.BackupSet{ID: "file"},
		)
		assert.Equal(t, "file", got)
	})
}

func TestFullDownloadSupported(t *testing.T) {
	cases := []struct {
		name    string
		storage config.StorageConfig
		split   bool
		want    bool
	}{
		{"non-split always supported", config.StorageConfig{Type: "s3", DownloadMode: "signed"}, false, true},
		{"local split supported", config.StorageConfig{Type: "local"}, true, true},
		{"sftp split supported", config.StorageConfig{Type: "sftp"}, true, true},
		{"s3 proxy split supported", config.StorageConfig{Type: "s3", DownloadMode: "proxy"}, true, true},
		{"s3 signed split NOT supported", config.StorageConfig{Type: "s3", DownloadMode: "signed"}, true, false},
		{"bunny direct split supported", config.StorageConfig{Type: "bunny"}, true, true},
		{
			"bunny pullzone split NOT supported",
			config.StorageConfig{Type: "bunny", PullZoneHostname: "z.b-cdn.net", PullZoneTokenAuthKey: "k"},
			true, false,
		},
		{
			"bunny pullzone missing key still supported",
			config.StorageConfig{Type: "bunny", PullZoneHostname: "z.b-cdn.net"},
			true, true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, fullDownloadSupported(c.storage, c.split))
		})
	}
}

func TestManager_GetJobConfig(t *testing.T) {
	bm := NewManager(&config.Config{Jobs: []config.JobConfig{{Name: "j1"}, {Name: "j2"}}})

	j, err := bm.GetJobConfig("j2")
	require.NoError(t, err)
	assert.Equal(t, "j2", j.Name)

	_, err = bm.GetJobConfig("nope")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrJobNotFound)
}

func TestManager_ListJobsAndNames(t *testing.T) {
	bm := NewManager(&config.Config{Jobs: []config.JobConfig{{Name: "a"}, {Name: "b"}}})
	jobs := bm.ListJobs()
	require.Len(t, jobs, 2)
	assert.Equal(t, "a", jobs[0].Name)

	names := bm.ListJobNames()
	assert.Equal(t, []string{"a", "b"}, names)
}

func TestManager_IsJobActive_And_TryStartJob(t *testing.T) {
	bm := NewManager(&config.Config{Jobs: []config.JobConfig{{Name: "j"}}})

	assert.False(t, bm.IsJobActive("j"))
	assert.False(t, bm.TryStartJob("missing"), "TryStartJob must reject unknown jobs")

	require.True(t, bm.TryStartJob("j"))
	assert.True(t, bm.IsJobActive("j"))
	assert.False(t, bm.TryStartJob("j"), "second reservation must fail")
}

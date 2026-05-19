package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestSanitizeJobName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "_unknown"},
		{"simple", "simple"},
		{"My-Job_1.2", "My-Job_1.2"},
		{"a b/c", "a_b_c"},
		{"привет", "____________"},
		{"path/../etc", "path_.._etc"},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, sanitizeJobName(c.in), "sanitizeJobName(%q)", c.in)
	}
}

func TestExtractJob(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"with job field", `{"level":"info","job":"db-backup","msg":"x"}`, "db-backup"},
		{"no job field", `{"level":"info","msg":"x"}`, ""},
		{"empty job value", `{"job":""}`, ""},
		{"not json", `not json at all`, ""},
		{"job in nested only", `{"level":"info","data":{"job":"nope"}}`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, extractJob([]byte(c.in)))
		})
	}
}

func TestNewFileSinks_Disabled(t *testing.T) {
	fs, err := NewFileSinks(config.LogsConfig{System: false, PerJob: false})
	require.NoError(t, err)
	assert.Nil(t, fs)
	assert.False(t, fs.SystemEnabled())
	assert.False(t, fs.PerJobEnabled())
	assert.Empty(t, fs.SystemLogPath())
	assert.Empty(t, fs.JobLogPath("x"))
	fs.Close()
}

func TestNewFileSinks_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	logsDir := filepath.Join(dir, "logs")

	fs, err := NewFileSinks(config.LogsConfig{
		Path:   logsDir,
		System: true,
		PerJob: true,
	})
	require.NoError(t, err)
	require.NotNil(t, fs)
	defer fs.Close()

	_, err = os.Stat(logsDir)
	assert.NoError(t, err, "logs dir created")
	_, err = os.Stat(filepath.Join(logsDir, "jobs"))
	assert.NoError(t, err, "jobs dir created")

	assert.True(t, fs.SystemEnabled())
	assert.True(t, fs.PerJobEnabled())
	assert.Equal(t, filepath.Join(logsDir, "snapr.log"), fs.SystemLogPath())
	assert.Equal(t, filepath.Join(logsDir, "jobs", "my_job.log"), fs.JobLogPath("my/job"))
}

func TestFileSinks_WriteRoutesToSystemAndJob(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileSinks(config.LogsConfig{
		Path:   dir,
		System: true,
		PerJob: true,
	})
	require.NoError(t, err)
	defer fs.Close()

	event := []byte(`{"level":"info","job":"db-backup","msg":"hello"}` + "\n")
	n, err := fs.Write(event)
	require.NoError(t, err)
	assert.Equal(t, len(event), n)

	fs.Close()

	sys, err := os.ReadFile(filepath.Join(dir, "snapr.log"))
	require.NoError(t, err)
	assert.Contains(t, string(sys), "hello")

	jobLog, err := os.ReadFile(filepath.Join(dir, "jobs", "db-backup.log"))
	require.NoError(t, err)
	assert.Contains(t, string(jobLog), "hello")
}

func TestFileSinks_WriteWithoutJobFieldSkipsJobSink(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileSinks(config.LogsConfig{
		Path:   dir,
		System: true,
		PerJob: true,
	})
	require.NoError(t, err)
	defer fs.Close()

	_, err = fs.Write([]byte(`{"level":"info","msg":"sys-only"}` + "\n"))
	require.NoError(t, err)

	fs.Close()

	sys, err := os.ReadFile(filepath.Join(dir, "snapr.log"))
	require.NoError(t, err)
	assert.Contains(t, string(sys), "sys-only")

	entries, err := os.ReadDir(filepath.Join(dir, "jobs"))
	require.NoError(t, err)
	assert.Empty(t, entries, "no per-job logs should exist when job field absent")
}

func TestFileSinks_SystemOnly(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileSinks(config.LogsConfig{
		Path:   dir,
		System: true,
		PerJob: false,
	})
	require.NoError(t, err)
	defer fs.Close()

	assert.Empty(t, fs.JobLogPath("anything"), "JobLogPath empty when PerJob disabled")

	_, err = fs.Write([]byte(`{"job":"db","msg":"x"}` + "\n"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "jobs"))
	assert.True(t, os.IsNotExist(err), "jobs dir not created when PerJob disabled")
}

func TestFileSinks_DefaultPath(t *testing.T) {
	t.Chdir(t.TempDir())

	fs, err := NewFileSinks(config.LogsConfig{System: true})
	require.NoError(t, err)
	defer fs.Close()

	_, err = os.Stat("./logs")
	assert.NoError(t, err, "default ./logs dir created when Path empty")
}

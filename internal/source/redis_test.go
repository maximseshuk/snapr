package source

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestBuildRedisCliArgs_Defaults(t *testing.T) {
	bin, args, env := buildRedisCliArgs("/tmp/dump.rdb", config.SourceConfig{})
	assert.Equal(t, "redis-cli", bin)
	assert.Equal(t, []string{"-h", "127.0.0.1", "-p", "6379", "--rdb", "/tmp/dump.rdb"}, args)
	assert.Empty(t, env)
}

func TestBuildRedisCliArgs_HostPort(t *testing.T) {
	_, args, _ := buildRedisCliArgs("/x.rdb", config.SourceConfig{
		Host: "redis.internal",
		Port: 6390,
	})
	assert.Equal(t, []string{"-h", "redis.internal", "-p", "6390", "--rdb", "/x.rdb"}, args)
}

func TestBuildRedisCliArgs_Socket(t *testing.T) {
	_, args, _ := buildRedisCliArgs("/x.rdb", config.SourceConfig{
		Socket: "/run/redis.sock",
	})
	assert.Contains(t, args, "-s")
	assert.Contains(t, args, "/run/redis.sock")
	assert.NotContains(t, args, "-h")
	assert.NotContains(t, args, "-p")
}

func TestBuildRedisCliArgs_UserAndPassword(t *testing.T) {
	_, args, env := buildRedisCliArgs("/x.rdb", config.SourceConfig{
		Username: "alice",
		Password: "hunter2",
	})
	assert.Contains(t, args, "--user")
	assert.Contains(t, args, "alice")
	assert.Equal(t, []string{"REDISCLI_AUTH=hunter2"}, env)
}

func TestCopyRDB(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.rdb")
	dst := filepath.Join(dir, "dst.rdb")
	require.NoError(t, os.WriteFile(src, []byte("REDIS0009\x00\x00\x00"), 0644))

	logger := zerolog.Nop()
	require.NoError(t, copyRDB(src, dst, &logger))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("REDIS0009\x00\x00\x00"), got)
}

func TestCopyRDB_MissingSourceErrors(t *testing.T) {
	dir := t.TempDir()
	logger := zerolog.Nop()
	err := copyRDB(filepath.Join(dir, "nope"), filepath.Join(dir, "dst"), &logger)
	require.Error(t, err)
}

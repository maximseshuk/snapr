package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestBuildPgDumpArgs_Basic(t *testing.T) {
	binary, args, env, dump := buildPgDumpArgs("/tmp/backup", config.SourceConfig{
		Type:     "postgresql",
		Host:     "db.example.com",
		Port:     5432,
		Username: "user1",
		Password: "secret",
		Database: "my_db",
	})
	assert.Equal(t, "pg_dump", binary)
	assert.Equal(t, "/tmp/backup/my_db.sql", dump)
	assert.Equal(t, []string{
		"-d", "postgresql://user1@db.example.com:5432/my_db",
		"-f", "/tmp/backup/my_db.sql",
	}, args)
	assert.Equal(t, []string{"PGPASSWORD=secret"}, env)
}

func TestBuildPgDumpArgs_NoPasswordOmitsEnv(t *testing.T) {
	_, _, env, _ := buildPgDumpArgs("/x", config.SourceConfig{
		Host: "h", Port: 1, Username: "u", Database: "d",
	})
	assert.Empty(t, env)
}

func TestBuildPgDumpArgs_ExcludeTables(t *testing.T) {
	_, args, _, _ := buildPgDumpArgs("/x", config.SourceConfig{
		Host: "h", Port: 1, Username: "u", Database: "d",
		ExcludeTables: []string{"audit_log", "sessions"},
	})
	require.GreaterOrEqual(t, len(args), 8)
	assert.Contains(t, args, "--exclude-table")
	assert.Contains(t, args, "audit_log")
	assert.Contains(t, args, "sessions")
}

func TestBuildPgDumpArgs_ExtraParams(t *testing.T) {
	_, args, _, _ := buildPgDumpArgs("/x", config.SourceConfig{
		Host: "h", Port: 1, Username: "u", Database: "d",
		ExtraParams: map[string]string{
			"verbose":  "",
			"jobs":     "4",
			"no-owner": "",
			"encoding": "UTF8",
		},
	})
	assert.Contains(t, args, "--verbose")
	assert.Contains(t, args, "--no-owner")
	assert.Contains(t, args, "--jobs=4")
	assert.Contains(t, args, "--encoding=UTF8")
}

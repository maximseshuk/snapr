package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestBuildMysqlDumpArgs_Defaults(t *testing.T) {
	bin, args, env, dump, err := buildMysqlDumpArgs("mysqldump", "/tmp/out", config.SourceConfig{
		Username: "root",
		Password: "pw",
		Database: "shop",
	})
	require.NoError(t, err)
	assert.Equal(t, "mysqldump", bin)
	assert.Equal(t, "/tmp/out/shop.sql", dump)
	assert.Equal(t, []string{"MYSQL_PWD=pw"}, env)

	// Host/port defaults applied.
	assert.Contains(t, args, "--host")
	assert.Contains(t, args, "127.0.0.1")
	assert.Contains(t, args, "--port")
	assert.Contains(t, args, "3306")
	assert.Contains(t, args, "-u")
	assert.Contains(t, args, "root")
	assert.Contains(t, args, "shop")
	assert.Contains(t, args, "--result-file=/tmp/out/shop.sql")
}

func TestBuildMysqlDumpArgs_Socket(t *testing.T) {
	_, args, _, _, err := buildMysqlDumpArgs("mysqldump", "/x", config.SourceConfig{
		Socket:   "/var/run/mysqld/mysqld.sock",
		Database: "d",
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--socket")
	assert.Contains(t, args, "/var/run/mysqld/mysqld.sock")
	assert.NotContains(t, args, "--host", "socket mode must skip host")
	assert.NotContains(t, args, "--port", "socket mode must skip port")
}

func TestBuildMysqlDumpArgs_AllDatabases(t *testing.T) {
	_, args, _, dump, err := buildMysqlDumpArgs("mysqldump", "/x", config.SourceConfig{
		AllDatabases: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "/x/all-databases.sql", dump)
	assert.Contains(t, args, "--all-databases")
	assert.Contains(t, args, "--result-file=/x/all-databases.sql")
}

func TestBuildMysqlDumpArgs_MissingDatabaseErrors(t *testing.T) {
	_, _, _, _, err := buildMysqlDumpArgs("mysqldump", "/x", config.SourceConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database or allDatabases")
}

func TestBuildMysqlDumpArgs_ExcludeTablesAndTables(t *testing.T) {
	_, args, _, _, err := buildMysqlDumpArgs("mysqldump", "/x", config.SourceConfig{
		Database:      "shop",
		ExcludeTables: []string{"audit"},
		Tables:        []string{"users", "orders"},
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--ignore-table=shop.audit")
	assert.Contains(t, args, "users")
	assert.Contains(t, args, "orders")
}

func TestBuildMysqlDumpArgs_NoPasswordOmitsEnv(t *testing.T) {
	_, _, env, _, err := buildMysqlDumpArgs("mysqldump", "/x", config.SourceConfig{Database: "d"})
	require.NoError(t, err)
	assert.Empty(t, env)
}

func TestBuildMysqlDumpArgs_BinaryPassthrough(t *testing.T) {
	bin, _, _, _, err := buildMysqlDumpArgs("mariadb-dump", "/x", config.SourceConfig{Database: "d"})
	require.NoError(t, err)
	assert.Equal(t, "mariadb-dump", bin)
}

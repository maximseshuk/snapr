package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestBuildMongoDumpArgs_Defaults(t *testing.T) {
	bin, args, err := buildMongoDumpArgs("/tmp/out", config.SourceConfig{
		Database: "users",
	})
	require.NoError(t, err)
	assert.Equal(t, "mongodump", bin)
	assert.Contains(t, args, "--out=/tmp/out")
	assert.Contains(t, args, "--host=127.0.0.1")
	assert.Contains(t, args, "--port=27017")
	assert.Contains(t, args, "--db=users")
}

func TestBuildMongoDumpArgs_URI(t *testing.T) {
	_, args, err := buildMongoDumpArgs("/x", config.SourceConfig{
		URI: "mongodb://u:p@h:27017/admin",
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--uri=mongodb://u:p@h:27017/admin")
	// host/port must not be set when URI is given
	for _, a := range args {
		assert.NotContains(t, a, "--host=", "host flag must not appear with URI")
		assert.NotContains(t, a, "--port=", "port flag must not appear with URI")
	}
}

func TestBuildMongoDumpArgs_AllDatabases(t *testing.T) {
	_, args, err := buildMongoDumpArgs("/x", config.SourceConfig{
		AllDatabases: true,
		Username:     "admin",
		AuthDatabase: "admin",
	})
	require.NoError(t, err)
	assert.NotContains(t, args, "--db=")
	assert.Contains(t, args, "--username=admin")
	assert.Contains(t, args, "--authenticationDatabase=admin")
}

func TestBuildMongoDumpArgs_OplogAndExclude(t *testing.T) {
	_, args, err := buildMongoDumpArgs("/x", config.SourceConfig{
		Database:      "d",
		Oplog:         true,
		ExcludeTables: []string{"sessions", "cache"},
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--oplog")
	assert.Contains(t, args, "--excludeCollection=sessions")
	assert.Contains(t, args, "--excludeCollection=cache")
}

func TestBuildMongoDumpArgs_PasswordOnlyWithoutURI(t *testing.T) {
	_, args, err := buildMongoDumpArgs("/x", config.SourceConfig{
		Database: "d",
		Password: "secret",
	})
	require.NoError(t, err)
	assert.Contains(t, args, "--password=secret")

	_, args, err = buildMongoDumpArgs("/x", config.SourceConfig{
		URI:      "mongodb://h",
		Password: "secret",
	})
	require.NoError(t, err)
	for _, a := range args {
		assert.NotEqual(t, "--password=secret", a, "password must not duplicate when URI provided")
	}
}

func TestBuildMongoDumpArgs_MissingErrors(t *testing.T) {
	_, _, err := buildMongoDumpArgs("/x", config.SourceConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uri, database, or allDatabases")
}

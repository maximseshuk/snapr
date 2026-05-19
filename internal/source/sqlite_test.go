package source

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestSQLiteSource_PathRequired(t *testing.T) {
	err := NewSQLiteSource().Backup(context.Background(), t.TempDir(), config.SourceConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path is required")
}

func TestSQLiteSource_MissingFileErrors(t *testing.T) {
	err := NewSQLiteSource().Backup(context.Background(), t.TempDir(), config.SourceConfig{
		Path: "/this/path/definitely/does/not/exist.db",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access")
}

func TestSQLiteSource_GetType(t *testing.T) {
	assert.Equal(t, "sqlite", NewSQLiteSource().GetType())
}

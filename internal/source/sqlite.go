package source

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type SQLiteSource struct{}

func NewSQLiteSource() *SQLiteSource { return &SQLiteSource{} }

func (s *SQLiteSource) GetType() string { return "sqlite" }

func (s *SQLiteSource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	if source.Path == "" {
		return fmt.Errorf("sqlite: path is required")
	}
	if _, err := os.Stat(source.Path); err != nil {
		return fmt.Errorf("sqlite: cannot access %s: %w", source.Path, err)
	}

	dbName := strings.TrimSuffix(filepath.Base(source.Path), filepath.Ext(source.Path))
	dumpPath := filepath.Join(destDir, dbName+".sql")

	out, err := os.Create(dumpPath)
	if err != nil {
		return fmt.Errorf("create dump file: %w", err)
	}
	defer func() { _ = out.Close() }()

	cmd := exec.CommandContext(ctx, "sqlite3", source.Path, ".dump") //nolint:gosec // source.Path from trusted user config
	cmd.Stdout = out

	var stderr strings.Builder
	cmd.Stderr = &stderr

	logger.Info().Str("path", source.Path).Msg("Dumping SQLite")

	if err := cmd.Run(); err != nil {
		_ = os.Remove(dumpPath)
		return fmt.Errorf("sqlite3 .dump failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	if fi, err := os.Stat(dumpPath); err == nil {
		logger.Info().
			Str("dump_file", dumpPath).
			Int64("size_bytes", fi.Size()).
			Str("size_human", utils.FormatBytes(fi.Size())).
			Msg("SQLite dump completed")
	}

	return nil
}

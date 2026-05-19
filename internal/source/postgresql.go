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

type PostgreSQLSource struct{}

func NewPostgreSQLSource() *PostgreSQLSource {
	return &PostgreSQLSource{}
}

func (p *PostgreSQLSource) GetType() string {
	return "postgresql"
}

func buildPgDumpArgs(destDir string, source config.SourceConfig) (string, []string, []string, string) {
	dumpFile := filepath.Join(destDir, fmt.Sprintf("%s.sql", source.Database))

	connectionString := fmt.Sprintf("postgresql://%s@%s:%d/%s",
		source.Username,
		source.Host,
		source.Port,
		source.Database,
	)

	args := []string{
		"-d", connectionString,
		"-f", dumpFile,
	}

	for key, value := range source.ExtraParams {
		if value == "" {
			args = append(args, "--"+key)
		} else {
			args = append(args, "--"+key+"="+value)
		}
	}

	for _, table := range source.ExcludeTables {
		args = append(args, "--exclude-table", table)
	}

	var env []string
	if source.Password != "" {
		env = []string{"PGPASSWORD=" + source.Password}
	}
	return "pg_dump", args, env, dumpFile
}

func (p *PostgreSQLSource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	logger.Info().
		Str("database", source.Database).
		Str("host", source.Host).
		Int("port", source.Port).
		Msg("Dumping PostgreSQL")

	binary, args, extraEnv, dumpFile := buildPgDumpArgs(destDir, source)

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = nil

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrOutput := stderr.String()
		if stderrOutput != "" {
			logger.Error().
				Err(err).
				Str("stderr", stderrOutput).
				Msg("pg_dump failed")
			return fmt.Errorf("error executing pg_dump: %w\nStderr: %s", err, stderrOutput)
		}
		logger.Error().Err(err).Msg("pg_dump failed")
		return fmt.Errorf("error executing pg_dump: %w", err)
	}

	stat, err := os.Stat(dumpFile)
	if err != nil {
		return fmt.Errorf("stat dump file: %w", err)
	}
	logger.Info().
		Str("size", utils.FormatBytes(stat.Size())).
		Msg("PostgreSQL dump done")

	return nil
}

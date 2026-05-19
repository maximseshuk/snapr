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

type MySQLSource struct{}

func NewMySQLSource() *MySQLSource { return &MySQLSource{} }

func (m *MySQLSource) GetType() string { return "mysql" }

func (m *MySQLSource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	return mysqlDump(ctx, "mysqldump", destDir, source)
}

func buildMysqlDumpArgs(binary, destDir string, source config.SourceConfig) (string, []string, []string, string, error) {
	host := source.Host
	if host == "" && source.Socket == "" {
		host = "127.0.0.1"
	}
	port := source.Port
	if port == 0 && source.Socket == "" {
		port = 3306
	}

	if !source.AllDatabases && source.Database == "" {
		return "", nil, nil, "", fmt.Errorf("mysql: either database or allDatabases must be set")
	}

	dumpName := source.Database + ".sql"
	if source.AllDatabases {
		dumpName = "all-databases.sql"
	}
	dumpPath := filepath.Join(destDir, dumpName)

	args := []string{}
	if source.Socket != "" {
		args = append(args, "--socket", source.Socket)
	} else {
		args = append(args, "--host", host, "--port", fmt.Sprintf("%d", port))
	}
	if source.Username != "" {
		args = append(args, "-u", source.Username)
	}

	if source.AllDatabases {
		args = append(args, "--all-databases")
	} else {
		for _, t := range source.ExcludeTables {
			args = append(args, "--ignore-table="+source.Database+"."+t)
		}
	}

	for k, v := range source.ExtraParams {
		if v == "" {
			args = append(args, "--"+k)
		} else {
			args = append(args, "--"+k+"="+v)
		}
	}

	if !source.AllDatabases {
		args = append(args, source.Database)
		args = append(args, source.Tables...)
	}

	args = append(args, "--result-file="+dumpPath)

	var env []string
	if source.Password != "" {
		env = []string{"MYSQL_PWD=" + source.Password}
	}
	return binary, args, env, dumpPath, nil
}

// Shared by mysql and mariadb (mariadb-dump accepts the same flags).
func mysqlDump(ctx context.Context, binary, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	bin, args, extraEnv, dumpPath, err := buildMysqlDumpArgs(binary, destDir, source)
	if err != nil {
		return err
	}

	host := source.Host
	if host == "" && source.Socket == "" {
		host = "127.0.0.1"
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	logger.Info().
		Str("database", source.Database).
		Str("host", host).
		Bool("all_databases", source.AllDatabases).
		Msg("Dumping MySQL")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w (stderr: %s)", binary, err, strings.TrimSpace(stderr.String()))
	}

	if fi, err := os.Stat(dumpPath); err == nil {
		logger.Info().
			Str("size", utils.FormatBytes(fi.Size())).
			Msg("MySQL dump done")
	}

	return nil
}

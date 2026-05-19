package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type RedisSource struct{}

func NewRedisSource() *RedisSource { return &RedisSource{} }

func (r *RedisSource) GetType() string { return "redis" }

func buildRedisCliArgs(dumpPath string, source config.SourceConfig) (string, []string, []string) {
	host := source.Host
	if host == "" && source.Socket == "" {
		host = "127.0.0.1"
	}
	port := source.Port
	if port == 0 && source.Socket == "" {
		port = 6379
	}

	args := []string{}
	if source.Socket != "" {
		args = append(args, "-s", source.Socket)
	} else {
		args = append(args, "-h", host, "-p", fmt.Sprintf("%d", port))
	}
	if source.Username != "" {
		args = append(args, "--user", source.Username)
	}
	args = append(args, "--rdb", dumpPath)

	var env []string
	if source.Password != "" {
		env = []string{"REDISCLI_AUTH=" + source.Password}
	}
	return "redis-cli", args, env
}

func (r *RedisSource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	dumpPath := filepath.Join(destDir, "dump.rdb")

	// source.Path → copy the rdb file. Otherwise dump live via redis-cli.
	if source.Path != "" {
		return copyRDB(source.Path, dumpPath, logger)
	}

	binary, args, extraEnv := buildRedisCliArgs(dumpPath, source)

	host := source.Host
	if host == "" && source.Socket == "" {
		host = "127.0.0.1"
	}
	port := source.Port
	if port == 0 && source.Socket == "" {
		port = 6379
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr

	logger.Info().Str("host", host).Int("port", port).Msg("Dumping Redis")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("redis-cli --rdb failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	if fi, err := os.Stat(dumpPath); err == nil {
		logger.Info().
			Str("dump_file", dumpPath).
			Int64("size_bytes", fi.Size()).
			Str("size_human", utils.FormatBytes(fi.Size())).
			Msg("Redis dump completed")
	}
	return nil
}

func copyRDB(src, dst string, logger *zerolog.Logger) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open rdb %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dump: %w", err)
	}
	defer func() { _ = out.Close() }()

	n, err := io.Copy(out, in)
	if err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("copy rdb: %w", err)
	}
	logger.Info().Str("src", src).Str("dst", dst).Int64("size_bytes", n).Msg("Copied Redis RDB")
	return nil
}

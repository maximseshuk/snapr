package source

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
)

type MongoDBSource struct{}

func NewMongoDBSource() *MongoDBSource { return &MongoDBSource{} }

func (m *MongoDBSource) GetType() string { return "mongodb" }

func buildMongoDumpArgs(destDir string, source config.SourceConfig) (string, []string, error) {
	if source.URI == "" && !source.AllDatabases && source.Database == "" {
		return "", nil, fmt.Errorf("mongodb: uri, database, or allDatabases must be set")
	}

	args := []string{"--out=" + destDir}

	if source.URI != "" {
		args = append(args, "--uri="+source.URI)
	} else {
		host := source.Host
		if host == "" {
			host = "127.0.0.1"
		}
		port := source.Port
		if port == 0 {
			port = 27017
		}
		args = append(args, "--host="+host, fmt.Sprintf("--port=%d", port))
		if source.Username != "" {
			args = append(args, "--username="+source.Username)
		}
		if source.AuthDatabase != "" {
			args = append(args, "--authenticationDatabase="+source.AuthDatabase)
		}
		if !source.AllDatabases {
			args = append(args, "--db="+source.Database)
		}
	}

	if source.Oplog {
		args = append(args, "--oplog")
	}
	for _, t := range source.ExcludeTables {
		args = append(args, "--excludeCollection="+t)
	}
	for k, v := range source.ExtraParams {
		if v == "" {
			args = append(args, "--"+k)
		} else {
			args = append(args, "--"+k+"="+v)
		}
	}

	// mongodump has no password env — pass it via the URI instead of the cmdline.
	if source.URI == "" && source.Password != "" {
		args = append(args, "--password="+source.Password)
	}

	return "mongodump", args, nil
}

func (m *MongoDBSource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	binary, args, err := buildMongoDumpArgs(destDir, source)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = nil
	var stderr strings.Builder
	cmd.Stderr = &stderr

	logger.Info().
		Str("database", source.Database).
		Bool("all_databases", source.AllDatabases).
		Bool("oplog", source.Oplog).
		Msg("Dumping MongoDB")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongodump failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	logger.Info().Msg("MongoDB dump done")
	return nil
}

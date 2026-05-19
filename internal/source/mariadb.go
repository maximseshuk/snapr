package source

import (
	"context"
	"os/exec"

	"github.com/maximseshuk/snapr/internal/config"
)

type MariaDBSource struct{}

func NewMariaDBSource() *MariaDBSource { return &MariaDBSource{} }

func (m *MariaDBSource) GetType() string { return "mariadb" }

func (m *MariaDBSource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	binary := "mariadb-dump"
	if _, err := exec.LookPath(binary); err != nil {
		binary = "mysqldump"
	}
	return mysqlDump(ctx, binary, destDir, source)
}

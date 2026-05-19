package compression

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/utils"
)

type ZipCompressor struct{}

func NewZipCompressor() *ZipCompressor {
	return &ZipCompressor{}
}

func (z *ZipCompressor) GetType() string {
	return "zip"
}

func (z *ZipCompressor) GetExtension() string {
	return ".zip"
}

func (z *ZipCompressor) Compress(ctx context.Context, sourcesDir, tmpDir, archiveName string) (string, error) {
	logger := zerolog.Ctx(ctx)
	archivePath := filepath.Join(tmpDir, archiveName+z.GetExtension())

	if _, err := os.Stat(sourcesDir); err != nil {
		return "", fmt.Errorf("access sources dir: %w", err)
	}

	args := []string{"-r", archivePath, "."}
	logger.Debug().Str("sources_dir", sourcesDir).Str("archive", archivePath).Strs("args", args).Msg("Running zip")

	start := time.Now()
	cmd := exec.CommandContext(ctx, "zip", args...)
	cmd.Dir = sourcesDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("zip: %w", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return "", fmt.Errorf("stat archive: %w", err)
	}

	ratio := float64(0)
	if size := utils.CalculateDirectorySize(sourcesDir); size > 0 {
		ratio = float64(info.Size()) / float64(size) * 100
	}

	logger.Info().
		Str("size", utils.FormatBytes(info.Size())).
		Float64("ratio_percent", ratio).
		Dur("duration", time.Since(start)).
		Msg("Archive created")

	return archivePath, nil
}

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

type TarCompressor struct {
	format string
}

func NewTarCompressor() *TarCompressor    { return &TarCompressor{format: "tar"} }
func NewTarGzCompressor() *TarCompressor  { return &TarCompressor{format: "tar.gz"} }
func NewTarZstCompressor() *TarCompressor { return &TarCompressor{format: "tar.zst"} }
func NewTarXzCompressor() *TarCompressor  { return &TarCompressor{format: "tar.xz"} }

func (t *TarCompressor) GetType() string { return t.format }

func (t *TarCompressor) GetExtension() string {
	switch t.format {
	case "tar.gz":
		return ".tar.gz"
	case "tar.zst":
		return ".tar.zst"
	case "tar.xz":
		return ".tar.xz"
	default:
		return ".tar"
	}
}

func (t *TarCompressor) Compress(ctx context.Context, sourcesDir, tmpDir, archiveName string) (string, error) {
	logger := zerolog.Ctx(ctx)
	archivePath := filepath.Join(tmpDir, archiveName+t.GetExtension())

	if _, err := os.Stat(sourcesDir); err != nil {
		return "", fmt.Errorf("access sources dir: %w", err)
	}

	args, err := t.buildArgs(archivePath, sourcesDir)
	if err != nil {
		return "", err
	}

	logger.Debug().Str("sources_dir", sourcesDir).Str("archive", archivePath).Strs("args", args).Msg("Running " + t.format)

	start := time.Now()
	cmd := exec.CommandContext(ctx, "tar", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", t.format, err)
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

func (t *TarCompressor) buildArgs(archivePath, sourcesDir string) ([]string, error) {
	program, err := t.compressProgram()
	if err != nil {
		return nil, err
	}

	if program == "" {
		return []string{"-chf", archivePath, "-C", sourcesDir, "."}, nil
	}
	return []string{"--use-compress-program=" + program, "-chf", archivePath, "-C", sourcesDir, "."}, nil
}

func (t *TarCompressor) compressProgram() (string, error) {
	var binary, program string
	switch t.format {
	case "tar":
		return "", nil
	case "tar.gz":
		// pigz when available, otherwise plain gzip.
		if _, err := exec.LookPath("pigz"); err == nil {
			return "pigz", nil
		}
		binary, program = "gzip", "gzip"
	case "tar.zst":
		binary, program = "zstd", "zstd -T0"
	case "tar.xz":
		binary, program = "xz", "xz -T0"
	default:
		return "", fmt.Errorf("unknown tar format: %s", t.format)
	}

	if _, err := exec.LookPath(binary); err != nil {
		return "", fmt.Errorf("%s required for %s but not found in PATH", binary, t.format)
	}
	return program, nil
}

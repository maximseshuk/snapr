package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type LocalSource struct{}

func NewLocalSource() *LocalSource {
	return &LocalSource{}
}

func (l *LocalSource) GetType() string {
	return "local"
}

func (l *LocalSource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	logger.Info().Str("path", source.Path).Msg("Local source")

	sourceInfo, err := os.Stat(source.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("source directory does not exist: %s", source.Path)
	}
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	if !sourceInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", source.Path)
	}

	if _, err := os.Lstat(destDir); err == nil {
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("remove existing dest: %w", err)
		}
	}

	if len(source.Excludes) > 0 {
		logger.Debug().Strs("excludes", source.Excludes).Msg("Copying with filter")

		start := time.Now()
		if err := l.copyDirectory(ctx, source.Path, destDir, source.Excludes); err != nil {
			return fmt.Errorf("copy directory: %w", err)
		}
		logger.Info().Dur("duration", time.Since(start)).Msg("Copy done")
		return nil
	}

	absSource, err := filepath.Abs(source.Path)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}
	if err := os.Symlink(absSource, destDir); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	logger.Debug().Str("source", absSource).Str("target", destDir).Msg("Symlinked")

	return nil
}

func (l *LocalSource) copyDirectory(ctx context.Context, src, dest string, excludes []string) error {
	logger := zerolog.Ctx(ctx)

	copiedFiles := 0
	copiedDirs := 0
	skippedItems := 0
	var totalBytes int64

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Warn().Err(err).Str("path", path).Msg("Error accessing path during walk")
			return err
		}

		select {
		case <-ctx.Done():
			logger.Warn().Msg("Copy operation cancelled")
			return ctx.Err()
		default:
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			logger.Error().Err(err).Str("path", path).Msg("Error calculating relative path")
			return err
		}

		if utils.IsExcluded(relPath, excludes) {
			skippedItems++
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			if err := os.MkdirAll(destPath, info.Mode()); err != nil {
				logger.Error().Err(err).Str("directory", destPath).Msg("Error creating directory")
				return err
			}
			copiedDirs++
		} else {
			if err := l.copyFile(path, destPath, info.Mode()); err != nil {
				logger.Error().Err(err).
					Str("source", path).
					Str("destination", destPath).
					Msg("Error copying file")
				return err
			}
			copiedFiles++
			totalBytes += info.Size()

			if copiedFiles%1000 == 0 {
				logger.Debug().
					Int("copied_files", copiedFiles).
					Int("copied_dirs", copiedDirs).
					Int("skipped_items", skippedItems).
					Str("total_size", utils.FormatBytes(totalBytes)).
					Msg("Copy progress")
			}
		}

		return nil
	})
}

func (l *LocalSource) copyFile(src, dest string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer func() {
		_ = srcFile.Close()
	}()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("error creating destination directory: %w", err)
	}

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating destination file: %w", err)
	}
	defer func() {
		_ = destFile.Close()
	}()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("error copying file content: %w", err)
	}

	if err := os.Chmod(dest, mode); err != nil {
		return fmt.Errorf("error setting file permissions: %w", err)
	}

	return nil
}

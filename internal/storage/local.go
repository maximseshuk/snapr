package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type LocalStorage struct{}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{}
}

func (l *LocalStorage) GetType() string {
	return "local"
}

func (l *LocalStorage) EnsureJobDir(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) error {
	dir := JobDirLocal(storage.Path, job.Name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create job dir %s: %w", dir, err)
	}
	zerolog.Ctx(ctx).Debug().Str("path", dir).Msg("Local job dir ready")
	return nil
}

func (l *LocalStorage) UploadInto(ctx context.Context, archivePath string, job *pkgconfig.JobConfig, wrapperRelDir string, storage pkgconfig.StorageConfig) error {
	logger := zerolog.Ctx(ctx)

	jobDir := JobDirLocal(storage.Path, job.Name)
	destDir := jobDir
	if wrapperRelDir != "" {
		destDir = filepath.Join(jobDir, wrapperRelDir)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination directory %s: %w", destDir, err)
	}

	destPath := filepath.Join(destDir, filepath.Base(archivePath))
	tmpPath := destPath + ".tmp"

	uploadStart := time.Now()

	logger.Info().
		Str("archive_path", archivePath).
		Str("destination_path", destPath).
		Msg("Starting upload to local storage")

	srcFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("error creating destination file: %w", err)
	}

	success := false
	defer func() {
		_ = dstFile.Close()
		if !success {
			if rmErr := os.Remove(tmpPath); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
				logger.Warn().Err(rmErr).Str("path", tmpPath).Msg("Error removing partial upload")
			}
		}
	}()

	bytesCopied, err := dstFile.ReadFrom(srcFile)
	if err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("error syncing destination file: %w", err)
	}
	if err := dstFile.Close(); err != nil {
		return fmt.Errorf("error closing destination file: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("error finalizing destination file: %w", err)
	}
	success = true

	logger.Info().
		Str("archive_name", filepath.Base(archivePath)).
		Str("destination_path", destPath).
		Int64("bytes_copied", bytesCopied).
		Str("size_human", utils.FormatBytes(bytesCopied)).
		Dur("upload_duration", time.Since(uploadStart)).
		Msg("Successfully copied archive to local storage")
	return nil
}

func (l *LocalStorage) ListFiles(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	logger := zerolog.Ctx(ctx)
	dir := JobDirLocal(storage.Path, job.Name)

	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []FileInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if e.IsDir() {
			if _, _, _, ok := ParseSplitWrapper(e.Name()); !ok {
				continue
			}
			out = append(out, FileInfo{
				Name:         e.Name(),
				LastModified: info.ModTime(),
				Wrapper:      true,
			})
			continue
		}
		out = append(out, FileInfo{
			Name:         e.Name(),
			LastModified: info.ModTime(),
			Size:         info.Size(),
		})
	}

	logger.Debug().
		Int("total_files", len(out)).
		Str("storage_path", dir).
		Msg("Listed local files")
	return out, nil
}

func (l *LocalStorage) ListWrapperParts(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	dir := filepath.Join(JobDirLocal(storage.Path, job.Name), wrapperName)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []FileInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read wrapper %s: %w", dir, err)
	}
	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, FileInfo{
			Name:         e.Name(),
			LastModified: info.ModTime(),
			Size:         info.Size(),
		})
	}
	return out, nil
}

func (l *LocalStorage) DeleteFile(ctx context.Context, job *pkgconfig.JobConfig, fileName string, storage pkgconfig.StorageConfig) error {
	filePath := filepath.Join(JobDirLocal(storage.Path, job.Name), fileName)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("error removing file: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Str("file_path", filePath).Msg("Deleted local file")
	return nil
}

func (l *LocalStorage) DeleteWrapper(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) error {
	dir := filepath.Join(JobDirLocal(storage.Path, job.Name), wrapperName)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove wrapper %s: %w", dir, err)
	}
	zerolog.Ctx(ctx).Debug().Str("wrapper", dir).Msg("Deleted local split wrapper")
	return nil
}

func (l *LocalStorage) Download(ctx context.Context, job *pkgconfig.JobConfig, wrapperRelDir, fileName string, storage pkgconfig.StorageConfig) (*DownloadResult, error) {
	dir := JobDirLocal(storage.Path, job.Name)
	if wrapperRelDir != "" {
		dir = filepath.Join(dir, wrapperRelDir)
	}
	filePath := filepath.Join(dir, fileName)

	stat, err := os.Stat(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("file not found: %s", fileName)
	}
	if err != nil {
		return nil, fmt.Errorf("error stating file: %w", err)
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	zerolog.Ctx(ctx).Debug().Str("file_path", filePath).Msg("Opened local file for streaming")
	return &DownloadResult{Body: f, Size: stat.Size()}, nil
}

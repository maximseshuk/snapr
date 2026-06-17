package storage

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/studio-b12/gowebdav"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type WebDAVStorage struct{}

func NewWebDAVStorage() *WebDAVStorage { return &WebDAVStorage{} }

func (w *WebDAVStorage) GetType() string { return "webdav" }

func (w *WebDAVStorage) jobDir(storage pkgconfig.StorageConfig, jobName string) string {
	return path.Join(webdavDir(storage), JobNameSegment(storage.IncludeJobName, jobName))
}

func (w *WebDAVStorage) EnsureJobDir(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) error {
	client, err := openWebDAV(ctx, storage)
	if err != nil {
		return err
	}
	dir := w.jobDir(storage, job.Name)
	if err := client.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("webdav mkdir %s: %w", dir, err)
	}
	return nil
}

func (w *WebDAVStorage) UploadInto(ctx context.Context, archivePath string, job *pkgconfig.JobConfig, wrapperRelDir string, storage pkgconfig.StorageConfig) error {
	logger := zerolog.Ctx(ctx)

	client, err := openWebDAV(ctx, storage)
	if err != nil {
		return err
	}

	dir := w.jobDir(storage, job.Name)
	if wrapperRelDir != "" {
		dir = path.Join(dir, wrapperRelDir)
	}
	if err := client.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("webdav mkdir %s: %w", dir, err)
	}

	fileName := filepath.Base(archivePath)
	remoteFile := path.Join(dir, fileName)

	src, err := os.Open(archivePath) //nolint:gosec // archivePath comes from snapr's own pipeline
	if err != nil {
		return fmt.Errorf("open local archive: %w", err)
	}
	defer func() { _ = src.Close() }()

	stat, err := src.Stat()
	if err != nil {
		return err
	}

	uploadStart := time.Now()
	if err := client.WriteStream(remoteFile, src, 0o644); err != nil {
		return fmt.Errorf("webdav write %s: %w", remoteFile, err)
	}

	logger.Info().
		Str("remote", remoteFile).
		Int64("size_bytes", stat.Size()).
		Str("size_human", utils.FormatBytes(stat.Size())).
		Dur("duration", time.Since(uploadStart)).
		Msg("Uploaded archive over WebDAV")
	return nil
}

func (w *WebDAVStorage) ListFiles(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	client, err := openWebDAV(ctx, storage)
	if err != nil {
		return nil, err
	}

	dir := w.jobDir(storage, job.Name)
	entries, err := client.ReadDir(dir)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return []FileInfo{}, nil
		}
		return nil, fmt.Errorf("webdav readdir %s: %w", dir, err)
	}

	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			if _, _, _, ok := ParseSplitWrapper(e.Name()); !ok {
				continue
			}
			out = append(out, FileInfo{
				Name:         e.Name(),
				LastModified: e.ModTime(),
				Wrapper:      true,
			})
			continue
		}
		out = append(out, FileInfo{
			Name:         e.Name(),
			LastModified: e.ModTime(),
			Size:         e.Size(),
		})
	}
	return out, nil
}

func (w *WebDAVStorage) ListWrapperParts(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	client, err := openWebDAV(ctx, storage)
	if err != nil {
		return nil, err
	}

	dir := path.Join(w.jobDir(storage, job.Name), wrapperName)
	entries, err := client.ReadDir(dir)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return []FileInfo{}, nil
		}
		return nil, fmt.Errorf("webdav readdir %s: %w", dir, err)
	}

	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out = append(out, FileInfo{
			Name:         e.Name(),
			LastModified: e.ModTime(),
			Size:         e.Size(),
		})
	}
	return out, nil
}

func (w *WebDAVStorage) DeleteFile(ctx context.Context, job *pkgconfig.JobConfig, fileName string, storage pkgconfig.StorageConfig) error {
	client, err := openWebDAV(ctx, storage)
	if err != nil {
		return err
	}
	return client.Remove(path.Join(w.jobDir(storage, job.Name), fileName))
}

func (w *WebDAVStorage) DeleteWrapper(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) error {
	client, err := openWebDAV(ctx, storage)
	if err != nil {
		return err
	}
	dir := path.Join(w.jobDir(storage, job.Name), wrapperName)
	return client.RemoveAll(dir)
}

func (w *WebDAVStorage) Download(ctx context.Context, job *pkgconfig.JobConfig, wrapperRelDir, fileName string, storage pkgconfig.StorageConfig) (*DownloadResult, error) {
	client, err := openWebDAV(ctx, storage)
	if err != nil {
		return nil, err
	}

	dir := w.jobDir(storage, job.Name)
	if wrapperRelDir != "" {
		dir = path.Join(dir, wrapperRelDir)
	}
	remote := path.Join(dir, fileName)

	stat, err := client.Stat(remote)
	if err != nil {
		return nil, fmt.Errorf("webdav stat %s: %w", remote, err)
	}
	stream, err := client.ReadStream(remote)
	if err != nil {
		return nil, fmt.Errorf("webdav read %s: %w", remote, err)
	}
	return &DownloadResult{Body: stream, Size: stat.Size()}, nil
}

func openWebDAV(ctx context.Context, storage pkgconfig.StorageConfig) (*gowebdav.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if storage.URL == "" {
		return nil, fmt.Errorf("webdav: url is required")
	}
	client := gowebdav.NewClient(storage.URL, storage.Username, storage.Password)
	client.SetTimeout(30 * time.Minute)
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("webdav connect %s: %w", storage.URL, err)
	}
	return client, nil
}

func webdavDir(storage pkgconfig.StorageConfig) string {
	if storage.Path == "" {
		return "/"
	}
	if !strings.HasPrefix(storage.Path, "/") {
		return "/" + storage.Path
	}
	return storage.Path
}

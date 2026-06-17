package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/rs/zerolog"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type SFTPStorage struct{}

func NewSFTPStorage() *SFTPStorage { return &SFTPStorage{} }

func (s *SFTPStorage) GetType() string { return "sftp" }

func (s *SFTPStorage) jobDir(storage pkgconfig.StorageConfig, jobName string) string {
	base := storage.Path
	if base == "" {
		base = "."
	}
	return path.Join(base, JobNameSegment(storage.IncludeJobName, jobName))
}

func (s *SFTPStorage) EnsureJobDir(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) error {
	client, sshClient, err := openSFTP(ctx, storage)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer func() { _ = sshClient.Close() }()

	dir := s.jobDir(storage, job.Name)
	if err := client.MkdirAll(dir); err != nil {
		return fmt.Errorf("sftp mkdir %s: %w", dir, err)
	}
	return nil
}

func (s *SFTPStorage) UploadInto(ctx context.Context, archivePath string, job *pkgconfig.JobConfig, wrapperRelDir string, storage pkgconfig.StorageConfig) error {
	logger := zerolog.Ctx(ctx)

	client, sshClient, err := openSFTP(ctx, storage)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer func() { _ = sshClient.Close() }()

	dir := s.jobDir(storage, job.Name)
	if wrapperRelDir != "" {
		dir = path.Join(dir, wrapperRelDir)
	}
	if err := client.MkdirAll(dir); err != nil {
		return fmt.Errorf("sftp mkdir %s: %w", dir, err)
	}

	fileName := filepath.Base(archivePath)
	remoteFile := path.Join(dir, fileName)
	tmpRemote := remoteFile + ".tmp"

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
	dst, err := client.OpenFile(tmpRemote, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("sftp create %s: %w", tmpRemote, err)
	}

	success := false
	defer func() {
		_ = dst.Close()
		if !success {
			_ = client.Remove(tmpRemote)
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("sftp upload: %w", err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("sftp close: %w", err)
	}
	if err := client.PosixRename(tmpRemote, remoteFile); err != nil {
		_ = client.Remove(remoteFile)
		if err := client.Rename(tmpRemote, remoteFile); err != nil {
			return fmt.Errorf("sftp rename: %w", err)
		}
	}
	success = true

	logger.Info().
		Str("remote", remoteFile).
		Int64("size_bytes", stat.Size()).
		Str("size_human", utils.FormatBytes(stat.Size())).
		Dur("duration", time.Since(uploadStart)).
		Msg("Uploaded archive over SFTP")
	return nil
}

func (s *SFTPStorage) ListFiles(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	client, sshClient, err := openSFTP(ctx, storage)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()
	defer func() { _ = sshClient.Close() }()

	dir := s.jobDir(storage, job.Name)
	entries, err := client.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []FileInfo{}, nil
		}
		return nil, fmt.Errorf("sftp readdir %s: %w", dir, err)
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
		if strings.HasSuffix(e.Name(), ".tmp") {
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

func (s *SFTPStorage) ListWrapperParts(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	client, sshClient, err := openSFTP(ctx, storage)
	if err != nil {
		return nil, err
	}
	defer func() { _ = client.Close() }()
	defer func() { _ = sshClient.Close() }()

	dir := path.Join(s.jobDir(storage, job.Name), wrapperName)
	entries, err := client.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []FileInfo{}, nil
		}
		return nil, fmt.Errorf("sftp readdir %s: %w", dir, err)
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

func (s *SFTPStorage) DeleteFile(ctx context.Context, job *pkgconfig.JobConfig, fileName string, storage pkgconfig.StorageConfig) error {
	client, sshClient, err := openSFTP(ctx, storage)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer func() { _ = sshClient.Close() }()

	return client.Remove(path.Join(s.jobDir(storage, job.Name), fileName))
}

func (s *SFTPStorage) DeleteWrapper(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) error {
	parts, err := s.ListWrapperParts(ctx, job, wrapperName, storage)
	if err != nil {
		return err
	}

	client, sshClient, err := openSFTP(ctx, storage)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	defer func() { _ = sshClient.Close() }()

	dir := path.Join(s.jobDir(storage, job.Name), wrapperName)
	for _, p := range parts {
		if err := client.Remove(path.Join(dir, p.Name)); err != nil {
			return fmt.Errorf("remove wrapper part: %w", err)
		}
	}
	if err := client.RemoveDirectory(dir); err != nil {
		return fmt.Errorf("remove wrapper dir %s: %w", dir, err)
	}
	return nil
}

func (s *SFTPStorage) Download(ctx context.Context, job *pkgconfig.JobConfig, wrapperRelDir, fileName string, storage pkgconfig.StorageConfig) (*DownloadResult, error) {
	client, sshClient, err := openSFTP(ctx, storage)
	if err != nil {
		return nil, err
	}

	dir := s.jobDir(storage, job.Name)
	if wrapperRelDir != "" {
		dir = path.Join(dir, wrapperRelDir)
	}
	remote := path.Join(dir, fileName)

	f, err := client.Open(remote)
	if err != nil {
		_ = client.Close()
		_ = sshClient.Close()
		return nil, fmt.Errorf("sftp open %s: %w", remote, err)
	}

	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		_ = client.Close()
		_ = sshClient.Close()
		return nil, fmt.Errorf("sftp stat %s: %w", remote, err)
	}

	body := &sftpReadCloser{file: f, client: client, ssh: sshClient}
	return &DownloadResult{Body: body, Size: stat.Size()}, nil
}

func openSFTP(ctx context.Context, storage pkgconfig.StorageConfig) (*sftp.Client, sshClientCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	sshClient, err := dialSSH(storage)
	if err != nil {
		return nil, nil, err
	}
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, fmt.Errorf("sftp client: %w", err)
	}
	return client, sshClient, nil
}

type sftpReadCloser struct {
	file   *sftp.File
	client *sftp.Client
	ssh    sshClientCloser
}

func (r *sftpReadCloser) Read(p []byte) (int, error) { return r.file.Read(p) }
func (r *sftpReadCloser) Close() error {
	err := r.file.Close()
	_ = r.client.Close()
	_ = r.ssh.Close()
	return err
}

type sshClientCloser interface{ Close() error }

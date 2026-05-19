package storage

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/maximseshuk/snapr/internal/config"
)

var ErrDownloadNotSupported = errors.New("storage does not support download")

type FileInfo struct {
	Name         string
	LastModified time.Time
	Size         int64
	// Wrapper is true for split-snapshot wrapper directories.
	Wrapper bool
}

// DownloadResult: exactly one of Body or RedirectURL is set.
type DownloadResult struct {
	Body        io.ReadCloser
	Size        int64
	RedirectURL string
}

type Storage interface {
	// EnsureJobDir is a no-op on object stores.
	EnsureJobDir(ctx context.Context, job *config.JobConfig, storage config.StorageConfig) error
	// UploadInto writes archivePath to <storage.Path>/<jobName>/<wrapperRelDir>/<basename>;
	// wrapperRelDir is "" for non-split.
	UploadInto(ctx context.Context, archivePath string, job *config.JobConfig, wrapperRelDir string, storage config.StorageConfig) error
	ListFiles(ctx context.Context, job *config.JobConfig, storage config.StorageConfig) ([]FileInfo, error)
	ListWrapperParts(ctx context.Context, job *config.JobConfig, wrapperName string, storage config.StorageConfig) ([]FileInfo, error)
	DeleteFile(ctx context.Context, job *config.JobConfig, fileName string, storage config.StorageConfig) error
	DeleteWrapper(ctx context.Context, job *config.JobConfig, wrapperName string, storage config.StorageConfig) error
	GetType() string
}

// Downloader is optional. Storages without it return ErrDownloadNotSupported.
type Downloader interface {
	Download(ctx context.Context, job *config.JobConfig, wrapperRelDir, fileName string, storage config.StorageConfig) (*DownloadResult, error)
}

type Factory struct {
	storages map[string]Storage
}

func NewFactory() *Factory {
	factory := &Factory{
		storages: make(map[string]Storage),
	}

	factory.Register(NewS3Storage())
	factory.Register(NewLocalStorage())
	factory.Register(NewBunnyStorage())
	factory.Register(NewSFTPStorage())
	factory.Register(NewWebDAVStorage())

	return factory
}

func (f *Factory) Register(storage Storage) {
	f.storages[storage.GetType()] = storage
}

func (f *Factory) Create(storageType string) Storage {
	return f.storages[storageType]
}

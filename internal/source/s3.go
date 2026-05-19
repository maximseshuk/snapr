package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type S3Source struct{}

func NewS3Source() *S3Source {
	return &S3Source{}
}

func (s *S3Source) GetType() string {
	return "s3"
}

type s3DownloadTask struct {
	key       string
	localPath string
	size      int64
	modified  time.Time
}

func (s *S3Source) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	logger.Info().
		Str("bucket", source.Bucket).
		Str("region", source.Region).
		Str("endpoint", source.Endpoint).
		Str("prefix", source.Path).
		Str("destination", destDir).
		Msg("Starting S3 source backup")

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			source.AccessKeyID,
			source.SecretAccessKey,
			"",
		)),
		awsconfig.WithRegion(source.Region),
	)
	if err != nil {
		return fmt.Errorf("error loading AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if source.Endpoint != "" {
			o.BaseEndpoint = aws.String(source.Endpoint)
			o.UsePathStyle = true
		} else if source.UsePathStyle {
			o.UsePathStyle = true
		}
	})

	var destBase string
	useIncremental := false

	if source.SyncPath != "" {
		destBase = source.SyncPath
		useIncremental = true
		if err := os.MkdirAll(destBase, 0755); err != nil {
			return fmt.Errorf("error creating sync directory: %w", err)
		}
		logger.Info().Str("sync_path", destBase).Bool("incremental", true).Msg("Using incremental sync")
	} else {
		destBase = destDir
		if _, err := os.Lstat(destBase); err == nil {
			if err := os.RemoveAll(destBase); err != nil {
				return fmt.Errorf("error removing existing destination: %w", err)
			}
		}
		if err := os.MkdirAll(destBase, 0755); err != nil {
			return fmt.Errorf("error creating destination directory: %w", err)
		}
	}

	prefix := strings.TrimPrefix(source.Path, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	tasks, remoteKeys, err := s.collectTasks(ctx, client, source.Bucket, prefix, destBase, source.Excludes, useIncremental)
	if err != nil {
		return fmt.Errorf("error listing S3 objects: %w", err)
	}

	if useIncremental {
		if err := s.cleanupDeleted(ctx, destBase, remoteKeys); err != nil {
			logger.Warn().Err(err).Msg("Error during cleanup of deleted files")
		}
	}

	if len(tasks) == 0 {
		logger.Info().Msg("No files to download")
		s.finalizeSyncSymlink(ctx, useIncremental, destBase, destDir)
		return nil
	}

	logger.Info().Int("files_to_download", len(tasks)).Msg("Starting downloads")

	tm := transfermanager.New(client)
	workers := utils.WorkerCount(source.ExtraParams, 10, 50)

	if err := s.downloadAll(ctx, tm, source.Bucket, tasks, workers); err != nil {
		return err
	}

	s.finalizeSyncSymlink(ctx, useIncremental, destBase, destDir)
	return nil
}

func (s *S3Source) finalizeSyncSymlink(ctx context.Context, useIncremental bool, syncPath, destDir string) {
	if !useIncremental || syncPath == destDir {
		return
	}
	logger := zerolog.Ctx(ctx)
	if _, err := os.Lstat(destDir); err == nil {
		if err := os.RemoveAll(destDir); err != nil {
			logger.Warn().Err(err).Str("path", destDir).Msg("Error removing existing destination")
			return
		}
	}
	if err := os.Symlink(syncPath, destDir); err != nil {
		logger.Warn().Err(err).Str("source", syncPath).Str("target", destDir).Msg("Error creating symlink")
	}
}

func (s *S3Source) collectTasks(ctx context.Context, client *s3.Client, bucket, prefix, destBase string, excludes []string, useIncremental bool) ([]s3DownloadTask, map[string]bool, error) {
	var tasks []s3DownloadTask
	remoteKeys := make(map[string]bool)

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			key := *obj.Key
			if strings.HasSuffix(key, "/") {
				continue
			}

			relPath := strings.TrimPrefix(key, prefix)
			if relPath == "" {
				continue
			}
			if utils.IsExcluded(relPath, excludes) {
				continue
			}

			localPath, err := utils.SafeJoin(destBase, relPath)
			if err != nil {
				return nil, nil, fmt.Errorf("unsafe object key %q: %w", key, err)
			}

			remoteKeys[filepath.ToSlash(relPath)] = true

			var size int64
			if obj.Size != nil {
				size = *obj.Size
			}
			var modified time.Time
			if obj.LastModified != nil {
				modified = *obj.LastModified
			}

			if useIncremental && utils.IsLocalUpToDate(localPath, size, modified) {
				continue
			}

			if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				return nil, nil, fmt.Errorf("error creating directory %s: %w", filepath.Dir(localPath), err)
			}

			tasks = append(tasks, s3DownloadTask{
				key:       key,
				localPath: localPath,
				size:      size,
				modified:  modified,
			})
		}
	}

	return tasks, remoteKeys, nil
}

func (s *S3Source) downloadAll(ctx context.Context, tm *transfermanager.Client, bucket string, tasks []s3DownloadTask, workers int) error {
	logger := zerolog.Ctx(ctx)

	taskChan := make(chan s3DownloadTask, len(tasks))
	for _, t := range tasks {
		taskChan <- t
	}
	close(taskChan)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	var successCount, failCount int

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if err := s.downloadOne(ctx, tm, bucket, task); err != nil {
					mu.Lock()
					failCount++
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					logger.Error().Err(err).Str("key", task.key).Msg("Failed to download object")
					continue
				}
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	logger.Info().
		Int("successful_downloads", successCount).
		Int("failed_downloads", failCount).
		Msg("S3 source download completed")

	if failCount > 0 {
		return fmt.Errorf("failed to download %d objects: %w", failCount, firstErr)
	}
	return nil
}

func (s *S3Source) downloadOne(ctx context.Context, tm *transfermanager.Client, bucket string, task s3DownloadTask) error {
	tempPath := task.localPath + ".tmp"
	f, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}

	out, err := tm.GetObject(ctx, &transfermanager.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(task.key),
	})
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("error getting object %s: %w", task.key, err)
	}

	_, copyErr := io.Copy(f, out.Body)
	_ = f.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("error writing object %s: %w", task.key, copyErr)
	}

	if err := os.Rename(tempPath, task.localPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("error renaming file: %w", err)
	}

	if !task.modified.IsZero() {
		_ = os.Chtimes(task.localPath, task.modified, task.modified)
	}
	return nil
}

func (s *S3Source) cleanupDeleted(ctx context.Context, destBase string, remoteKeys map[string]bool) error {
	logger := zerolog.Ctx(ctx)

	deleted := 0
	err := filepath.Walk(destBase, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil //nolint:nilerr
		}
		rel, relErr := filepath.Rel(destBase, path)
		if relErr != nil {
			return nil //nolint:nilerr
		}
		rel = filepath.ToSlash(rel)
		if !remoteKeys[rel] {
			if rerr := os.Remove(path); rerr == nil { //nolint:gosec // cleanup of own walked tree
				deleted++
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	if deleted > 0 {
		logger.Info().Int("deleted_files", deleted).Msg("Cleaned up deleted/excluded files")
	}

	utils.RemoveEmptyDirs(destBase)
	return nil
}

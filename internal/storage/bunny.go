package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
	bunnyclient "github.com/maximseshuk/snapr/internal/providers/bunny"
	"github.com/maximseshuk/snapr/internal/utils"
)

const (
	bunnyMaxAttempts = 3
	bunnyBaseDelay   = 500 * time.Millisecond
)

type BunnyStorage struct {
	httpClient *http.Client
}

func NewBunnyStorage() *BunnyStorage {
	return &BunnyStorage{
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        20,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (b *BunnyStorage) GetType() string {
	return "bunny"
}

func (b *BunnyStorage) EnsureJobDir(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) error {
	_ = ctx
	_ = job
	_ = storage
	return nil
}

func (b *BunnyStorage) UploadInto(ctx context.Context, archivePath string, job *pkgconfig.JobConfig, wrapperRelDir string, storage pkgconfig.StorageConfig) error {
	logger := zerolog.Ctx(ctx)

	fileName := filepath.Base(archivePath)
	uploadURL := b.objectURL(storage, job.Name, wrapperRelDir, fileName)

	file, err := os.Open(archivePath) //nolint:gosec // archivePath comes from snapr's own pipeline
	if err != nil {
		return fmt.Errorf("error opening archive file: %w", err)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}

	uploadStart := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, file)
	if err != nil {
		return fmt.Errorf("error creating upload request: %w", err)
	}
	req.Header.Set("AccessKey", storage.AccessKey)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = stat.Size()

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error uploading to Bunny: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("bunny rate limited (429): %s", string(body))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("bunny upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	logger.Info().
		Str("file_name", fileName).
		Str("zone_name", storage.ZoneName).
		Int64("file_size_bytes", stat.Size()).
		Str("file_size_human", utils.FormatBytes(stat.Size())).
		Dur("upload_duration", time.Since(uploadStart)).
		Msg("Successfully uploaded archive to Bunny Storage")
	return nil
}

func (b *BunnyStorage) ListFiles(ctx context.Context, job *pkgconfig.JobConfig, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	items, err := b.listEntries(ctx, b.listURL(storage, job.Name, ""), storage.AccessKey)
	if err != nil {
		return nil, err
	}

	out := make([]FileInfo, 0, len(items))
	for _, item := range items {
		modified, err := bunnyclient.ParseTimestamp(item.LastChanged)
		if err != nil {
			modified = time.Now()
		}
		if item.IsDirectory {
			if _, _, _, ok := ParseSplitWrapper(item.ObjectName); !ok {
				continue
			}
			out = append(out, FileInfo{
				Name:         item.ObjectName,
				LastModified: modified,
				Wrapper:      true,
			})
			continue
		}
		out = append(out, FileInfo{
			Name:         item.ObjectName,
			LastModified: modified,
			Size:         item.Length,
		})
	}

	zerolog.Ctx(ctx).Debug().
		Int("entries", len(out)).
		Str("zone_name", storage.ZoneName).
		Msg("Listed Bunny entries")
	return out, nil
}

func (b *BunnyStorage) ListWrapperParts(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) ([]FileInfo, error) {
	items, err := b.listEntries(ctx, b.listURL(storage, job.Name, wrapperName), storage.AccessKey)
	if err != nil {
		return nil, err
	}

	out := make([]FileInfo, 0, len(items))
	for _, item := range items {
		if item.IsDirectory {
			continue
		}
		modified, err := bunnyclient.ParseTimestamp(item.LastChanged)
		if err != nil {
			modified = time.Now()
		}
		out = append(out, FileInfo{
			Name:         item.ObjectName,
			LastModified: modified,
			Size:         item.Length,
		})
	}
	return out, nil
}

func (b *BunnyStorage) DeleteFile(ctx context.Context, job *pkgconfig.JobConfig, fileName string, storage pkgconfig.StorageConfig) error {
	return b.deleteURL(ctx, b.objectURL(storage, job.Name, "", fileName), storage.AccessKey, fileName)
}

func (b *BunnyStorage) DeleteWrapper(ctx context.Context, job *pkgconfig.JobConfig, wrapperName string, storage pkgconfig.StorageConfig) error {
	parts, err := b.ListWrapperParts(ctx, job, wrapperName, storage)
	if err != nil {
		return err
	}
	for _, p := range parts {
		urlStr := b.objectURL(storage, job.Name, wrapperName, p.Name)
		if err := b.deleteURL(ctx, urlStr, storage.AccessKey, p.Name); err != nil {
			return fmt.Errorf("delete wrapper part %s: %w", p.Name, err)
		}
	}
	return nil
}

func (b *BunnyStorage) Download(ctx context.Context, job *pkgconfig.JobConfig, wrapperRelDir, fileName string, storage pkgconfig.StorageConfig) (*DownloadResult, error) {
	logger := zerolog.Ctx(ctx)

	if storage.PullZoneHostname != "" && storage.PullZoneTokenAuthKey != "" {
		signedURL := b.signPullZoneURL(storage, job.Name, wrapperRelDir, fileName)
		logger.Debug().
			Str("file_name", fileName).
			Str("hostname", storage.PullZoneHostname).
			Msg("Generated signed Bunny pull zone URL")
		return &DownloadResult{RedirectURL: signedURL}, nil
	}

	downloadURL := b.objectURL(storage, job.Name, wrapperRelDir, fileName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating download request: %w", err)
	}
	req.Header.Set("AccessKey", storage.AccessKey)

	resp, err := bunnyDoRetry(ctx, b.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("error downloading from Bunny: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("bunny download failed with status %d", resp.StatusCode)
	}

	logger.Debug().
		Str("file_name", fileName).
		Str("zone_name", storage.ZoneName).
		Int64("content_length", resp.ContentLength).
		Msg("Streaming download from Bunny")
	return &DownloadResult{Body: resp.Body, Size: resp.ContentLength}, nil
}

func (b *BunnyStorage) baseURL(storage pkgconfig.StorageConfig) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(storage.Endpoint, "/"), storage.ZoneName)
}

func (b *BunnyStorage) jobBaseURL(storage pkgconfig.StorageConfig, jobName string) string {
	parts := []string{b.baseURL(storage)}
	if storage.Path != "" {
		parts = append(parts, strings.Trim(storage.Path, "/"))
	}
	parts = append(parts, jobName)
	return strings.Join(parts, "/")
}

func (b *BunnyStorage) objectURL(storage pkgconfig.StorageConfig, jobName, wrapperRelDir, fileName string) string {
	base := b.jobBaseURL(storage, jobName)
	if wrapperRelDir != "" {
		base = base + "/" + wrapperRelDir
	}
	return base + "/" + url.PathEscape(fileName)
}

func (b *BunnyStorage) listURL(storage pkgconfig.StorageConfig, jobName, wrapperRelDir string) string {
	base := b.jobBaseURL(storage, jobName)
	if wrapperRelDir != "" {
		base = base + "/" + wrapperRelDir
	}
	return base + "/"
}

func (b *BunnyStorage) signPullZoneURL(storage pkgconfig.StorageConfig, jobName, wrapperRelDir, fileName string) string {
	p := "/"
	if storage.Path != "" {
		p += strings.Trim(storage.Path, "/") + "/"
	}
	p += jobName + "/"
	if wrapperRelDir != "" {
		p += wrapperRelDir + "/"
	}
	p += fileName

	return bunnyclient.SignURL(bunnyclient.SignOptions{
		Hostname:    storage.PullZoneHostname,
		SecurityKey: storage.PullZoneTokenAuthKey,
		Path:        p,
		TTL:         storage.PullZoneTokenTTL,
	})
}

func (b *BunnyStorage) listEntries(ctx context.Context, listURL, accessKey string) ([]bunnyclient.ListItem, error) {
	listCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(listCtx, http.MethodGet, listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating list request: %w", err)
	}
	req.Header.Set("AccessKey", accessKey)

	resp, err := bunnyDoRetry(listCtx, b.httpClient, req)
	if err != nil {
		return nil, fmt.Errorf("error listing Bunny files: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bunny list failed with status %d", resp.StatusCode)
	}

	var items []bunnyclient.ListItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("error parsing list response: %w", err)
	}
	return items, nil
}

func (b *BunnyStorage) deleteURL(ctx context.Context, urlStr, accessKey, label string) error {
	delCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(delCtx, http.MethodDelete, urlStr, nil)
	if err != nil {
		return fmt.Errorf("error creating delete request: %w", err)
	}
	req.Header.Set("AccessKey", accessKey)

	resp, err := bunnyDoRetry(delCtx, b.httpClient, req)
	if err != nil {
		return fmt.Errorf("error deleting Bunny object: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound && (resp.StatusCode < 200 || resp.StatusCode >= 300) {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("bunny delete %s failed with status %d: %s", label, resp.StatusCode, string(body))
	}
	zerolog.Ctx(ctx).Debug().Str("file_name", label).Msg("Deleted Bunny object")
	return nil
}

func bunnyDoRetry(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	var lastResp *http.Response
	for attempt := 0; attempt < bunnyMaxAttempts; attempt++ {
		if attempt > 0 {
			delay := bunnyBaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		_ = resp.Body.Close()
		lastResp = resp
	}
	if lastResp != nil {
		return nil, fmt.Errorf("bunny: rate limited after %d attempts", bunnyMaxAttempts)
	}
	return nil, fmt.Errorf("bunny: exhausted retries")
}

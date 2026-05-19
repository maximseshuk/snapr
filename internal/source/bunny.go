package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
	bunnyclient "github.com/maximseshuk/snapr/internal/providers/bunny"
	"github.com/maximseshuk/snapr/internal/utils"
)

type BunnySource struct{}

type scanTask struct {
	path string
}

type scanResult struct {
	path  string
	files []bunnyclient.ListItem
	err   error
}

type downloadTask struct {
	file      bunnyclient.ListItem
	localPath string
	retries   int
}

type downloadResult struct {
	task downloadTask
	err  error
}

const (
	maxConcurrentConnections = 30
	maxScanWorkers           = 10
	maxRetries               = 5
	baseRetryDelay           = time.Second
	maxRetryDelay            = 30 * time.Second
)

func NewBunnySource() *BunnySource {
	return &BunnySource{}
}

func (b *BunnySource) GetType() string {
	return "bunny"
}

func (b *BunnySource) Backup(ctx context.Context, destDir string, source config.SourceConfig) error {
	logger := zerolog.Ctx(ctx)

	logger.Info().
		Str("zone_name", source.ZoneName).
		Str("endpoint", source.Endpoint).
		Str("destination", destDir).
		Msg("Starting Bunny Storage backup")

	if source.Endpoint == "" {
		return fmt.Errorf("endpoint is required for Bunny Storage")
	}
	if source.ZoneName == "" {
		return fmt.Errorf("zone_name is required for Bunny Storage")
	}
	if source.AccessKey == "" {
		return fmt.Errorf("access_key is required for Bunny Storage")
	}

	var bunnyDestDir string
	var useIncrementalSync bool

	if source.SyncPath != "" {
		bunnyDestDir = source.SyncPath
		useIncrementalSync = true

		if err := os.MkdirAll(bunnyDestDir, 0755); err != nil {
			logger.Error().Err(err).Str("sync_path", bunnyDestDir).Msg("Error creating sync directory")
			return fmt.Errorf("error creating sync directory: %w", err)
		}

		logger.Info().
			Str("sync_path", bunnyDestDir).
			Bool("incremental", true).
			Msg("Using incremental sync")
	} else {
		bunnyDestDir = destDir
		useIncrementalSync = false

		logger.Info().
			Str("destination", bunnyDestDir).
			Bool("incremental", false).
			Msg("Using direct download")
	}

	workers := utils.WorkerCount(source.ExtraParams, 10, maxConcurrentConnections)

	if err := b.syncFilesParallel(ctx, source, bunnyDestDir, useIncrementalSync, workers); err != nil {
		return err
	}

	if useIncrementalSync && bunnyDestDir != destDir {
		if _, err := os.Lstat(destDir); err == nil {
			if err := os.RemoveAll(destDir); err != nil {
				logger.Error().Err(err).Str("path", destDir).Msg("Error removing existing destination")
				return fmt.Errorf("error removing existing destination: %w", err)
			}
		}

		if err := os.Symlink(bunnyDestDir, destDir); err != nil {
			logger.Error().Err(err).
				Str("source", bunnyDestDir).
				Str("target", destDir).
				Msg("Error creating symlink")
			return fmt.Errorf("error creating symlink: %w", err)
		}

		logger.Info().
			Str("source", bunnyDestDir).
			Str("target", destDir).
			Msg("Created symlink to sync path")
	}

	return nil
}

func (b *BunnySource) buildDownloadURL(source config.SourceConfig, file bunnyclient.ListItem) (string, bool) {
	if source.PullZoneHostname != "" && source.PullZoneTokenAuthKey != "" {
		zonePrefix := "/" + source.ZoneName
		fullPath := file.Path + file.ObjectName
		urlPath := strings.TrimPrefix(fullPath, zonePrefix)
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath
		}

		signed := bunnyclient.SignURL(bunnyclient.SignOptions{
			Hostname:    source.PullZoneHostname,
			SecurityKey: source.PullZoneTokenAuthKey,
			Path:        urlPath,
			TTL:         source.PullZoneTokenTTL,
		})
		return signed, false
	}

	return strings.TrimSuffix(source.Endpoint, "/") + file.Path + file.ObjectName, true
}

func (b *BunnySource) buildStorageURL(source config.SourceConfig, path string) string {
	endpoint := strings.TrimSuffix(source.Endpoint, "/")
	baseURL := fmt.Sprintf("%s/%s", endpoint, source.ZoneName)

	if path != "" && path != "/" {
		path = strings.Trim(path, "/")
		baseURL = fmt.Sprintf("%s/%s", baseURL, path)
	}

	return baseURL
}

func (b *BunnySource) syncFilesParallel(ctx context.Context, source config.SourceConfig, destDir string, useIncremental bool, workers int) error {
	logger := zerolog.Ctx(ctx)

	logger.Info().
		Bool("incremental", useIncremental).
		Int("workers", workers).
		Msg("Starting file synchronization")

	if useIncremental {
		if err := b.syncLocalDirectory(ctx, source, destDir); err != nil {
			logger.Warn().Err(err).Msg("Error during cleanup of deleted files")
		}
	}

	allTasks, err := b.collectDownloadTasks(ctx, source, destDir, source.Path, useIncremental)
	if err != nil {
		return fmt.Errorf("error collecting download tasks: %w", err)
	}

	if len(allTasks) == 0 {
		logger.Info().Msg("No files to download")
		return nil
	}

	logger.Info().Int("files_to_download", len(allTasks)).Msg("Starting downloads")

	taskChan := make(chan downloadTask, len(allTasks))
	resultChan := make(chan downloadResult, len(allTasks))

	for _, task := range allTasks {
		taskChan <- task
	}
	close(taskChan)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go b.downloadWorker(ctx, &wg, source, taskChan, resultChan)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var failedTasks []downloadTask
	successCount := 0

	for result := range resultChan {
		if result.err != nil {
			if result.task.retries < maxRetries {
				result.task.retries++
				failedTasks = append(failedTasks, result.task)
			} else {
				logger.Error().
					Str("file", result.task.file.ObjectName).
					Int("max_retries", maxRetries).
					Msg("Max retries exceeded for file")
			}
		} else {
			successCount++
		}
	}

	if len(failedTasks) > 0 {
		logger.Info().Int("failed_count", len(failedTasks)).Msg("Retrying failed downloads")
		if err := b.retryFailedDownloads(ctx, source, failedTasks, workers); err != nil {
			logger.Warn().Err(err).Msg("Some retries failed")
		}
	}

	logger.Info().
		Int("successful_downloads", successCount).
		Int("failed_downloads", len(allTasks)-successCount).
		Msg("File synchronization completed")

	return nil
}

func (b *BunnySource) syncLocalDirectory(ctx context.Context, source config.SourceConfig, destDir string) error {
	logger := zerolog.Ctx(ctx)

	remoteFiles, err := b.getAllRemoteFilesParallel(ctx, source, source.Path)
	if err != nil {
		return fmt.Errorf("scan remote tree: %w", err)
	}

	shouldExistLocally := make(map[string]bool)
	for _, file := range remoteFiles {
		if !file.IsDirectory {
			relativePath := b.getRelativePath(file, source)
			if !utils.IsExcluded(relativePath, source.Excludes) {
				shouldExistLocally[relativePath] = true
			}
		}
	}

	deletedCount := 0
	walkErr := filepath.Walk(destDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relativePath, relErr := filepath.Rel(destDir, localPath)
		if relErr != nil {
			logger.Warn().Err(relErr).Str("path", localPath).Msg("Skip file: cannot compute relative path")
			return nil
		}

		relativePath = filepath.ToSlash(relativePath)

		if !shouldExistLocally[relativePath] {
			if err := os.Remove(localPath); err != nil { //nolint:gosec // cleanup of own walked tree
				logger.Warn().Err(err).Str("path", localPath).Msg("Could not delete local file")
			} else {
				deletedCount++
			}
		}

		return nil
	})

	if walkErr != nil {
		return fmt.Errorf("error walking local directory: %w", walkErr)
	}

	utils.RemoveEmptyDirs(destDir)

	if deletedCount > 0 {
		logger.Info().Int("deleted_files", deletedCount).Msg("Cleaned up deleted/excluded files")
	}

	return nil
}

func (b *BunnySource) getRelativePath(file bunnyclient.ListItem, source config.SourceConfig) string {
	fullPath := file.Path + file.ObjectName

	zonePath := "/" + source.ZoneName + "/"
	relativePath := strings.TrimPrefix(fullPath, zonePath)

	if source.Path != "" && source.Path != "/" {
		configPath := strings.Trim(source.Path, "/")
		if configPath != "" {
			configPath += "/"
			relativePath = strings.TrimPrefix(relativePath, configPath)
		}
	}

	return relativePath
}

// getAllRemoteFilesParallel walks the remote tree via a bounded scan channel.
// Aborts with an error on channel overflow rather than silently skipping branches.
func (b *BunnySource) getAllRemoteFilesParallel(ctx context.Context, source config.SourceConfig, rootPath string) ([]bunnyclient.ListItem, error) {
	logger := zerolog.Ctx(ctx)

	var allFiles []bunnyclient.ListItem
	var filesMutex sync.Mutex

	scanChan := make(chan scanTask, 1000)
	resultChan := make(chan scanResult, 1000)

	scanChan <- scanTask{path: rootPath}

	var wg sync.WaitGroup
	for range maxScanWorkers {
		wg.Add(1)
		go b.scanWorker(ctx, &wg, source, scanChan, resultChan)
	}

	var coordinatorErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		pendingScans := 1
		processedScans := 0
		closed := false
		closeScan := func() {
			if !closed {
				close(scanChan)
				closed = true
			}
		}

		for result := range resultChan {
			processedScans++

			if result.err != nil {
				logger.Warn().Err(result.err).Str("path", result.path).Msg("Error scanning directory")
			} else {
				filesMutex.Lock()
				allFiles = append(allFiles, result.files...)
				filesMutex.Unlock()

				newDirs := 0
				for _, file := range result.files {
					if !file.IsDirectory {
						continue
					}
					subPath := filepath.Join(result.path, file.ObjectName)
					select {
					case scanChan <- scanTask{path: subPath}:
						newDirs++
					case <-ctx.Done():
						coordinatorErr = ctx.Err()
						closeScan()
						return
					default:
						coordinatorErr = fmt.Errorf("scan queue overflow at %q; raise scan buffer or reduce parallelism", subPath)
						closeScan()
						return
					}
				}
				pendingScans += newDirs
			}

			if processedScans >= pendingScans {
				closeScan()
				return
			}
		}
	}()

	wg.Wait()
	close(resultChan)
	<-done

	if coordinatorErr != nil {
		return nil, coordinatorErr
	}

	logger.Info().Int("total_items", len(allFiles)).Msg("Completed remote directory scanning")
	return allFiles, nil
}

func (b *BunnySource) scanWorker(ctx context.Context, wg *sync.WaitGroup, source config.SourceConfig, scanChan <-chan scanTask, resultChan chan<- scanResult) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 2 * time.Minute,
	}

	for task := range scanChan {
		select {
		case <-ctx.Done():
			resultChan <- scanResult{path: task.path, err: ctx.Err()}
			return
		default:
		}

		files, err := b.listFilesWithClient(ctx, client, source, task.path)
		resultChan <- scanResult{
			path:  task.path,
			files: files,
			err:   err,
		}
	}
}

func (b *BunnySource) listFiles(ctx context.Context, source config.SourceConfig, path string) ([]bunnyclient.ListItem, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	return b.listFilesWithClient(ctx, client, source, path)
}

func (b *BunnySource) listFilesWithClient(ctx context.Context, client *http.Client, source config.SourceConfig, path string) ([]bunnyclient.ListItem, error) {
	url := b.buildStorageURL(source, path) + "/"

	var files []bunnyclient.ListItem
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			if attempt == 2 {
				return nil, fmt.Errorf("error creating list request: %w", err)
			}
			continue
		}

		req.Header.Set("AccessKey", source.AccessKey)

		resp, err := client.Do(req)
		if err != nil {
			if attempt == 2 {
				return nil, fmt.Errorf("error listing files after retries: %w", err)
			}
			continue
		}

		if resp.StatusCode == 429 {
			_ = resp.Body.Close()
			if attempt == 2 {
				return nil, fmt.Errorf("rate limited while listing files")
			}
			time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			if attempt == 2 {
				return nil, fmt.Errorf("list request failed with status: %d", resp.StatusCode)
			}
			continue
		}

		err = json.NewDecoder(resp.Body).Decode(&files)
		_ = resp.Body.Close()

		if err != nil {
			if attempt == 2 {
				return nil, fmt.Errorf("error parsing file list: %w", err)
			}
			continue
		}

		break
	}

	return files, nil
}

func (b *BunnySource) collectDownloadTasks(ctx context.Context, source config.SourceConfig, destDir, currentPath string, useIncremental bool) ([]downloadTask, error) {
	var tasks []downloadTask

	files, err := b.listFiles(ctx, source, currentPath)
	if err != nil {
		return nil, fmt.Errorf("error listing files in %s: %w", currentPath, err)
	}

	for _, file := range files {
		if file.IsDirectory {
			subPath := filepath.Join(currentPath, file.ObjectName)
			subTasks, err := b.collectDownloadTasks(ctx, source, destDir, subPath, useIncremental)
			if err != nil {
				zerolog.Ctx(ctx).Warn().Err(err).Str("directory", subPath).Msg("Error collecting tasks from directory")
				continue
			}
			tasks = append(tasks, subTasks...)
		} else {
			if task, shouldDownload := b.createDownloadTask(file, source, destDir, useIncremental); shouldDownload {
				tasks = append(tasks, task)
			}
		}
	}

	return tasks, nil
}

func (b *BunnySource) createDownloadTask(file bunnyclient.ListItem, source config.SourceConfig, destDir string, useIncremental bool) (downloadTask, bool) {
	relativePath := b.getRelativePath(file, source)

	if utils.IsExcluded(relativePath, source.Excludes) {
		return downloadTask{}, false
	}

	localPath, err := utils.SafeJoin(destDir, relativePath)
	if err != nil {
		return downloadTask{}, false
	}

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return downloadTask{}, false
	}

	if useIncremental && b.isFileUpToDate(localPath, file) {
		return downloadTask{}, false
	}

	return downloadTask{
		file:      file,
		localPath: localPath,
		retries:   0,
	}, true
}

func (b *BunnySource) downloadWorker(ctx context.Context, wg *sync.WaitGroup, source config.SourceConfig, taskChan <-chan downloadTask, resultChan chan<- downloadResult) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	for task := range taskChan {
		select {
		case <-ctx.Done():
			resultChan <- downloadResult{task: task, err: ctx.Err()}
			return
		default:
		}

		err := b.downloadFileWithRetry(ctx, client, source, task)
		resultChan <- downloadResult{task: task, err: err}
	}
}

func (b *BunnySource) downloadFileWithRetry(ctx context.Context, client *http.Client, source config.SourceConfig, task downloadTask) error {
	logger := zerolog.Ctx(ctx)
	var lastErr error

	for attempt := 0; attempt <= task.retries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Min(float64(baseRetryDelay)*math.Pow(2, float64(attempt-1)), float64(maxRetryDelay)))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := b.downloadSingleFile(ctx, client, source, task); err != nil {
			lastErr = err

			if b.shouldRetry(err) {
				continue
			}
			break
		}

		if attempt > 0 {
			logger.Info().
				Str("file", task.file.ObjectName).
				Int("attempts", attempt+1).
				Msg("Successfully downloaded file after retries")
		}
		return nil
	}

	return fmt.Errorf("failed to download %s after %d attempts: %w", task.file.ObjectName, maxRetries+1, lastErr)
}

func (b *BunnySource) downloadSingleFile(ctx context.Context, client *http.Client, source config.SourceConfig, task downloadTask) error {
	downloadURL, useAccessKey := b.buildDownloadURL(source, task.file)

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("error creating download request: %w", err)
	}

	if useAccessKey {
		req.Header.Set("AccessKey", source.AccessKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error downloading file: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limited (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	tempPath := task.localPath + ".tmp"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer func() {
		_ = tempFile.Close()
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("error closing temp file: %w", err)
	}

	if err := os.Rename(tempPath, task.localPath); err != nil {
		return fmt.Errorf("error moving temp file: %w", err)
	}

	_ = os.Chtimes(task.localPath, b.parseTime(task.file.LastChanged), b.parseTime(task.file.LastChanged))

	return nil
}

func (b *BunnySource) shouldRetry(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504")
}

func (b *BunnySource) retryFailedDownloads(ctx context.Context, source config.SourceConfig, failedTasks []downloadTask, workers int) error {
	if len(failedTasks) == 0 {
		return nil
	}

	retryWorkers := workers / 2
	if retryWorkers < 1 {
		retryWorkers = 1
	}

	taskChan := make(chan downloadTask, len(failedTasks))
	resultChan := make(chan downloadResult, len(failedTasks))

	for _, task := range failedTasks {
		taskChan <- task
	}
	close(taskChan)

	var wg sync.WaitGroup
	for range retryWorkers {
		wg.Add(1)
		go b.downloadWorker(ctx, &wg, source, taskChan, resultChan)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var stillFailed int
	for result := range resultChan {
		if result.err != nil {
			stillFailed++
		}
	}

	if stillFailed > 0 {
		return fmt.Errorf("%d files still failed after retry", stillFailed)
	}

	return nil
}

func (b *BunnySource) isFileUpToDate(localPath string, remoteFile bunnyclient.ListItem) bool {
	// Downloads set local mtime to remote LastChanged, so older local = stale.
	remoteModified, err := bunnyclient.ParseTimestamp(remoteFile.LastChanged)
	if err != nil {
		return false
	}
	return utils.IsLocalUpToDate(localPath, remoteFile.Length, remoteModified)
}

func (b *BunnySource) parseTime(timeStr string) time.Time {
	parsed, err := bunnyclient.ParseTimestamp(timeStr)
	if err != nil {
		return time.Now()
	}
	return parsed
}

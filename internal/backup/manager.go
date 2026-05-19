package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/compression"
	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/encryptor"
	"github.com/maximseshuk/snapr/internal/logger"
	"github.com/maximseshuk/snapr/internal/metrics"
	"github.com/maximseshuk/snapr/internal/notifier"
	"github.com/maximseshuk/snapr/internal/source"
	"github.com/maximseshuk/snapr/internal/storage"
	"github.com/maximseshuk/snapr/internal/utils"
)

var (
	ErrJobAlreadyRunning = errors.New("job already running")
	ErrJobNotFound       = errors.New("job not found")
)

type jobExecution struct {
	Job         *config.JobConfig
	Ctx         context.Context
	Cancel      context.CancelFunc
	TmpDir      string
	SourcesDir  string
	SourcePaths map[int]string
	StartTime   time.Time
	Timestamp   string
	ArchivePath string
	logger      zerolog.Logger
}

type Manager struct {
	cfg                *config.Config
	activeJobs         map[string]*jobExecution
	activeJobsMu       sync.Mutex
	jobLogs            map[string]*JobLog
	jobLogsMu          sync.RWMutex
	tmpDir             string
	sourceFactory      *source.Factory
	storageFactory     *storage.Factory
	compressionFactory *compression.Factory
	encryptorFactory   *encryptor.Factory
	systemLogger       zerolog.Logger
	jobsWg             sync.WaitGroup
}

func NewManager(cfg *config.Config) *Manager {
	tmpDir := filepath.Join(os.TempDir(), "snapr")
	systemLogger := logger.NewSystemLogger("manager")

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		systemLogger.Error().Err(err).Str("path", tmpDir).Msg("Cannot create temp directory")
	}

	return &Manager{
		cfg:                cfg,
		activeJobs:         make(map[string]*jobExecution),
		jobLogs:            make(map[string]*JobLog),
		tmpDir:             tmpDir,
		sourceFactory:      source.NewFactory(),
		storageFactory:     storage.NewFactory(),
		compressionFactory: compression.NewFactory(),
		encryptorFactory:   encryptor.NewFactory(),
		systemLogger:       systemLogger,
	}
}

func (bm *Manager) Config() *config.Config {
	return bm.cfg
}

func (bm *Manager) Shutdown(ctx context.Context) {
	bm.systemLogger.Info().Msg("Shutting down")

	bm.activeJobsMu.Lock()
	activeCount := len(bm.activeJobs)
	for _, jobExec := range bm.activeJobs {
		jobExec.Cancel()
	}
	bm.activeJobsMu.Unlock()

	done := make(chan struct{})
	go func() {
		bm.jobsWg.Wait()
		close(done)
	}()

	if activeCount > 0 {
		bm.systemLogger.Info().Int("jobs", activeCount).Msg("Waiting for active jobs")
	}

	select {
	case <-done:
	case <-ctx.Done():
		bm.systemLogger.Warn().Err(ctx.Err()).Msg("Shutdown timeout, jobs may still run")
		// Workers may still be writing — don't touch tmpDir.
		return
	}

	if err := os.RemoveAll(bm.tmpDir); err != nil {
		bm.systemLogger.Warn().Err(err).Str("path", bm.tmpDir).Msg("Cannot remove temp dir")
	}
}

func (bm *Manager) ListJobs() []*config.JobConfig {
	cfg := bm.cfg
	jobs := make([]*config.JobConfig, len(cfg.Jobs))
	for i := range cfg.Jobs {
		jobs[i] = &cfg.Jobs[i]
	}
	return jobs
}

func (bm *Manager) ListJobNames() []string {
	cfg := bm.cfg
	names := make([]string, len(cfg.Jobs))
	for i, job := range cfg.Jobs {
		names[i] = job.Name
	}
	return names
}

func (bm *Manager) GetJobConfig(jobName string) (*config.JobConfig, error) {
	if job := findJob(bm.cfg, jobName); job != nil {
		return job, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
}

func (bm *Manager) IsJobActive(jobName string) bool {
	bm.activeJobsMu.Lock()
	defer bm.activeJobsMu.Unlock()
	_, active := bm.activeJobs[jobName]
	return active
}

// TryStartJob reserves a run slot. Caller MUST call RunReservedJob or the slot leaks.
func (bm *Manager) TryStartJob(jobName string) bool {
	if findJob(bm.cfg, jobName) == nil {
		return false
	}
	return bm.reserveJobSlot(jobName)
}

func (bm *Manager) RunJob(jobName string) error {
	if !bm.TryStartJob(jobName) {
		if bm.IsJobActive(jobName) {
			return ErrJobAlreadyRunning
		}
		return fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
	}
	return bm.RunReservedJob(jobName)
}

func (bm *Manager) RunReservedJob(jobName string) error {
	bm.activeJobsMu.Lock()
	jobExec, exists := bm.activeJobs[jobName]
	bm.activeJobsMu.Unlock()
	if !exists || jobExec == nil {
		return fmt.Errorf("%w: %s (no reservation)", ErrJobNotFound, jobName)
	}

	releaseReservation := func() {
		bm.activeJobsMu.Lock()
		delete(bm.activeJobs, jobName)
		bm.activeJobsMu.Unlock()
		jobExec.Cancel()
	}

	job := findJob(bm.cfg, jobName)
	if job == nil {
		releaseReservation()
		return fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
	}

	bm.jobsWg.Add(1)
	defer bm.jobsWg.Done()

	jobLog := bm.getOrCreateJobLog(jobName)
	jobLog.BeginRun()

	jobLogger := NewJobLogger(jobName)
	timestamp := time.Now().Format("20060102-150405")

	jobLogger.Info().Msg("Starting job")

	jobTmpDir := filepath.Join(bm.tmpDir, fmt.Sprintf("%s-%s", jobName, timestamp))
	if err := os.MkdirAll(jobTmpDir, 0755); err != nil {
		jobLogger.Error().Err(err).Str("path", jobTmpDir).Msg("Cannot create job temp dir")
		releaseReservation()
		return fmt.Errorf("create job temp dir: %w", err)
	}

	sourcesDir := filepath.Join(jobTmpDir, "sources")
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		jobLogger.Error().Err(err).Str("path", sourcesDir).Msg("Cannot create sources dir")
		releaseReservation()
		return fmt.Errorf("create sources dir: %w", err)
	}

	ctx := jobLogger.WithContext(jobExec.Ctx)

	sourcePaths := make(map[int]string)
	sourceTypeCounters := make(map[string]int)

	for i, sourceConfig := range job.Sources {
		sourceTypeCounters[sourceConfig.Type]++
		counter := sourceTypeCounters[sourceConfig.Type]

		sourceDirName := fmt.Sprintf("%s_%d", sourceConfig.Type, counter)
		fullPath := filepath.Join(sourcesDir, sourceDirName)

		if err := os.MkdirAll(fullPath, 0755); err != nil {
			jobLogger.Error().Err(err).
				Str("source_type", sourceConfig.Type).
				Str("path", fullPath).
				Msg("Cannot create source dir")
			releaseReservation()
			return fmt.Errorf("create source dir %s: %w", fullPath, err)
		}

		sourcePaths[i] = fullPath
	}

	jobExec.Job = job
	jobExec.Ctx = ctx
	jobExec.TmpDir = jobTmpDir
	jobExec.SourcesDir = sourcesDir
	jobExec.SourcePaths = sourcePaths
	jobExec.StartTime = time.Now()
	jobExec.Timestamp = timestamp
	jobExec.logger = jobLogger

	defer func() {
		bm.activeJobsMu.Lock()
		delete(bm.activeJobs, jobName)
		bm.activeJobsMu.Unlock()
		jobExec.Cancel()

		if err := os.RemoveAll(jobTmpDir); err != nil {
			jobLogger.Warn().Err(err).Str("path", jobTmpDir).Msg("Cannot remove job temp dir")
		}
	}()

	errorCh := make(chan error, 1)
	go func() {
		errorCh <- bm.executeJob(jobExec)
	}()

	select {
	case err := <-errorCh:
		duration := time.Since(jobExec.StartTime)
		if err != nil {
			jobLogger.Error().Err(err).Msg("Job failed")
			jobLog.Complete(false, err)
			metrics.ObserveFailure(jobName, duration.Seconds())
			bm.dispatchNotifications(jobExec.Job, notifier.Event{
				JobName: jobName, Success: false,
				Duration: duration.String(), Error: err.Error(),
			})
		} else {
			jobLogger.Info().Dur("duration", duration).Msg("Job done")
			jobLog.Complete(true, nil)
			var size int64
			if jobExec.ArchivePath != "" {
				if fi, statErr := os.Stat(jobExec.ArchivePath); statErr == nil {
					size = fi.Size()
				}
			}
			metrics.ObserveSuccess(jobName, duration.Seconds(), size)
			bm.dispatchNotifications(jobExec.Job, notifier.Event{
				JobName: jobName, Success: true,
				Duration: duration.String(),
			})
		}
		return err
	case <-ctx.Done():
		jobLogger.Warn().Msg("Job cancelled, waiting for worker")
		// Wait for the worker — otherwise the deferred RemoveAll races its files.
		<-errorCh
		duration := time.Since(jobExec.StartTime)
		jobLog.Complete(false, ctx.Err())
		metrics.ObserveFailure(jobName, duration.Seconds())
		bm.dispatchNotifications(jobExec.Job, notifier.Event{
			JobName: jobName, Success: false,
			Duration: duration.String(), Error: ctx.Err().Error(),
		})
		return ctx.Err()
	}
}

func (bm *Manager) CancelJob(jobName string) error {
	bm.activeJobsMu.Lock()
	defer bm.activeJobsMu.Unlock()

	jobExec, exists := bm.activeJobs[jobName]
	if !exists {
		return fmt.Errorf("job %q not running", jobName)
	}

	bm.systemLogger.Info().Str("job", jobName).Msg("Cancelling job")
	jobExec.Cancel()
	return nil
}

func (bm *Manager) executeJob(jobExec *jobExecution) error {
	jobLogger := jobExec.logger

	if script := jobExec.Job.BeforeScript; script != "" {
		jobLogger.Info().Msg("Running beforeScript")
		w := newScriptLineWriter(jobLogger, "beforeScript")
		err := utils.ExecScript(jobExec.Ctx, script, w)
		w.flush()
		if err != nil {
			return fmt.Errorf("beforeScript: %w", err)
		}
	}

	if script := jobExec.Job.AfterScript; script != "" {
		defer func() {
			// Fresh ctx — jobExec.Ctx may already be cancelled.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			jobLogger.Info().Msg("Running afterScript")
			w := newScriptLineWriter(jobLogger, "afterScript")
			err := utils.ExecScript(ctx, script, w)
			w.flush()
			if err != nil {
				jobLogger.Error().Err(err).Msg("afterScript failed")
			}
		}()
	}

	for i, sourceConfig := range jobExec.Job.Sources {
		fullPath, exists := jobExec.SourcePaths[i]
		if !exists {
			return fmt.Errorf("source path missing for index %d", i)
		}

		src := bm.sourceFactory.Create(sourceConfig.Type)
		if src == nil {
			return fmt.Errorf("unknown source type: %s", sourceConfig.Type)
		}

		sourceLogger := logger.NewSourceLogger(jobLogger, sourceConfig.Type, i+1)
		sourceCtx := sourceLogger.WithContext(jobExec.Ctx)

		sourceStart := time.Now()
		if err := src.Backup(sourceCtx, fullPath, sourceConfig); err != nil {
			sourceLogger.Error().Err(err).Dur("duration", time.Since(sourceStart)).Msg("Source failed")
			return fmt.Errorf("source %d (%s): %w", i+1, sourceConfig.Type, err)
		}

		sourceLogger.Info().Dur("duration", time.Since(sourceStart)).Msg("Source done")
	}

	compressor := bm.compressionFactory.Create(jobExec.Job.Compression)
	if compressor == nil {
		return fmt.Errorf("unknown compression type: %s", jobExec.Job.Compression)
	}

	compressionLogger := logger.NewCompressionLogger(jobLogger, jobExec.Job.Compression)
	compressionCtx := compressionLogger.WithContext(jobExec.Ctx)

	archiveName := fmt.Sprintf("%s-%s", jobExec.Job.Name, jobExec.Timestamp)
	compressionStart := time.Now()

	archivePath, err := compressor.Compress(compressionCtx, jobExec.SourcesDir, jobExec.TmpDir, archiveName)
	if err != nil {
		compressionLogger.Error().Err(err).Dur("duration", time.Since(compressionStart)).Msg("Compress failed")
		return fmt.Errorf("compress: %w", err)
	}
	jobExec.ArchivePath = archivePath

	if encCfg := jobExec.Job.Encryption; encCfg != nil {
		enc := bm.encryptorFactory.Create(encCfg.Type)
		if enc == nil {
			return fmt.Errorf("unknown encryption type: %s", encCfg.Type)
		}
		encCtx := jobLogger.With().Str("component", "encryptor").Logger().WithContext(jobExec.Ctx)
		encryptedPath, err := enc.Encrypt(encCtx, archivePath, *encCfg)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		archivePath = encryptedPath
		jobExec.ArchivePath = encryptedPath
	}

	uploadPaths := []string{archivePath}
	if splitCfg := jobExec.Job.Split; splitCfg != nil {
		chunkSize, err := utils.ParseSize(splitCfg.ChunkSize)
		if err != nil {
			return fmt.Errorf("split: %w", err)
		}
		splitLogger := jobLogger.With().Str("component", "split").Logger()
		splitStart := time.Now()
		parts, err := utils.SplitFile(archivePath, chunkSize)
		if err != nil {
			splitLogger.Error().Err(err).Msg("Split failed")
			return fmt.Errorf("split: %w", err)
		}
		splitLogger.Info().
			Int("parts", len(parts)).
			Int64("chunk_size_bytes", chunkSize).
			Dur("duration", time.Since(splitStart)).
			Msg("Archive split into parts")
		uploadPaths = parts
		jobExec.ArchivePath = parts[len(parts)-1]
	}

	wrapperRelDir := ""
	if len(uploadPaths) > 1 {
		var totalSize int64
		for _, p := range uploadPaths {
			st, err := os.Stat(p)
			if err != nil {
				return fmt.Errorf("stat part %s: %w", p, err)
			}
			totalSize += st.Size()
		}
		wrapperRelDir = storage.SplitWrapperName(filepath.Base(archivePath), len(uploadPaths), totalSize)
	}

	for _, storageConfig := range jobExec.Job.Storages {
		storageImpl := bm.storageFactory.Create(storageConfig.Type)
		if storageImpl == nil {
			return fmt.Errorf("unknown storage type: %s", storageConfig.Type)
		}

		storageLogger := logger.NewStorageLogger(jobLogger, storageConfig.Name)
		storageCtx := storageLogger.WithContext(jobExec.Ctx)

		if err := storageImpl.EnsureJobDir(storageCtx, jobExec.Job, storageConfig); err != nil {
			storageLogger.Error().Err(err).Msg("Cannot ensure job dir")
			return fmt.Errorf("ensure job dir on %s: %w", storageConfig.Name, err)
		}

		uploadStart := time.Now()
		for i, partPath := range uploadPaths {
			if err := storageImpl.UploadInto(storageCtx, partPath, jobExec.Job, wrapperRelDir, storageConfig); err != nil {
				storageLogger.Error().Err(err).
					Str("part", filepath.Base(partPath)).
					Int("index", i).
					Dur("duration", time.Since(uploadStart)).
					Msg("Upload failed")
				return fmt.Errorf("upload to %s: %w", storageConfig.Name, err)
			}
		}
		storageLogger.Info().
			Int("parts", len(uploadPaths)).
			Dur("duration", time.Since(uploadStart)).
			Msg("Upload done")

		storage.InvalidateSetCache(jobExec.Job.Name, storageConfig.Name)

		retentionLogger := logger.NewRetentionLogger(jobLogger, storageConfig.Name)
		retentionCtx := retentionLogger.WithContext(jobExec.Ctx)
		retentionStart := time.Now()

		if err := storage.ApplyRetention(retentionCtx, storageImpl, jobExec.Job, storageConfig); err != nil {
			retentionLogger.Warn().Err(err).Dur("duration", time.Since(retentionStart)).Msg("Retention failed")
		}
	}

	return nil
}

func (bm *Manager) reserveJobSlot(jobName string) bool {
	bm.activeJobsMu.Lock()
	defer bm.activeJobsMu.Unlock()
	if _, exists := bm.activeJobs[jobName]; exists {
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	bm.activeJobs[jobName] = &jobExecution{Ctx: ctx, Cancel: cancel}
	return true
}

func (bm *Manager) dispatchNotifications(job *config.JobConfig, ev notifier.Event) {
	if len(job.Notifiers) == 0 {
		return
	}
	d := notifier.NewDispatcher(job.Notifiers)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	d.Dispatch(ctx, ev)
}

func (bm *Manager) getOrCreateJobLog(jobName string) *JobLog {
	bm.jobLogsMu.Lock()
	defer bm.jobLogsMu.Unlock()
	jl, ok := bm.jobLogs[jobName]
	if !ok {
		jl = NewJobLog(jobName)
		bm.jobLogs[jobName] = jl
	}
	return jl
}

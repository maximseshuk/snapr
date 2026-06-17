package backup

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/storage"
)

type BackupFile struct {
	ID                    string    `json:"id"`
	JobName               string    `json:"jobName"`
	Name                  string    `json:"name"`
	Path                  string    `json:"path"`
	Size                  int64     `json:"size"`
	CreatedAt             string    `json:"createdAt"`
	StorageType           string    `json:"storageType"`
	IsSplit               bool      `json:"isSplit"`
	PartsCount            int       `json:"partsCount,omitempty"`
	FullDownloadSupported bool      `json:"fullDownloadSupported"`
	ModTime               time.Time `json:"-"`
}

func findJob(cfg *config.Config, name string) *config.JobConfig {
	for i := range cfg.Jobs {
		if cfg.Jobs[i].Name == name {
			return &cfg.Jobs[i]
		}
	}
	return nil
}

func (bm *Manager) resolveStorage(job *config.JobConfig) (config.StorageConfig, bool) {
	if len(job.Storages) == 0 {
		return config.StorageConfig{}, false
	}

	if job.DefaultStorage != "" {
		for _, s := range job.Storages {
			if s.Name == job.DefaultStorage {
				return s, true
			}
		}
		bm.systemLogger.Warn().
			Str("job", job.Name).
			Str("default_storage", job.DefaultStorage).
			Msg("Default storage not found, using first")
	}

	return job.Storages[0], true
}

// GetJobRunInfo returns the latest run metadata, or nil if the job hasn't run yet.
func (bm *Manager) GetJobRunInfo(jobName string) *JobLog {
	bm.jobLogsMu.RLock()
	defer bm.jobLogsMu.RUnlock()
	return bm.jobLogs[jobName]
}

func (bm *Manager) GetJobBackups(ctx context.Context, jobName string) ([]BackupFile, error) {
	job := findJob(bm.cfg, jobName)
	if job == nil {
		return nil, fmt.Errorf("%w: %s", ErrJobNotFound, jobName)
	}

	storageToUse, ok := bm.resolveStorage(job)
	if !ok {
		return []BackupFile{}, nil
	}

	storageHandler := bm.storageFactory.Create(storageToUse.Type)
	if storageHandler == nil {
		return nil, fmt.Errorf("unknown storage type: %s", storageToUse.Type)
	}

	sets, err := storage.ListBackupSets(ctx, storageHandler, job, storageToUse)
	if err != nil {
		return nil, fmt.Errorf("list sets from %s: %w", storageToUse.Type, err)
	}

	backups := make([]BackupFile, 0, len(sets))
	for _, set := range sets {
		path := backupPath(storageToUse, jobName, &set)

		bf := BackupFile{
			ID:                    set.ID,
			JobName:               jobName,
			Name:                  set.ID,
			Path:                  path,
			Size:                  set.TotalSize,
			CreatedAt:             set.LastModified.Format(time.RFC3339),
			StorageType:           storageToUse.Type,
			IsSplit:               set.IsSplit,
			PartsCount:            set.PartsCount,
			FullDownloadSupported: fullDownloadSupported(storageToUse, set.IsSplit),
			ModTime:               set.LastModified,
		}
		backups = append(backups, bf)
	}

	return backups, nil
}

// fullDownloadSupported reports whether the full-set download endpoint can
// stream a split snapshot. Redirect-based downloads (S3 signed URLs, Bunny
// Pull Zone) can't chain N parts into one response — only per-part works.
func fullDownloadSupported(storageConfig config.StorageConfig, isSplit bool) bool {
	if !isSplit {
		return true
	}
	if storageConfig.Type == "s3" && storageConfig.DownloadMode == "signed" {
		return false
	}
	if storageConfig.Type == "bunny" && storageConfig.PullZoneHostname != "" && storageConfig.PullZoneTokenAuthKey != "" {
		return false
	}
	return true
}

func backupPath(storageConfig config.StorageConfig, jobName string, set *storage.BackupSet) string {
	leaf := set.ID
	if set.IsSplit && set.WrapperName != "" {
		leaf = set.WrapperName
	}
	seg := storage.JobNameSegment(storageConfig.IncludeJobName, jobName)
	switch storageConfig.Type {
	case "local":
		return filepath.Join(storageConfig.Path, seg, leaf)
	case "s3":
		parts := joinNonEmpty(strings.Trim(storageConfig.Path, "/"), seg, leaf)
		return fmt.Sprintf("s3://%s/%s", storageConfig.Bucket, parts)
	default:
		return joinNonEmpty(seg, leaf)
	}
}

// joinNonEmpty joins the non-empty segments with "/", skipping blanks so an
// omitted job-name segment doesn't leave a doubled or leading slash.
func joinNonEmpty(segments ...string) string {
	out := make([]string, 0, len(segments))
	for _, s := range segments {
		if s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, "/")
}

func (bm *Manager) GetJobLogInfo(jobName string) (*JobLog, error) {
	bm.jobLogsMu.RLock()
	defer bm.jobLogsMu.RUnlock()

	jobLog, exists := bm.jobLogs[jobName]
	if !exists {
		return nil, fmt.Errorf("no logs for job %q", jobName)
	}
	if jobLog.GetStatus() == "idle" {
		return nil, fmt.Errorf("no runs yet for job %q", jobName)
	}
	return jobLog, nil
}

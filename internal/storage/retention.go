package storage

import (
	"context"
	"fmt"
	"sort"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
)

func ApplyRetention(ctx context.Context, storage Storage, job *config.JobConfig, storageConfig config.StorageConfig) error {
	logger := zerolog.Ctx(ctx)

	if job.Retention.Last <= 0 {
		logger.Debug().Msg("Retention disabled, skipping")
		return nil
	}

	logger.Debug().
		Int("keep_last", job.Retention.Last).
		Str("storage", storage.GetType()).
		Msg("Applying retention")

	sets, err := ListBackupSets(ctx, storage, job, storageConfig)
	if err != nil {
		return fmt.Errorf("list sets: %w", err)
	}

	if len(sets) <= job.Retention.Last {
		logger.Debug().
			Int("sets", len(sets)).
			Int("keep_last", job.Retention.Last).
			Msg("Nothing to delete")
		return nil
	}

	sort.Slice(sets, func(i, j int) bool {
		return sets[i].LastModified.After(sets[j].LastModified)
	})
	setsToDelete := sets[job.Retention.Last:]

	deletedSets := 0
	var deletionErrors []string

	for _, set := range setsToDelete {
		if set.IsSplit {
			if err := storage.DeleteWrapper(ctx, job, set.WrapperName, storageConfig); err != nil {
				logger.Error().Err(err).Str("wrapper", set.WrapperName).Msg("Cannot delete split wrapper")
				deletionErrors = append(deletionErrors, fmt.Sprintf("%s: %v", set.WrapperName, err))
				continue
			}
			deletedSets++
			logger.Debug().Str("wrapper", set.WrapperName).Msg("Deleted split wrapper")
			continue
		}
		setOK := true
		for _, file := range set.Files {
			if err := storage.DeleteFile(ctx, job, file.Name, storageConfig); err != nil {
				logger.Error().Err(err).Str("file", file.Name).Str("set", set.ID).Msg("Cannot delete file")
				deletionErrors = append(deletionErrors, fmt.Sprintf("%s: %v", file.Name, err))
				setOK = false
				continue
			}
			logger.Debug().Str("file", file.Name).Msg("Deleted")
		}
		if setOK {
			deletedSets++
		}
	}

	InvalidateSetCache(job.Name, storageConfig.Name)

	if len(deletionErrors) > 0 {
		logger.Warn().
			Int("errors", len(deletionErrors)).
			Int("deleted_sets", deletedSets).
			Strs("failed", deletionErrors).
			Msg("Some deletions failed")
	}

	logger.Info().
		Int("deleted_sets", deletedSets).
		Int("kept", job.Retention.Last).
		Msg("Retention applied")
	return nil
}

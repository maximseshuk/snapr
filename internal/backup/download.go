package backup

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/storage"
)

func (bm *Manager) OpenBackupDownload(ctx context.Context, jobName, filename string) (*storage.DownloadResult, error) {
	jobConfig, err := bm.GetJobConfig(jobName)
	if err != nil {
		return nil, err
	}

	storageToUse, ok := bm.resolveStorage(jobConfig)
	if !ok {
		return nil, fmt.Errorf("no storages for job %q", jobName)
	}

	storageHandler := bm.storageFactory.Create(storageToUse.Type)
	if storageHandler == nil {
		return nil, fmt.Errorf("unknown storage type: %s", storageToUse.Type)
	}

	downloader, ok := storageHandler.(storage.Downloader)
	if !ok {
		return nil, storage.ErrDownloadNotSupported
	}

	sets, err := storage.ListBackupSets(ctx, storageHandler, jobConfig, storageToUse)
	if err != nil {
		return nil, fmt.Errorf("list sets: %w", err)
	}
	var match *storage.BackupSet
	for i := range sets {
		if sets[i].ID == filename {
			match = &sets[i]
			break
		}
	}

	if match == nil {
		for i := range sets {
			if !sets[i].IsSplit {
				continue
			}
			parts, err := storageHandler.ListWrapperParts(ctx, jobConfig, sets[i].WrapperName, storageToUse)
			if err != nil {
				continue
			}
			for _, p := range parts {
				if p.Name == filename {
					return downloader.Download(ctx, jobConfig, sets[i].WrapperName, filename, storageToUse)
				}
			}
		}
		return nil, fmt.Errorf("backup not found: %s", filename)
	}

	if !match.IsSplit {
		result, err := downloader.Download(ctx, jobConfig, "", match.Files[0].Name, storageToUse)
		if err != nil {
			return nil, fmt.Errorf("download: %w", err)
		}
		return result, nil
	}

	parts, err := storageHandler.ListWrapperParts(ctx, jobConfig, match.WrapperName, storageToUse)
	if err != nil {
		return nil, fmt.Errorf("list wrapper parts: %w", err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("split set %q has no parts on disk", match.ID)
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].Name < parts[j].Name })

	first, err := downloader.Download(ctx, jobConfig, match.WrapperName, parts[0].Name, storageToUse)
	if err != nil {
		return nil, fmt.Errorf("download part %s: %w", parts[0].Name, err)
	}
	if first.RedirectURL != "" {
		return nil, fmt.Errorf("%w: split archives cannot be downloaded from redirect-based storages", storage.ErrDownloadNotSupported)
	}

	partNames := make([]string, len(parts))
	totalSize := int64(0)
	for i, f := range parts {
		partNames[i] = f.Name
		totalSize += f.Size
	}
	body := newSplitDownloadBody(ctx, downloader, jobConfig, storageToUse, match.WrapperName, partNames, first.Body)
	return &storage.DownloadResult{Body: body, Size: totalSize}, nil
}

type splitDownloadBody struct {
	ctx        context.Context
	downloader storage.Downloader
	job        *config.JobConfig
	storage    config.StorageConfig
	wrapper    string
	parts      []string
	idx        int
	current    io.ReadCloser
	err        error
}

func newSplitDownloadBody(
	ctx context.Context,
	downloader storage.Downloader,
	job *config.JobConfig,
	storageCfg config.StorageConfig,
	wrapperRelDir string,
	parts []string,
	first io.ReadCloser,
) *splitDownloadBody {
	return &splitDownloadBody{
		ctx:        ctx,
		downloader: downloader,
		job:        job,
		storage:    storageCfg,
		wrapper:    wrapperRelDir,
		parts:      parts,
		idx:        0,
		current:    first,
	}
}

func (b *splitDownloadBody) Read(p []byte) (int, error) {
	if b.err != nil {
		return 0, b.err
	}
	for {
		if b.current == nil {
			b.err = io.EOF
			return 0, io.EOF
		}
		n, err := b.current.Read(p)
		if n > 0 {
			return n, nil
		}
		if err == io.EOF {
			if cerr := b.current.Close(); cerr != nil {
				b.err = fmt.Errorf("close part %s: %w", b.parts[b.idx], cerr)
				b.current = nil
				return 0, b.err
			}
			b.idx++
			if b.idx >= len(b.parts) {
				b.current = nil
				b.err = io.EOF
				return 0, io.EOF
			}
			next, derr := b.downloader.Download(b.ctx, b.job, b.wrapper, b.parts[b.idx], b.storage)
			if derr != nil {
				b.err = fmt.Errorf("open part %s: %w", b.parts[b.idx], derr)
				b.current = nil
				return 0, b.err
			}
			if next.RedirectURL != "" {
				_ = next.Body.Close()
				b.err = fmt.Errorf("part %s returned a redirect URL mid-stream", b.parts[b.idx])
				b.current = nil
				return 0, b.err
			}
			b.current = next.Body
			continue
		}
		if err != nil {
			b.err = err
			return 0, err
		}
	}
}

func (b *splitDownloadBody) Close() error {
	if b.current != nil {
		err := b.current.Close()
		b.current = nil
		return err
	}
	return nil
}

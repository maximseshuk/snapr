package storage

import (
	"context"
	"sort"
	"sync"
	"time"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

// BackupSet is one logical snapshot. For split snapshots Files is nil — parts
// inside the wrapper are listed only on demand.
type BackupSet struct {
	ID           string
	Files        []FileInfo
	LastModified time.Time
	TotalSize    int64
	IsSplit      bool
	PartsCount   int
	WrapperName  string
}

func GroupBackupSets(files []FileInfo, _ string) []BackupSet {
	groups := make(map[string]*BackupSet)

	for _, f := range files {
		if f.Wrapper {
			base, parts, size, ok := ParseSplitWrapper(f.Name)
			if !ok {
				continue
			}
			set := &BackupSet{
				ID:           base,
				IsSplit:      true,
				PartsCount:   parts,
				TotalSize:    size,
				WrapperName:  f.Name,
				LastModified: f.LastModified,
			}
			groups[base] = set
			continue
		}

		set, ok := groups[f.Name]
		if !ok {
			set = &BackupSet{ID: f.Name}
			groups[f.Name] = set
		}
		set.Files = append(set.Files, f)
		set.TotalSize += f.Size
		if f.LastModified.After(set.LastModified) {
			set.LastModified = f.LastModified
		}
		if !set.IsSplit {
			set.PartsCount = len(set.Files)
		}
	}

	out := make([]BackupSet, 0, len(groups))
	for _, set := range groups {
		sort.Slice(set.Files, func(i, j int) bool {
			return set.Files[i].Name < set.Files[j].Name
		})
		out = append(out, *set)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastModified.Equal(out[j].LastModified) {
			return out[i].ID < out[j].ID
		}
		return out[i].LastModified.After(out[j].LastModified)
	})
	return out
}

const setCacheTTL = 5 * time.Minute

type cachedEntry struct {
	sets      []BackupSet
	expiresAt time.Time
}

type setCache struct {
	mu      sync.Mutex
	entries map[string]cachedEntry
}

func newSetCache() *setCache {
	return &setCache{entries: make(map[string]cachedEntry)}
}

func (c *setCache) get(key string) ([]BackupSet, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.sets, true
}

func (c *setCache) put(key string, sets []BackupSet, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cachedEntry{sets: sets, expiresAt: time.Now().Add(ttl)}
}

func (c *setCache) invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

var globalSetCache = newSetCache()

func cacheKey(jobName, storageName string) string {
	return jobName + "|" + storageName
}

// ListBackupSets reads through a TTL cache. Callers must InvalidateSetCache after Upload/Delete.
func ListBackupSets(ctx context.Context, st Storage, job *pkgconfig.JobConfig, storageConfig pkgconfig.StorageConfig) ([]BackupSet, error) {
	key := cacheKey(job.Name, storageConfig.Name)
	if sets, ok := globalSetCache.get(key); ok {
		return sets, nil
	}

	files, err := st.ListFiles(ctx, job, storageConfig)
	if err != nil {
		return nil, err
	}
	sets := GroupBackupSets(files, job.Name)
	globalSetCache.put(key, sets, setCacheTTL)
	return sets, nil
}

func InvalidateSetCache(jobName, storageName string) {
	globalSetCache.invalidate(cacheKey(jobName, storageName))
}

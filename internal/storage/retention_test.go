package storage

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

// fakeStorage is a minimal in-memory Storage for testing retention.
type fakeStorage struct {
	files           []FileInfo
	deletedFiles    []string
	deletedWrappers []string
}

func (f *fakeStorage) GetType() string { return "fake" }
func (f *fakeStorage) EnsureJobDir(context.Context, *pkgconfig.JobConfig, pkgconfig.StorageConfig) error {
	return nil
}
func (f *fakeStorage) UploadInto(context.Context, string, *pkgconfig.JobConfig, string, pkgconfig.StorageConfig) error {
	return nil
}
func (f *fakeStorage) ListFiles(context.Context, *pkgconfig.JobConfig, pkgconfig.StorageConfig) ([]FileInfo, error) {
	return f.files, nil
}
func (f *fakeStorage) ListWrapperParts(context.Context, *pkgconfig.JobConfig, string, pkgconfig.StorageConfig) ([]FileInfo, error) {
	return nil, nil
}
func (f *fakeStorage) DeleteFile(_ context.Context, _ *pkgconfig.JobConfig, name string, _ pkgconfig.StorageConfig) error {
	f.deletedFiles = append(f.deletedFiles, name)
	return nil
}
func (f *fakeStorage) DeleteWrapper(_ context.Context, _ *pkgconfig.JobConfig, name string, _ pkgconfig.StorageConfig) error {
	f.deletedWrappers = append(f.deletedWrappers, name)
	return nil
}

func TestApplyRetention_KeepsLastN(t *testing.T) {
	t.Cleanup(func() { InvalidateSetCache("rjob", "rfake") })

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	t4 := time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC)

	fake := &fakeStorage{files: []FileInfo{
		{Name: "a.tar.gz", LastModified: t1, Size: 1},
		{Name: "b.tar.gz", LastModified: t2, Size: 1},
		{Name: "c.tar.gz", LastModified: t3, Size: 1},
		{Name: "d.tar.gz", LastModified: t4, Size: 1},
	}}

	job := &pkgconfig.JobConfig{Name: "rjob", Retention: pkgconfig.RetentionConfig{Last: 2}}
	cfg := pkgconfig.StorageConfig{Name: "rfake"}

	require.NoError(t, ApplyRetention(context.Background(), fake, job, cfg))

	sort.Strings(fake.deletedFiles)
	assert.Equal(t, []string{"a.tar.gz", "b.tar.gz"}, fake.deletedFiles, "must delete the 2 oldest")
	assert.Empty(t, fake.deletedWrappers)
}

func TestApplyRetention_NothingToDelete(t *testing.T) {
	t.Cleanup(func() { InvalidateSetCache("rjob2", "rfake2") })

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fake := &fakeStorage{files: []FileInfo{
		{Name: "a.tar.gz", LastModified: t1, Size: 1},
	}}
	job := &pkgconfig.JobConfig{Name: "rjob2", Retention: pkgconfig.RetentionConfig{Last: 5}}
	cfg := pkgconfig.StorageConfig{Name: "rfake2"}

	require.NoError(t, ApplyRetention(context.Background(), fake, job, cfg))
	assert.Empty(t, fake.deletedFiles)
}

func TestApplyRetention_DisabledByZero(t *testing.T) {
	t.Cleanup(func() { InvalidateSetCache("rjob3", "rfake3") })

	fake := &fakeStorage{files: []FileInfo{{Name: "a", LastModified: time.Now(), Size: 1}}}
	job := &pkgconfig.JobConfig{Name: "rjob3"}
	cfg := pkgconfig.StorageConfig{Name: "rfake3"}

	require.NoError(t, ApplyRetention(context.Background(), fake, job, cfg))
	assert.Empty(t, fake.deletedFiles)
}

func TestApplyRetention_DeletesOldWrappers(t *testing.T) {
	t.Cleanup(func() { InvalidateSetCache("rjob4", "rfake4") })

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	wrapperOld := SplitWrapperName("old.tar.gz", 2, 100)
	wrapperNew := SplitWrapperName("new.tar.gz", 2, 100)

	fake := &fakeStorage{files: []FileInfo{
		{Name: wrapperOld, LastModified: t1, Wrapper: true},
		{Name: wrapperNew, LastModified: t2, Wrapper: true},
	}}

	job := &pkgconfig.JobConfig{Name: "rjob4", Retention: pkgconfig.RetentionConfig{Last: 1}}
	cfg := pkgconfig.StorageConfig{Name: "rfake4"}

	require.NoError(t, ApplyRetention(context.Background(), fake, job, cfg))
	assert.Equal(t, []string{wrapperOld}, fake.deletedWrappers)
	assert.Empty(t, fake.deletedFiles)
}

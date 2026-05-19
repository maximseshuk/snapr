package backup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/storage"
)

type fakeDownloader struct {
	parts map[string]string
	err   map[string]error
	calls []string
}

func (f *fakeDownloader) Download(_ context.Context, _ *config.JobConfig, _, fileName string, _ config.StorageConfig) (*storage.DownloadResult, error) {
	f.calls = append(f.calls, fileName)
	if err, ok := f.err[fileName]; ok {
		return nil, err
	}
	body, ok := f.parts[fileName]
	if !ok {
		return nil, errors.New("not found: " + fileName)
	}
	return &storage.DownloadResult{
		Body: io.NopCloser(strings.NewReader(body)),
		Size: int64(len(body)),
	}, nil
}

func TestSplitDownloadBody_ConcatenatesParts(t *testing.T) {
	dl := &fakeDownloader{parts: map[string]string{
		"a.part-aaa": "AAAA",
		"a.part-aab": "BBBB",
		"a.part-aac": "CC",
	}}
	first, err := dl.Download(context.Background(), nil, "wrap", "a.part-aaa", config.StorageConfig{})
	require.NoError(t, err)

	body := newSplitDownloadBody(context.Background(), dl, nil, config.StorageConfig{}, "wrap",
		[]string{"a.part-aaa", "a.part-aab", "a.part-aac"}, first.Body)

	got, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, "AAAABBBBCC", string(got))
	require.NoError(t, body.Close())
}

func TestSplitDownloadBody_PropagatesDownloadError(t *testing.T) {
	dl := &fakeDownloader{
		parts: map[string]string{"p1": "AAAA"},
		err:   map[string]error{"p2": errors.New("network down")},
	}
	first, err := dl.Download(context.Background(), nil, "w", "p1", config.StorageConfig{})
	require.NoError(t, err)

	body := newSplitDownloadBody(context.Background(), dl, nil, config.StorageConfig{}, "w",
		[]string{"p1", "p2"}, first.Body)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "p2")
	assert.Contains(t, err.Error(), "network down")
	assert.Equal(t, "AAAA", buf.String(), "must yield first part fully before failing on next")
}

// redirectDownloader returns RedirectURL mid-stream to simulate signed-URL backends.
type redirectDownloader struct{ partsBefore *fakeDownloader }

func (r *redirectDownloader) Download(ctx context.Context, j *config.JobConfig, wrapper, fileName string, s config.StorageConfig) (*storage.DownloadResult, error) {
	if fileName == "p2" {
		return &storage.DownloadResult{RedirectURL: "https://signed.example/p2", Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	return r.partsBefore.Download(ctx, j, wrapper, fileName, s)
}

func TestSplitDownloadBody_RejectsMidStreamRedirect(t *testing.T) {
	inner := &fakeDownloader{parts: map[string]string{"p1": "AAAA"}}
	dl := &redirectDownloader{partsBefore: inner}

	first, err := dl.Download(context.Background(), nil, "w", "p1", config.StorageConfig{})
	require.NoError(t, err)

	body := newSplitDownloadBody(context.Background(), dl, nil, config.StorageConfig{}, "w",
		[]string{"p1", "p2"}, first.Body)

	_, err = io.ReadAll(body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redirect")
}

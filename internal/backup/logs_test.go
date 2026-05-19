package backup

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobLog_Lifecycle(t *testing.T) {
	jl := NewJobLog("myjob")
	assert.Equal(t, "myjob", jl.JobName)
	assert.Equal(t, "idle", jl.GetStatus())
	assert.Nil(t, jl.GetEndTime())

	jl.BeginRun()
	assert.Equal(t, "running", jl.GetStatus())
	assert.False(t, jl.GetStartTime().IsZero())
	assert.Nil(t, jl.GetEndTime())

	jl.Complete(true, nil)
	assert.Equal(t, "success", jl.GetStatus())
	assert.NotNil(t, jl.GetEndTime())
	assert.Empty(t, jl.GetError())
}

func TestJobLog_Failure(t *testing.T) {
	jl := NewJobLog("j")
	jl.BeginRun()
	jl.Complete(false, errors.New("disk full"))
	assert.Equal(t, "failed", jl.GetStatus())
	assert.Equal(t, "disk full", jl.GetError())
}

func TestJobLog_RerunResetsState(t *testing.T) {
	jl := NewJobLog("j")
	jl.BeginRun()
	jl.Complete(false, errors.New("boom"))
	assert.Equal(t, "failed", jl.GetStatus())
	assert.Equal(t, "boom", jl.GetError())

	jl.BeginRun()
	assert.Equal(t, "running", jl.GetStatus())
	assert.Empty(t, jl.GetError(), "BeginRun must clear previous error")
	assert.Nil(t, jl.GetEndTime(), "BeginRun must clear previous endTime")
}

func TestJobLog_ConcurrentSafe(t *testing.T) {
	// Just verify race detector doesn't fire under -race.
	jl := NewJobLog("j")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				jl.BeginRun()
				jl.Complete(true, nil)
			} else {
				_ = jl.GetStatus()
				_ = jl.GetEndTime()
				_ = jl.GetError()
			}
		}(i)
	}
	wg.Wait()
}

func TestScriptLineWriter_LineFlushing(t *testing.T) {
	var buf strings.Builder
	lg := zerolog.New(&buf)
	w := newScriptLineWriter(lg, "before")

	n, err := w.Write([]byte("first line\nsecond"))
	require.NoError(t, err)
	assert.Equal(t, 17, n)

	// First line emitted, "second" still buffered.
	assert.Contains(t, buf.String(), "first line")
	assert.NotContains(t, buf.String(), "second")

	_, err = w.Write([]byte(" continues\n"))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "second continues")
}

func TestScriptLineWriter_FlushTrailing(t *testing.T) {
	var buf strings.Builder
	lg := zerolog.New(&buf)
	w := newScriptLineWriter(lg, "x")

	_, err := w.Write([]byte("no-newline"))
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "no-newline", "writer must wait for newline")

	w.flush()
	assert.Contains(t, buf.String(), "no-newline")
}

func TestScriptLineWriter_TagInOutput(t *testing.T) {
	var buf strings.Builder
	lg := zerolog.New(&buf)
	w := newScriptLineWriter(lg, "beforeScript")

	_, _ = w.Write([]byte("hello\n"))
	assert.Contains(t, buf.String(), `"script":"beforeScript"`)
}

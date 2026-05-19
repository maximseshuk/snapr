package backup

import (
	"bytes"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type JobLog struct {
	JobName string

	mu        sync.RWMutex
	startTime time.Time
	endTime   *time.Time
	status    string // "idle", "running", "success", "failed"
	errMsg    string
}

func NewJobLog(jobName string) *JobLog {
	return &JobLog{
		JobName: jobName,
		status:  "idle",
	}
}

func (jl *JobLog) BeginRun() {
	jl.mu.Lock()
	defer jl.mu.Unlock()
	jl.startTime = time.Now()
	jl.endTime = nil
	jl.status = "running"
	jl.errMsg = ""
}

func (jl *JobLog) Complete(success bool, err error) {
	jl.mu.Lock()
	defer jl.mu.Unlock()

	now := time.Now()
	jl.endTime = &now

	if success {
		jl.status = "success"
		return
	}
	jl.status = "failed"
	if err != nil {
		jl.errMsg = err.Error()
	}
}

func (jl *JobLog) GetStartTime() time.Time {
	jl.mu.RLock()
	defer jl.mu.RUnlock()
	return jl.startTime
}

func (jl *JobLog) GetEndTime() *time.Time {
	jl.mu.RLock()
	defer jl.mu.RUnlock()
	return jl.endTime
}

func (jl *JobLog) GetStatus() string {
	jl.mu.RLock()
	defer jl.mu.RUnlock()
	return jl.status
}

func (jl *JobLog) GetError() string {
	jl.mu.RLock()
	defer jl.mu.RUnlock()
	return jl.errMsg
}

func NewJobLogger(jobName string) zerolog.Logger {
	return log.With().Str("job", jobName).Logger()
}

type scriptLineWriter struct {
	logger zerolog.Logger
	tag    string
	mu     sync.Mutex
	buffer bytes.Buffer
}

func newScriptLineWriter(lg zerolog.Logger, tag string) *scriptLineWriter {
	return &scriptLineWriter{logger: lg, tag: tag}
}

func (s *scriptLineWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	s.buffer.Write(p)
	for {
		data := s.buffer.Bytes()
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}
		line := string(data[:idx])
		s.buffer.Next(idx + 1)
		s.emit(line)
	}
	s.mu.Unlock()
	return len(p), nil
}

func (s *scriptLineWriter) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.buffer.Len() == 0 {
		return
	}
	s.emit(s.buffer.String())
	s.buffer.Reset()
}

func (s *scriptLineWriter) emit(line string) {
	s.logger.Info().Str("script", s.tag).Msg(line)
}

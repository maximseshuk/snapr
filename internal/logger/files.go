package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/maximseshuk/snapr/internal/config"
)

type FileSinks struct {
	cfg config.LogsConfig
	dir string

	system *lumberjack.Logger

	mu       sync.RWMutex
	jobSinks map[string]*lumberjack.Logger
}

func NewFileSinks(cfg config.LogsConfig) (*FileSinks, error) {
	if !cfg.System && !cfg.PerJob {
		return nil, nil
	}
	dir := cfg.Path
	if dir == "" {
		dir = "./logs"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create logs dir %s: %w", dir, err)
	}
	if cfg.PerJob {
		if err := os.MkdirAll(filepath.Join(dir, "jobs"), 0o755); err != nil {
			return nil, fmt.Errorf("create jobs logs dir: %w", err)
		}
	}

	fs := &FileSinks{
		cfg:      cfg,
		dir:      dir,
		jobSinks: make(map[string]*lumberjack.Logger),
	}
	if cfg.System {
		fs.system = newRotator(filepath.Join(dir, "snapr.log"), cfg)
	}
	return fs, nil
}

func (fs *FileSinks) Write(p []byte) (int, error) {
	if fs.system != nil {
		_, _ = fs.system.Write(p)
	}

	if fs.cfg.PerJob {
		if name := extractJob(p); name != "" {
			sink := fs.jobSink(name)
			_, _ = sink.Write(p)
		}
	}
	return len(p), nil
}

func (fs *FileSinks) SystemEnabled() bool {
	return fs != nil && fs.cfg.System
}

func (fs *FileSinks) PerJobEnabled() bool {
	return fs != nil && fs.cfg.PerJob
}

func (fs *FileSinks) JobLogPath(jobName string) string {
	if fs == nil || !fs.cfg.PerJob {
		return ""
	}
	return filepath.Join(fs.dir, "jobs", sanitizeJobName(jobName)+".log")
}

func (fs *FileSinks) SystemLogPath() string {
	if fs == nil || !fs.cfg.System {
		return ""
	}
	return filepath.Join(fs.dir, "snapr.log")
}

func (fs *FileSinks) Close() {
	if fs == nil {
		return
	}
	if fs.system != nil {
		_ = fs.system.Close()
	}
	fs.mu.Lock()
	for _, s := range fs.jobSinks {
		_ = s.Close()
	}
	fs.mu.Unlock()
}

func (fs *FileSinks) jobSink(jobName string) io.Writer {
	safe := sanitizeJobName(jobName)

	fs.mu.RLock()
	sink, ok := fs.jobSinks[safe]
	fs.mu.RUnlock()
	if ok {
		return sink
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()
	if sink, ok = fs.jobSinks[safe]; ok {
		return sink
	}
	sink = newRotator(filepath.Join(fs.dir, "jobs", safe+".log"), fs.cfg)
	fs.jobSinks[safe] = sink
	return sink
}

func newRotator(path string, cfg config.LogsConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   path,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}

func extractJob(p []byte) string {
	idx := -1
	for i := 0; i+6 < len(p); i++ {
		if p[i] == '"' && p[i+1] == 'j' && p[i+2] == 'o' && p[i+3] == 'b' && p[i+4] == '"' && p[i+5] == ':' {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ""
	}
	var event struct {
		Job string `json:"job"`
	}
	if err := json.Unmarshal(p, &event); err != nil {
		return ""
	}
	return event.Job
}

func sanitizeJobName(name string) string {
	if name == "" {
		return "_unknown"
	}
	out := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '.' || c == '_' || c == '-':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}

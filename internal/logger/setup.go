package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/maximseshuk/snapr/internal/utils"
)

type preFileBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *preFileBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *preFileBuffer) Drain() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]byte, b.buf.Len())
	copy(out, b.buf.Bytes())
	b.buf.Reset()
	return out
}

var earlyBuffer = &preFileBuffer{}

// Setup installs the global zerolog logger. Pass nil to buffer events until the
// next call (used during early bootstrap before config is loaded).
func Setup(sinks *FileSinks) {
	if !isDev() {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	console := NewConsoleWriter(os.Stdout)

	var w io.Writer
	if sinks != nil {
		if early := earlyBuffer.Drain(); len(early) > 0 {
			_, _ = sinks.Write(early)
		}
		w = zerolog.MultiLevelWriter(console, sinks)
	} else {
		w = zerolog.MultiLevelWriter(console, earlyBuffer)
	}

	log.Logger = zerolog.New(w).With().Timestamp().Logger()
}

func SetGlobalLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

func GetLevel() zerolog.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		if isDev() {
			return zerolog.DebugLevel
		}
		return zerolog.InfoLevel
	}
}

func isDev() bool {
	return utils.GetEnvironment() != "production"
}

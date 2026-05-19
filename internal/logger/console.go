package logger

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

func NewConsoleWriter(out io.Writer) zerolog.ConsoleWriter {
	cw := zerolog.ConsoleWriter{
		Out:           out,
		TimeFormat:    "15:04:05",
		NoColor:       false,
		PartsOrder:    []string{"time", "level", "job", "component", "message"},
		FieldsExclude: []string{"job", "component"},
	}

	cw.FormatLevel = func(i interface{}) string {
		ll, ok := i.(string)
		if !ok {
			return ""
		}
		switch ll {
		case "trace":
			return "\033[35m[TRACE]\033[0m"
		case "debug":
			return "\033[36m[DEBUG]\033[0m"
		case "info":
			return "\033[32m[INFO]\033[0m"
		case "warn":
			return "\033[33m[WARN]\033[0m"
		case "error":
			return "\033[31m[ERROR]\033[0m"
		case "fatal":
			return "\033[91m[FATAL]\033[0m"
		case "panic":
			return "\033[95m[PANIC]\033[0m"
		default:
			return fmt.Sprintf("[%s]", strings.ToUpper(ll))
		}
	}

	cw.FormatPartValueByName = func(i interface{}, name string) string {
		switch name {
		case "job":
			if i == nil {
				return ""
			}
			return fmt.Sprintf("\033[34m[JOB:%s]\033[0m", i)
		case "component":
			if i == nil {
				return ""
			}
			s, ok := i.(string)
			if !ok {
				return ""
			}
			return fmt.Sprintf("\033[90m[%s]\033[0m", strings.ToUpper(s))
		}
		return fmt.Sprintf("%s", i)
	}

	return cw
}

var renderPool = sync.Pool{
	New: func() any {
		buf := &bytes.Buffer{}
		cw := NewConsoleWriter(buf)
		return &renderState{buf: buf, cw: cw}
	},
}

type renderState struct {
	buf *bytes.Buffer
	cw  zerolog.ConsoleWriter
}

func RenderJSONToANSI(jsonLine []byte) string {
	if len(jsonLine) == 0 {
		return ""
	}
	st := renderPool.Get().(*renderState)
	st.buf.Reset()
	defer renderPool.Put(st)

	if _, err := st.cw.Write(jsonLine); err != nil {
		return string(jsonLine)
	}
	out := st.buf.Bytes()
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return string(out)
}

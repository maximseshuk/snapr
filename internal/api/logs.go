package api

import (
	"github.com/maximseshuk/snapr/internal/logger"
)

// maxTailRequest bounds ?tail=N to keep one request from reading millions of lines.
const maxTailRequest = 50000

func (api *BackupAPI) systemLogPath() string {
	if api.sinks == nil {
		return ""
	}
	return api.sinks.SystemLogPath()
}

func (api *BackupAPI) jobLogPath(jobName string) string {
	if api.sinks == nil {
		return ""
	}
	return api.sinks.JobLogPath(jobName)
}

func readLogTail(path string, n int) ([]string, error) {
	if path == "" {
		return []string{}, nil
	}
	raw, err := logger.TailFile(path, n)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		if len(line) == 0 {
			continue
		}
		out = append(out, logger.RenderJSONToANSI(line))
	}
	return out, nil
}

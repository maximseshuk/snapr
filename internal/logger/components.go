package logger

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func NewSourceLogger(jobLogger zerolog.Logger, sourceType string, index int) zerolog.Logger {
	component := fmt.Sprintf("source:%s_%d", sourceType, index)
	return jobLogger.With().Str("component", component).Logger()
}

func NewStorageLogger(jobLogger zerolog.Logger, storageName string) zerolog.Logger {
	component := fmt.Sprintf("storage:%s", storageName)
	return jobLogger.With().Str("component", component).Logger()
}

func NewCompressionLogger(jobLogger zerolog.Logger, compressionType string) zerolog.Logger {
	component := fmt.Sprintf("compress:%s", compressionType)
	return jobLogger.With().Str("component", component).Logger()
}

func NewRetentionLogger(jobLogger zerolog.Logger, storageName string) zerolog.Logger {
	component := fmt.Sprintf("retention:%s", storageName)
	return jobLogger.With().Str("component", component).Logger()
}

func NewSystemLogger(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}

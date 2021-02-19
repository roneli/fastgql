package adapters

import (
	"github.com/rs/zerolog"
)

type Logger struct {
	logger zerolog.Logger
}

// NewLogger accepts a zerolog.Logger as input and returns a new custom fastgql
// logging fascade as output.
func NewZerologAdapter(logger zerolog.Logger) *Logger {
	return &Logger{
		logger: logger.With().Str("module", "fastgql").Logger(),
	}
}

func (l Logger) Trace(msg string, fields map[string]interface{}) {
	l.logger.Debug().Fields(fields).Msg(msg)
}

func (l Logger) Debug(msg string, fields map[string]interface{}) {
	l.logger.Debug().Fields(fields).Msg(msg)
}

func (l Logger) Info(msg string, fields map[string]interface{}) {
	l.logger.Info().Fields(fields).Msg(msg)
}

func (l Logger) Warn(msg string, fields map[string]interface{}) {
	l.logger.Warn().Fields(fields).Msg(msg)
}

func (l Logger) Error(msg string, fields map[string]interface{}) {
	l.logger.Error().Fields(fields).Msg(msg)
}

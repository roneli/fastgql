package adapters

import (
	"github.com/rs/zerolog"
)

type Logger struct {
	logger zerolog.Logger
}

// NewZerologAdapter accepts a zerolog.Logger as input and returns a new custom fastgql
// logging fascade as output.
func NewZerologAdapter(logger zerolog.Logger) *Logger {
	return &Logger{
		logger: logger.With().Str("module", "fastgql").Logger(),
	}
}

func (l Logger) Trace(msg string, args ...interface{}) {
	l.logger.Debug().Fields(args).Msg(msg)
}

func (l Logger) Debug(msg string, args ...interface{}) {
	l.logger.Debug().Fields(args).Msg(msg)
}

func (l Logger) Info(msg string, args ...interface{}) {
	l.logger.Info().Fields(args).Msg(msg)
}

func (l Logger) Warn(msg string, args ...interface{}) {
	l.logger.Warn().Fields(args).Msg(msg)
}

func (l Logger) Error(msg string, args ...interface{}) {
	l.logger.Error().Fields(args).Msg(msg)
}

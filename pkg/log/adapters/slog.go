package adapters

import (
	"log/slog"
)

type Slog struct {
	logger *slog.Logger
}

// NewSlogAdapter accepts a slog.Logger as input and returns a new custom fastgql
// logging facade as output.
func NewSlogAdapter(logger *slog.Logger) *Slog {
	return &Slog{
		logger: logger.With("module", "fastgql"),
	}
}

func (l Slog) Trace(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

func (l Slog) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

func (l Slog) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

func (l Slog) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

func (l Slog) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

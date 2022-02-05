package log

type (
	// Logger is an interface that can be passed to BuilderOptions.Logger.
	Logger interface {
		Trace(msg string, args ...interface{})
		Debug(msg string, args ...interface{})
		Info(msg string, args ...interface{})
		Warn(msg string, args ...interface{})
		Error(msg string, args ...interface{})
	}

	NullLogger struct{}
)

func (n NullLogger) Trace(msg string, args ...interface{}) {}
func (n NullLogger) Debug(msg string, args ...interface{}) {}
func (n NullLogger) Info(msg string, args ...interface{})  {}
func (n NullLogger) Warn(msg string, args ...interface{})  {}
func (n NullLogger) Error(msg string, args ...interface{}) {}

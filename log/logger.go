package log

type (
	// Logger is an interface that can be passed to BuilderOptions.Logger.
	Logger interface {
		Trace(msg string, fields map[string]interface{})
		Debug(msg string, fields map[string]interface{})
		Info(msg string, fields map[string]interface{})
		Warn(msg string, fields map[string]interface{})
		Error(msg string, fields map[string]interface{})
	}

	NullLogger struct{}
)

func (n NullLogger) Trace(msg string, fields map[string]interface{}) {}
func (n NullLogger) Debug(msg string, fields map[string]interface{}) {}
func (n NullLogger) Info(msg string, fields map[string]interface{})  {}
func (n NullLogger) Warn(msg string, fields map[string]interface{})  {}
func (n NullLogger) Error(msg string, fields map[string]interface{}) {}

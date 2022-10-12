package core

type Logger interface {
	Debug(string)
	Info(string)
	Warn(string)
	Error(string)
	Debugf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
}

type NilLogger struct{}

var nilLogger = &NilLogger{}

func NewNilLogger() *NilLogger {
	return nilLogger
}

func (*NilLogger) Debug(string) {
	// no-op
}

func (*NilLogger) Info(string) {
	// no-op
}

func (*NilLogger) Warn(string) {
	// no-op
}

func (*NilLogger) Error(string) {
	// no-op
}

func (*NilLogger) Debugf(template string, args ...interface{}) {
	// no-op
}

func (*NilLogger) Infof(template string, args ...interface{}) {
	// no-op
}

func (*NilLogger) Warnf(template string, args ...interface{}) {
	// no-op
}

func (*NilLogger) Errorf(template string, args ...interface{}) {
	// no-op
}

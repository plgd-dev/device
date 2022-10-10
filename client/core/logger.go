package core

import (
	"github.com/pion/logging"
)

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

type DefaultLogger logging.DefaultLeveledLogger

// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"fmt"
	"log"
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

func (*NilLogger) Debugf(string, ...interface{}) {
	// no-op
}

func (*NilLogger) Infof(string, ...interface{}) {
	// no-op
}

func (*NilLogger) Warnf(string, ...interface{}) {
	// no-op
}

func (*NilLogger) Errorf(string, ...interface{}) {
	// no-op
}

type LogLevel int

const (
	LogLevelNone LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

type StdLogger struct {
	*log.Logger
	level LogLevel
}

func NewStdLogger(logLevel LogLevel) *StdLogger {
	return &StdLogger{
		Logger: log.Default(),
		level:  logLevel,
	}
}

func (l *StdLogger) checkLevel(level LogLevel) bool {
	return l.level != LogLevelNone && l.level <= level
}

func (l *StdLogger) Debug(msg string) {
	if l.checkLevel(LogLevelDebug) {
		l.Println(msg)
	}
}

func (l *StdLogger) Info(msg string) {
	if l.checkLevel(LogLevelInfo) {
		l.Println(msg)
	}
}

func (l *StdLogger) Warn(msg string) {
	if l.checkLevel(LogLevelWarn) {
		l.Println(msg)
	}
}

func (l *StdLogger) Error(msg string) {
	if l.checkLevel(LogLevelError) {
		l.Println(msg)
	}
}

// Debugf uses fmt.Sprintf to construct and log.Printf to log a message.
func (l *StdLogger) Debugf(format string, args ...interface{}) {
	if l.checkLevel(LogLevelDebug) {
		l.Printf("%s\n", fmt.Sprintf(format, args...))
	}
}

// Infof uses fmt.Sprintf to construct and log.Printf to log a message.
func (l *StdLogger) Infof(format string, args ...interface{}) {
	if l.checkLevel(LogLevelInfo) {
		l.Printf("%s\n", fmt.Sprintf(format, args...))
	}
}

// Warnf uses fmt.Sprintf to construct and log.Printf to log a message.
func (l *StdLogger) Warnf(format string, args ...interface{}) {
	if l.checkLevel(LogLevelWarn) {
		l.Printf("%s\n", fmt.Sprintf(format, args...))
	}
}

// Errorf uses fmt.Errorf to construct and log.Printf to log a message.
func (l *StdLogger) Errorf(format string, args ...interface{}) {
	if l.checkLevel(LogLevelError) {
		l.Printf("%s\n", fmt.Errorf(format, args...))
	}
}

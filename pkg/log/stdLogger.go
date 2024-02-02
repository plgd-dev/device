/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package log

import (
	"bytes"
	"fmt"
	"io"
	"log"
)

type Level int8

const (
	LevelNone Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

func ParseLevel(text string) (Level, error) {
	var level Level
	err := level.UnmarshalText([]byte(text))
	return level, err
}

// String returns a lower-case ASCII representation of the log level.
func (l Level) String() string {
	switch l {
	case LevelNone:
		return "none"
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

func (l *Level) UnmarshalText(text []byte) error {
	if !l.unmarshalText(text) && !l.unmarshalText(bytes.ToLower(text)) {
		return fmt.Errorf("unrecognized level: %q", text)
	}
	return nil
}

func (l *Level) unmarshalText(text []byte) bool {
	switch string(text) {
	case "", "none":
		*l = LevelNone
	case "debug":
		*l = LevelDebug
	case "info":
		*l = LevelInfo
	case "warn":
		*l = LevelWarn
	case "error":
		*l = LevelError
	default:
		return false
	}
	return true
}

type StdLogger struct {
	std   *log.Logger
	level Level
}

func NewStdLogger(logLevel Level) *StdLogger {
	return &StdLogger{
		std:   log.Default(),
		level: logLevel,
	}
}

func (l *StdLogger) SetOutput(w io.Writer) {
	l.std.SetOutput(w)
}

func (l *StdLogger) checkLevel(level Level) bool {
	return l.level != LevelNone && l.level <= level
}

func (l *StdLogger) LogWithLevel(level Level, msg string) {
	if l.checkLevel(level) {
		l.std.Println(msg)
	}
}

func (l *StdLogger) Debug(msg string) {
	l.LogWithLevel(LevelDebug, msg)
}

func (l *StdLogger) Info(msg string) {
	l.LogWithLevel(LevelInfo, msg)
}

func (l *StdLogger) Warn(msg string) {
	l.LogWithLevel(LevelWarn, msg)
}

func (l *StdLogger) Error(msg string) {
	l.LogWithLevel(LevelError, msg)
}

// LogfWithLevel uses fmt.Errorf to construct and log.Printf to log a message.
func (l *StdLogger) LogfWithLevel(level Level, format string, args ...interface{}) {
	if l.checkLevel(level) {
		l.std.Printf("%s\n", fmt.Errorf(format, args...))
	}
}

func (l *StdLogger) Debugf(format string, args ...interface{}) {
	l.LogfWithLevel(LevelDebug, format, args...)
}

func (l *StdLogger) Infof(format string, args ...interface{}) {
	l.LogfWithLevel(LevelInfo, format, args...)
}

func (l *StdLogger) Warnf(format string, args ...interface{}) {
	l.LogfWithLevel(LevelWarn, format, args...)
}

func (l *StdLogger) Errorf(format string, args ...interface{}) {
	l.LogfWithLevel(LevelError, format, args...)
}

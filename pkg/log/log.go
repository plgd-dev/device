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

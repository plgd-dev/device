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

package log_test

import (
	"bytes"
	"testing"

	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestStdLogger(t *testing.T) {
	var b bytes.Buffer
	l := log.NewStdLogger(log.LevelNone)
	l.SetOutput(&b)
	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")
	l.Debugf("debugf")
	l.Infof("infof")
	l.Warnf("warnf")
	l.Errorf("errorf")
	require.Empty(t, b.String())

	b.Reset()
	l = log.NewStdLogger(log.LevelDebug)
	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")
	l.Debugf("debugf")
	l.Infof("infof")
	l.Warnf("warnf")
	l.Errorf("errorf")
	require.Contains(t, b.String(), "debug")
	require.Contains(t, b.String(), "info")
	require.Contains(t, b.String(), "warn")
	require.Contains(t, b.String(), "error")
	require.Contains(t, b.String(), "debugf")
	require.Contains(t, b.String(), "infof")
	require.Contains(t, b.String(), "warnf")
	require.Contains(t, b.String(), "errorf")

	b.Reset()
	l = log.NewStdLogger(log.LevelError)
	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")
	l.Debugf("debugf")
	l.Infof("infof")
	l.Warnf("warnf")
	l.Errorf("errorf")
	require.NotContains(t, b.String(), "debug")
	require.NotContains(t, b.String(), "info")
	require.NotContains(t, b.String(), "warn")
	require.Contains(t, b.String(), "error")
	require.NotContains(t, b.String(), "debugf")
	require.NotContains(t, b.String(), "infof")
	require.NotContains(t, b.String(), "warnf")
	require.Contains(t, b.String(), "errorf")
}

func TestJsonEncodeLevel(t *testing.T) {
	b, err := json.Encode(log.LevelNone)
	require.NoError(t, err)
	require.Equal(t, []byte(`"none"`), b)
	b, err = json.Encode(log.LevelDebug)
	require.NoError(t, err)
	require.Equal(t, []byte(`"debug"`), b)
	b, err = json.Encode(log.LevelInfo)
	require.NoError(t, err)
	require.Equal(t, []byte(`"info"`), b)
	b, err = json.Encode(log.LevelWarn)
	require.NoError(t, err)
	require.Equal(t, []byte(`"warn"`), b)
	b, err = json.Encode(log.LevelError)
	require.NoError(t, err)
	require.Equal(t, []byte(`"error"`), b)
}

func TestJsonDecodeLevel(t *testing.T) {
	var v log.Level
	err := json.Decode([]byte(`""`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelNone, v)
	err = json.Decode([]byte(`"none"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelNone, v)
	err = json.Decode([]byte(`"NONE"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelNone, v)
	err = json.Decode([]byte(`"debug"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelDebug, v)
	err = json.Decode([]byte(`"DEBUG"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelDebug, v)
	err = json.Decode([]byte(`"info"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelInfo, v)
	err = json.Decode([]byte(`"INFO"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelInfo, v)
	err = json.Decode([]byte(`"warn"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelWarn, v)
	err = json.Decode([]byte(`"WARN"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelWarn, v)
	err = json.Decode([]byte(`"error"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelError, v)
	err = json.Decode([]byte(`"ERROR"`), &v)
	require.NoError(t, err)
	require.Equal(t, log.LevelError, v)

	err = json.Decode([]byte(`"unknown"`), &v)
	require.Error(t, err)
}

func TestParseLevel(t *testing.T) {
	lvl, err := log.ParseLevel("none")
	require.NoError(t, err)
	require.Equal(t, log.LevelNone, lvl)
	lvl, err = log.ParseLevel("error")
	require.NoError(t, err)
	require.Equal(t, log.LevelError, lvl)

	_, err = log.ParseLevel("unknown")
	require.Error(t, err)
}

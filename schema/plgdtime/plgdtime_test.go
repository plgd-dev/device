// ************************************************************************
// Copyright (C) 2023 plgd.dev, s.r.o.
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

package plgdtime_test

import (
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/stretchr/testify/require"
)

func TestPlgdTimeGetTime(t *testing.T) {
	type args struct {
		time string
	}
	tests := []struct {
		name     string
		args     args
		wantTime time.Time
		wantErr  bool
	}{
		{
			name:    "Empty string",
			args:    args{time: ""},
			wantErr: true,
		},
		{
			name:    "Invalid string",
			args:    args{time: "This is not a time string"},
			wantErr: true,
		},
		{
			name:     "Valid string",
			args:     args{time: "2023-01-15T15:04:05.000000000Z"},
			wantTime: time.Date(2023, time.January, 15, 15, 4, 5, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt := plgdtime.PlgdTime{
				Time: tt.args.time,
			}
			actualTime, err := pt.GetTime()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantTime, actualTime)
		})
	}
}

func TestPlgdTimeGetLastSyncedTime(t *testing.T) {
	type args struct {
		time string
	}
	tests := []struct {
		name     string
		args     args
		wantTime time.Time
		wantErr  bool
	}{
		{
			name:    "Invalid string",
			args:    args{time: "This is not a time string"},
			wantErr: true,
		},
		{
			name:     "Empty string",
			args:     args{time: ""},
			wantTime: time.Time{},
		},
		{
			name:     "Valid string",
			args:     args{time: "2023-01-15T15:04:05.000000000Z"},
			wantTime: time.Date(2023, time.January, 15, 15, 4, 5, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt := plgdtime.PlgdTime{
				LastSyncedTime: tt.args.time,
			}
			lastSyncedTime, err := pt.GetLastSyncedTime()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantTime, lastSyncedTime)
		})
	}
}

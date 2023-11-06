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

package platform_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/stretchr/testify/require"
)

func TestPlatformGetVersion(t *testing.T) {
	type args struct {
		version uint32
	}
	tests := []struct {
		name       string
		args       args
		wantMajor  uint8
		wantMinor  uint8
		wantPatch  uint8
		wantBugfix uint8
	}{
		{
			name:       "Valid Version",
			args:       args{version: 0x01020304},
			wantMajor:  0x01,
			wantMinor:  0x02,
			wantPatch:  0x03,
			wantBugfix: 0x04,
		},
		{
			name:       "Zero Version",
			args:       args{version: 0},
			wantMajor:  0,
			wantMinor:  0,
			wantPatch:  0,
			wantBugfix: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := platform.Platform{Version: tt.args.version}
			major, minor, patch, bugfix := p.GetVersion()
			require.Equal(t, tt.wantMajor, major)
			require.Equal(t, tt.wantMinor, minor)
			require.Equal(t, tt.wantPatch, patch)
			require.Equal(t, tt.wantBugfix, bugfix)
		})
	}
}

func TestPlatformGetVersionString(t *testing.T) {
	type args struct {
		version uint32
	}
	tests := []struct {
		name     string
		args     args
		expected string
	}{
		{
			name:     "Valid Version",
			args:     args{version: 0x0202050B},
			expected: "2.2.5.11",
		},
		{
			name:     "Zero Version",
			args:     args{version: 0},
			expected: "0.0.0.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := platform.Platform{Version: test.args.version}
			require.Equal(t, test.expected, p.GetVersionString())
		})
	}
}

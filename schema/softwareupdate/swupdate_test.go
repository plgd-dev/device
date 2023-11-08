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

package softwareupdate_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/stretchr/testify/require"
)

func TestSoftwareUpdateGetUpdateResultNilReceiver(t *testing.T) {
	var sw *softwareupdate.SoftwareUpdate
	require.Equal(t, -1, sw.GetUpdateResult())
}

func TestSoftwareUpdateGetUpdateResult(t *testing.T) {
	type args struct {
		result *int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Nil",
			args: args{result: nil},
			want: -1,
		},
		{
			name: "Zero",
			args: args{result: new(int)},
			want: 0,
		},
		{
			name: "Valid",
			args: args{result: func() *int {
				i := 42
				return &i
			}()},
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw := &softwareupdate.SoftwareUpdate{
				UpdateResult: tt.args.result,
			}
			require.Equal(t, tt.want, sw.GetUpdateResult())
		})
	}
}

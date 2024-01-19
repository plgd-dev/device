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

package resources_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/stretchr/testify/require"
)

func TestToUUID(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name string
		args args
		want uuid.UUID
	}{
		{
			name: "valid",
			args: args{
				id: "00000000-0000-0000-0000-000000000001",
			},
			want: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
		{
			name: "invalid",
			args: args{
				id: "00000000-0000-0000-0000-0000000000",
			},
			want: uuid.NewSHA1(uuid.NameSpaceURL, []byte("00000000-0000-0000-0000-0000000000")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resources.ToUUID(tt.args.id)
			require.Equal(t, tt.want, got)
		})
	}
}

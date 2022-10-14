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

package acl_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/stretchr/testify/require"
)

func TestPermissionString(t *testing.T) {
	tests := []struct {
		name string
		s    acl.Permission
		want string
	}{
		{
			name: "Empty",
			s:    0,
			want: "",
		},
		{
			name: "Unknown",
			s:    acl.Permission_NOTIFY << 1, // double of the last acl.Permission value
			want: "unknown(32)",
		},
		{
			name: "Single",
			s:    acl.Permission_CREATE,
			want: "CREATE",
		},
		{
			name: "All",
			s:    acl.AllPermissions,
			want: "CREATE|READ|WRITE|DELETE|NOTIFY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.String()
			require.Equal(t, tt.want, got)
		})
	}
}

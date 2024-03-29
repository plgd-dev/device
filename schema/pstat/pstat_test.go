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

package pstat_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/pstat"
	"github.com/stretchr/testify/require"
)

func TestOperationalStateString(t *testing.T) {
	states := map[pstat.OperationalState]string{
		pstat.OperationalState_RESET:  "RESET",
		pstat.OperationalState_RFOTM:  "RFOTM",
		pstat.OperationalState_RFPRO:  "RFPRO",
		pstat.OperationalState_RFNOP:  "RFNOP",
		pstat.OperationalState_SRESET: "SRESET",
	}

	for k, v := range states {
		require.Equal(t, v, k.String())
	}

	unknown := pstat.OperationalState_SRESET + 1
	require.Equal(t, "unknown 5", unknown.String())
}

func TestOperationalModeString(t *testing.T) {
	tests := []struct {
		name string
		s    pstat.OperationalMode
		want string
	}{
		{
			name: "Empty",
			s:    0,
			want: "",
		},
		{
			name: "Unknown",
			s:    pstat.OperationalMode_CLIENT_DIRECTED << 1, // double of the last pstat.OperationalMode value
			want: "unknown(8)",
		},
		{
			name: "Single",
			s:    pstat.OperationalMode_SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES,
			want: "SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES",
		},
		{
			name: "All",
			s: pstat.OperationalMode_SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES | pstat.OperationalMode_SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE |
				pstat.OperationalMode_CLIENT_DIRECTED,
			want: "SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES|SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE|CLIENT_DIRECTED",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.String()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestProvisioningModeString(t *testing.T) {
	tests := []struct {
		name string
		s    pstat.ProvisioningMode
		want string
	}{
		{
			name: "Empty",
			s:    0,
			want: "",
		},
		{
			name: "Unknown",
			s:    pstat.ProvisioningMode_INIT_SEC_SOFT_UPDATE << 1, // double of the last pstat.ProvisioningMode value
			want: "unknown(256)",
		},
		{
			name: "Single",
			s:    pstat.ProvisioningMode_INIT_SOFT_VER_VALIDATION,
			want: "INIT_SOFT_VER_VALIDATION",
		},
		{
			name: "All",
			s:    pstat.ProvisioningMode_INIT_SOFT_VER_VALIDATION | pstat.ProvisioningMode_INIT_SEC_SOFT_UPDATE,
			want: "INIT_SOFT_VER_VALIDATION|INIT_SEC_SOFT_UPDATE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.String()
			require.Equal(t, tt.want, got)
		})
	}
}

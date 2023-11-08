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

package credential_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/stretchr/testify/require"
)

func testCredentialData(t *testing.T, checkData func(data interface{}, expected []byte)) {
	type args struct {
		data interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Nil",
			args: args{data: nil},
			want: nil,
		},
		{
			name: "String",
			args: args{data: "test"},
			want: []byte("test"),
		},
		{
			name: "Bytes",
			args: args{data: []byte("test")},
			want: []byte("test"),
		},
		{
			name: "Invalid",
			args: args{data: 42},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkData(tt.args.data, tt.want)
		})
	}
}

func TestCredentialOptionalData(t *testing.T) {
	testCredentialData(t, func(data interface{}, expected []byte) {
		c := credential.CredentialOptionalData{
			DataInternal: data,
		}
		require.Equal(t, expected, c.Data())
	})
}

func TestCredentialPrivateData(t *testing.T) {
	testCredentialData(t, func(data interface{}, expected []byte) {
		c := credential.CredentialPrivateData{
			DataInternal: data,
		}
		require.Equal(t, expected, c.Data())
	})
}

func TestCredentialPublicData(t *testing.T) {
	testCredentialData(t, func(data interface{}, expected []byte) {
		c := credential.CredentialPublicData{
			DataInternal: data,
		}
		require.Equal(t, expected, c.Data())
	})
}

func TestCredentialTypeString(t *testing.T) {
	tests := []struct {
		name string
		s    credential.CredentialType
		want string
	}{
		{
			name: "Empty",
			s:    0,
			want: "EMPTY",
		},
		{
			name: "Unknown",
			s:    credential.CredentialType_ASYMMETRIC_ENCRYPTION_KEY << 1, // double of the last credential.CredentialType value
			want: "unknown(64)",
		},
		{
			name: "Single",
			s:    credential.CredentialType_SYMMETRIC_PAIR_WISE,
			want: "SYMMETRIC_PAIR_WISE",
		},
		{
			name: "All",
			s: credential.CredentialType_SYMMETRIC_PAIR_WISE | credential.CredentialType_SYMMETRIC_GROUP |
				credential.CredentialType_ASYMMETRIC_SIGNING | credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE |
				credential.CredentialType_PIN_OR_PASSWORD | credential.CredentialType_ASYMMETRIC_ENCRYPTION_KEY,
			want: "SYMMETRIC_PAIR_WISE|SYMMETRIC_GROUP|ASYMMETRIC_SIGNING|ASYMMETRIC_SIGNING_WITH_CERTIFICATE|PIN_OR_PASSWORD|ASYMMETRIC_ENCRYPTION_KEY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.String()
			require.Equal(t, tt.want, got)
		})
	}
}

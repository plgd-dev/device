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

package csr_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/stretchr/testify/require"
)

func TestCSR(t *testing.T) {
	type args struct {
		request interface{}
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Nil",
			args: args{request: nil},
			want: nil,
		},
		{
			name: "String",
			args: args{request: "test"},
			want: []byte("test"),
		},
		{
			name: "Bytes",
			args: args{request: []byte("test")},
			want: []byte("test"),
		},
		{
			name: "Invalid",
			args: args{request: 42},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := csr.CertificateSigningRequestResponse{
				CertificateSigningRequest: tt.args.request,
			}
			require.Equal(t, tt.want, c.CSR())
		})
	}
}

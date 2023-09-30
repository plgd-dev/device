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

package coap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObservedResourceEncodeETagsForIncrementChanged(t *testing.T) {
	tests := []struct {
		name  string
		etags [][]byte
		want  []string
	}{
		{
			name:  "empty",
			etags: nil,
		},
		{
			name:  "not-nil",
			etags: [][]byte{},
		},
		{
			name: "one-etag",
			etags: [][]byte{
				[]byte("01234567"),
			},
			want: []string{
				prefixQueryIncChanges + "MDEyMzQ1Njc",
			},
		},
		{
			name: "two-etags",
			etags: [][]byte{
				[]byte("1"),
				[]byte("2"),
			},
			want: []string{
				prefixQueryIncChanges + "MQ,Mg",
			},
		},
		{
			name: "two-etags-invalid-etag",
			etags: [][]byte{
				[]byte("1"),
				[]byte("2"),
				[]byte("invalid-etag-is-ignored"),
			},
			want: []string{
				prefixQueryIncChanges + "MQ,Mg",
			},
		},
		{
			name: "multiple-etags",
			etags: [][]byte{
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"),
				[]byte("01234567"), // 21
			},
			want: []string{
				prefixQueryIncChanges + "MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc,MDEyMzQ1Njc",
				prefixQueryIncChanges + "MDEyMzQ1Njc",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeETagsForIncrementalChanges(tt.etags)
			for _, g := range got {
				require.Less(t, len(g), maxURIQueryLen)
			}
			require.Equal(t, tt.want, got)
		})
	}
}

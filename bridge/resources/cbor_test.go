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

	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/stretchr/testify/require"
)

func TestMergeCBORStructs(t *testing.T) {
	type args struct {
		structs []interface{}
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			name: "valid",
			args: args{
				structs: []interface{}{
					map[interface{}]interface{}{"key1": "value1"},
					map[interface{}]interface{}{"key2": "value2"},
				},
			},
			want: map[interface{}]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "merging nil structs",
			args: args{
				structs: []interface{}{
					nil,
					nil,
				},
			},
			want: nil,
		},
		{
			name: "invalid CBOR encoding",
			args: args{
				structs: []interface{}{
					map[interface{}]interface{}{"key1": "value1"},
					"invalid CBOR",
				},
			},
			want: map[interface{}]interface{}{
				"key1": "value1",
			},
		},
		{
			name: "merging struct with empty CBOR encoding",
			args: args{
				structs: []interface{}{
					map[interface{}]interface{}{"key1": "value1"},
					map[interface{}]interface{}{},
				},
			},
			want: map[interface{}]interface{}{
				"key1": "value1",
			},
		},
		{
			name: "merging multiple structs",
			args: args{
				structs: []interface{}{
					map[interface{}]interface{}{"key1": "value1"},
					map[interface{}]interface{}{"key2": "value2"},
					map[interface{}]interface{}{"key3": "value3"},
					map[interface{}]interface{}{"key4": "value4"},
				},
			},
			want: map[interface{}]interface{}{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
				"key4": "value4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resources.MergeCBORStructs(tt.args.structs...)
			if tt.want != nil {
				require.Equal(t, tt.want, got)
				return
			}
			require.Nil(t, got)
		})
	}
}

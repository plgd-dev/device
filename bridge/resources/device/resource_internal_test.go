package device

import (
	"testing"

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
			got := mergeCBORStructs(tt.args.structs...)
			if tt.want != nil {
				require.Equal(t, tt.want, got)
				return
			}
			require.Nil(t, got)
		})
	}
}

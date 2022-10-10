package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToCBOR(t *testing.T) {
	type args struct {
		json string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				json: "",
			},
			wantErr: true,
		},
		{
			name: "empty object",
			args: args{
				json: "{}",
			},
			want: []byte{0xa0},
		},
		{
			name: "empty array",
			args: args{
				json: "[]",
			},
			want: []byte{0x80},
		},
		{
			name: "empty string",
			args: args{
				json: `""`,
			},
			want: []byte{0x60},
		},
		{
			name: "string",
			args: args{
				json: `"abc"`,
			},
			want: []byte{0x63, 0x61, 0x62, 0x63},
		},
		{
			name: "string with escape",
			args: args{
				json: `"a\"b"`,
			},
			want: []byte{0x63, 0x61, 0x22, 0x62},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToCBOR(tt.args.json)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestEncode(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				v: nil,
			},
			want: []byte("null"),
		},
		{
			name: "empty object",
			args: args{
				v: struct{}{},
			},
			want: []byte("{}"),
		},
		{
			name: "empty array",
			args: args{
				v: []struct{}{},
			},
			want: []byte("[]"),
		},
		{
			name: "empty string",
			args: args{
				v: "",
			},
			want: []byte(`""`),
		},
		{
			name: "string",
			args: args{
				v: "abc",
			},
			want: []byte(`"abc"`),
		},
		{
			name: "string with escape",
			args: args{
				v: "a\"b",
			},
			want: []byte(`"a\"b"`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.args.v)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

package cbor

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	type args struct {
		cbor []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				cbor: nil,
			},
			wantErr: true,
		},
		{
			name: "empty object",
			args: args{
				cbor: []byte{0xa0},
			},
			want: "{}",
		},
		{
			name: "empty array",
			args: args{
				cbor: []byte{0x80},
			},
			want: "[]",
		},
		{
			name: "empty string",
			args: args{
				cbor: []byte{0x60},
			},
			want: `""`,
		},
		{
			name: "string",
			args: args{
				cbor: []byte{0x63, 0x61, 0x62, 0x63},
			},
			want: `"abc"`,
		},
		{
			name: "string with escape",
			args: args{
				cbor: []byte{0x63, 0x61, 0x22, 0x62},
			},
			want: `"a\"b"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON(tt.args.cbor)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWriteTo(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				v: nil,
			},
			wantW: "\xf6",
		},
		{
			name: "empty object",
			args: args{
				v: map[string]interface{}{},
			},
			wantW: "\xa0",
		},
		{
			name: "empty array",
			args: args{
				v: []interface{}{},
			},
			wantW: "\x80",
		},
		{
			name: "empty string",
			args: args{
				v: "",
			},
			wantW: "\x60",
		},
		{
			name: "string",
			args: args{
				v: "abc",
			},
			wantW: "\x63\x61\x62\x63",
		},
		{
			name: "string with escape",
			args: args{
				v: "a\"b",
			},
			wantW: "\x63\x61\x22\x62",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := WriteTo(w, tt.args.v)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantW, w.String())
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
			want: []byte{0xf6},
		},
		{
			name: "empty object",
			args: args{
				v: map[string]interface{}{},
			},
			want: []byte{0xa0},
		},
		{
			name: "empty array",
			args: args{
				v: []interface{}{},
			},
			want: []byte{0x80},
		},
		{
			name: "empty string",
			args: args{
				v: "",
			},
			want: []byte{0x60},
		},
		{
			name: "string",
			args: args{
				v: "abc",
			},
			want: []byte{0x63, 0x61, 0x62, 0x63},
		},
		{
			name: "string with escape",
			args: args{
				v: "a\"b",
			},
			want: []byte{0x63, 0x61, 0x22, 0x62},
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

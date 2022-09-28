package ocf

import (
	"bytes"
	"context"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/stretchr/testify/require"
)

func TestVNDOCFCBORCodecDecode(t *testing.T) {
	type args struct {
		m *pool.Message
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    interface{}
	}{
		{
			name: "empty",
			args: args{
				m: pool.NewMessage(context.TODO()),
			},
			wantErr: true,
		},
		{
			name: "empty object",
			args: args{
				m: func() *pool.Message {
					m := pool.NewMessage(context.TODO())
					m.SetContentFormat(message.AppOcfCbor)
					m.SetBody(bytes.NewReader([]byte{0xa0}))
					m.SetCode(codes.Content)
					return m
				}(),
			},
			want: map[interface{}]interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := VNDOCFCBORCodec{}
			var got interface{}
			err := v.Decode(tt.args.m, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRawCodecDecode(t *testing.T) {
	type fields struct {
		MediaType message.MediaType
	}
	type args struct {
		m *pool.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    []byte
	}{
		{
			name: "empty",
			args: args{
				m: pool.NewMessage(context.TODO()),
			},
			want: nil,
		},
		{
			name: "unknown media type",
			args: args{
				m: func() *pool.Message {
					m := pool.NewMessage(context.TODO())
					m.SetContentFormat(message.AppOcfCbor)
					m.SetBody(bytes.NewReader([]byte{0xa0}))
					m.SetCode(codes.Content)
					return m
				}(),
			},
			wantErr: true,
		},
		{
			name: "object",
			fields: fields{
				MediaType: message.AppOcfCbor,
			},
			args: args{
				m: func() *pool.Message {
					m := pool.NewMessage(context.TODO())
					m.SetContentFormat(message.AppOcfCbor)
					m.SetBody(bytes.NewReader([]byte{0xa1, 0x63, 0x6b, 0x65, 0x79, 0x61, 0x76, 0x61, 0x6c, 0x75, 0x65}))
					m.SetCode(codes.Content)
					return m
				}(),
			},
			want: []byte{0xa1, 0x63, 0x6b, 0x65, 0x79, 0x61, 0x76, 0x61, 0x6c, 0x75, 0x65},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := RawCodec{
				EncodeMediaType:  tt.fields.MediaType,
				DecodeMediaTypes: []message.MediaType{tt.fields.MediaType},
			}
			var got []byte
			err := c.Decode(tt.args.m, &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDump(t *testing.T) {
	type args struct {
		message *pool.Message
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get request",
			args: args{
				message: func() *pool.Message {
					m := pool.NewMessage(context.TODO())
					m.SetContentFormat(message.AppOcfCbor)
					err := m.SetPath("/oic/res")
					require.NoError(t, err)
					m.SetBody(bytes.NewReader([]byte{0xa1, 0x63, 0x6b, 0x65, 0x79, 0x61, 0x76, 0x61, 0x6c, 0x75, 0x65}))
					m.SetCode(codes.GET)
					return m
				}(),
			},
			want: "Format: application/vnd.ocf+cbor\n\nContent: {\"key\":\"v\"}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Dump(tt.args.message)
			require.Equal(t, tt.want, got)
		})
	}
}

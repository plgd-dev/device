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

package ocf

import (
	"bytes"
	"context"
	"io"
	"strings"
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
			name: "invalid cbor format",
			args: args{
				m: func() *pool.Message {
					m := pool.NewMessage(context.TODO())
					m.SetContentFormat(message.TextPlain)
					return m
				}(),
			},
			wantErr: true,
		},
		{
			name: "empty body",
			args: args{
				m: func() *pool.Message {
					m := pool.NewMessage(context.TODO())
					m.SetContentFormat(message.AppCBOR)
					return m
				}(),
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
			name: "empty body",
			args: args{
				m: func() *pool.Message {
					m := pool.NewMessage(context.TODO())
					m.SetContentFormat(message.AppCBOR)
					return m
				}(),
			},
			wantErr: true,
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
			require.Equal(t, tt.fields.MediaType, c.ContentFormat())
			require.ElementsMatch(t, []message.MediaType{tt.fields.MediaType}, c.DecodeContentFormat())
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDump(t *testing.T) {
	type args struct {
		cf   message.MediaType
		body io.ReadSeeker
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get request (AppJSON)",
			args: args{
				cf:   message.AppJSON,
				body: strings.NewReader("{\"key\":\"v\"}"),
			},
			want: "Format: application/json\n\nContent: {\"key\":\"v\"}\n",
		},
		{
			name: "get request (AppOcfCbor)",
			args: args{
				cf:   message.AppOcfCbor,
				body: bytes.NewReader([]byte{0xa1, 0x63, 0x6b, 0x65, 0x79, 0x61, 0x76, 0x61, 0x6c, 0x75, 0x65}),
			},
			want: "Format: application/vnd.ocf+cbor\n\nContent: {\"key\":\"v\"}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := pool.NewMessage(context.TODO())
			m.SetContentFormat(tt.args.cf)
			err := m.SetPath("/oic/res")
			require.NoError(t, err)
			m.SetBody(tt.args.body)
			m.SetCode(codes.GET)
			got := Dump(m)
			require.Equal(t, tt.want, got)
		})
	}
}

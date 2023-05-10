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

package schema_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/ael"
	"github.com/plgd-dev/device/v2/schema/credential"
	kitNet "github.com/plgd-dev/kit/v2/net"
	"github.com/stretchr/testify/require"
)

func TestBitMaskHas(t *testing.T) {
	b := schema.BitMask(0)
	require.False(t, b.Has(schema.Discoverable))
	require.False(t, b.Has(schema.Observable))

	b = schema.Discoverable
	require.True(t, b.Has(schema.Discoverable))
	require.False(t, b.Has(schema.Observable))

	b = schema.Discoverable | schema.Observable
	require.True(t, b.Has(schema.Discoverable))
	require.True(t, b.Has(schema.Observable))
}

func TestResourceLinkHasType(t *testing.T) {
	type fields struct {
		resourceTypes []string
	}
	type args struct {
		resourceType string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Empty",
			fields: fields{
				resourceTypes: []string{credential.ResourceType, acl.ResourceType},
			},
			args: args{
				resourceType: "",
			},
			want: false,
		},
		{
			name: "Mismatch",
			fields: fields{
				resourceTypes: []string{credential.ResourceType, acl.ResourceType},
			},
			args: args{
				resourceType: ael.ResourceType,
			},
			want: false,
		},
		{
			name: "Match",
			fields: fields{
				resourceTypes: []string{credential.ResourceType, acl.ResourceType},
			},
			args: args{
				resourceType: acl.ResourceType,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := schema.ResourceLink{
				ResourceTypes: tt.fields.resourceTypes,
			}
			require.Equal(t, tt.want, r.HasType(tt.args.resourceType))
		})
	}
}

func TestResourceLinkPatchEndpoint(t *testing.T) {
	type args struct {
		addr kitNet.Addr
	}
	tests := []struct {
		name string
		args args
		want schema.Endpoints
	}{
		{
			name: "IPv4",
			args: args{
				addr: kitNet.MakeAddr("coap", "127.0.0.1", 5683),
			},
			want: schema.Endpoints{
				schema.Endpoint{URI: "coap://127.0.0.1:5683"},
				schema.Endpoint{URI: "coap+tcp://127.0.0.1:5683"},
				schema.Endpoint{URI: "coaps+tcp://127.0.0.1:5684"},
			},
		},
		{
			name: "IPv6",
			args: args{
				addr: kitNet.MakeAddr("coap", "fe80::1", 5683),
			},
			want: schema.Endpoints{
				schema.Endpoint{URI: "coap://[fe80::1]:5683"},
				schema.Endpoint{URI: "coap+tcp://[fe80::1]:5683"},
				schema.Endpoint{URI: "coaps+tcp://[fe80::1]:5684"},
			},
		},
	}
	for _, tt := range tests {
		port1 := uint16(5683)
		port2 := uint16(5684)
		t.Run(tt.name, func(t *testing.T) {
			r := schema.ResourceLink{
				Policy: &schema.Policy{
					UDPPort:    &port1,
					TCPPort:    &port1,
					TCPTLSPort: &port2,
				},
			}
			got := r.PatchEndpoint(tt.args.addr, nil)
			require.Len(t, got.Endpoints, len(tt.want))
			if len(tt.want) > 0 {
				got.Endpoints = got.Endpoints.Sort()
				tt.want = tt.want.Sort()
				for i := range got.Endpoints {
					require.Equal(t, got.Endpoints[i], tt.want[i])
				}
			}
		})
	}
}

func TestResourceLinkPatchEndpointLinkLocal(t *testing.T) {
	type fields struct {
		Endpoints schema.Endpoints
	}
	type args struct {
		addr kitNet.Addr
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   schema.Endpoints
	}{
		{
			name: "IPv4",
			fields: fields{
				Endpoints: schema.Endpoints{
					schema.Endpoint{
						URI: "coap://169.254.0.1:5683",
					},
				},
			},
			args: args{
				addr: kitNet.MakeAddr("coap", "169.254.0.1", 5683),
			},
			want: schema.Endpoints{
				schema.Endpoint{URI: "coap://169.254.0.1:5683"},
			},
		},
		{
			name: "IPv6",
			fields: fields{
				Endpoints: schema.Endpoints{
					schema.Endpoint{
						URI: "coap://[fe80::1]:5683",
					},
				},
			},
			args: args{
				addr: kitNet.MakeAddr("coap", "fe80::1%eth0", 0),
			},
			want: schema.Endpoints{
				schema.Endpoint{URI: "coap://[fe80::1%eth0]:5683"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := schema.ResourceLink{
				Endpoints: tt.fields.Endpoints,
			}
			got := r.PatchEndpoint(tt.args.addr, nil)
			require.Len(t, got.Endpoints, len(tt.want))
			if len(tt.want) > 0 {
				got.Endpoints = got.Endpoints.Sort()
				tt.want = tt.want.Sort()
				for i := range got.Endpoints {
					require.Equal(t, got.Endpoints[i], tt.want[i])
				}
			}
		})
	}
}

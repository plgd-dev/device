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

func TestEndpointGetAddress(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    kitNet.Addr
		wantErr bool
	}{
		{
			name: "Invalid",
			args: args{
				uri: "invalid-uri",
			},
			wantErr: true,
		},
		{
			name: "Valid",
			args: args{
				uri: "tcp://example.com:12345",
			},
			want: kitNet.MakeAddr("tcp", "example.com", 12345),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := schema.Endpoint{
				URI: tt.args.uri,
			}
			addr, err := ep.GetAddr()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, addr)
		})
	}
}

func TestEndpointsGetAddrInvalid(t *testing.T) {
	endpoints := schema.Endpoints{}
	_, err := endpoints.GetAddr(schema.TCPScheme)
	require.Error(t, err)

	endpoints = schema.Endpoints{
		schema.Endpoint{URI: "not-valid-uri", Priority: 2},
		schema.Endpoint{URI: "coaps+tcp://exampleTLS.com:123", Priority: 3},
		schema.Endpoint{URI: "coaps://exampleDTLS.com:789", Priority: 1},
	}
	_, err = endpoints.GetAddr(schema.TCPScheme)
	require.Error(t, err)
}

func TestEndpointsGetAddr(t *testing.T) {
	endpoints := schema.Endpoints{
		schema.Endpoint{URI: "coaps+tcp://exampleTLS.com:123", Priority: 3},
		schema.Endpoint{URI: "coap+tcp://exampleTCP.com:456", Priority: 2},
		schema.Endpoint{URI: "coaps://exampleDTLS.com:789", Priority: 1},
	}

	_, err := endpoints.GetAddr(schema.UDPScheme)
	require.Error(t, err)
	addr, err := endpoints.GetAddr(schema.TCPSecureScheme)
	require.NoError(t, err)
	require.Equal(t, kitNet.MakeAddr(string(schema.TCPSecureScheme), "exampleTLS.com", 123), addr)
	addr, err = endpoints.GetAddr(schema.TCPScheme)
	require.NoError(t, err)
	require.Equal(t, kitNet.MakeAddr(string(schema.TCPScheme), "exampleTCP.com", 456), addr)
	addr, err = endpoints.GetAddr(schema.UDPSecureScheme)
	require.NoError(t, err)
	require.Equal(t, kitNet.MakeAddr(string(schema.UDPSecureScheme), "exampleDTLS.com", 789), addr)
}

func TestResourceLinkGetSecureEndpoints(t *testing.T) {
	eps := schema.Endpoints{}
	require.Len(t, eps.FilterSecureEndpoints(), 0)

	rls := schema.ResourceLink{
		Endpoints: schema.Endpoints{
			schema.Endpoint{URI: "coap://example.com:5683"},
			schema.Endpoint{URI: "coap+tcp://example.com:5683"},
			schema.Endpoint{URI: "coaps://example.com:5684"},
			schema.Endpoint{URI: "coaps+tcp://example.com:5684"},
		},
	}

	seps := rls.GetSecureEndpoints()
	require.Len(t, seps, 2)
	require.Equal(t, "coaps://example.com:5684", seps[0].URI)
	require.Equal(t, "coaps+tcp://example.com:5684", seps[1].URI)
}

func TestResourceLinkGetUnsecureEndpoints(t *testing.T) {
	eps := schema.Endpoints{}
	require.Len(t, eps.FilterUnsecureEndpoints(), 0)

	rls := schema.ResourceLink{
		Endpoints: schema.Endpoints{
			schema.Endpoint{URI: "coap://example.com:5683"},
			schema.Endpoint{URI: "coap+tcp://example.com:5683"},
			schema.Endpoint{URI: "coaps://example.com:5684"},
			schema.Endpoint{URI: "coaps+tcp://example.com:5684"},
		},
	}

	useps := rls.GetUnsecureEndpoints()
	require.Len(t, useps, 2)
	require.Equal(t, "coap://example.com:5683", useps[0].URI)
	require.Equal(t, "coap+tcp://example.com:5683", useps[1].URI)
}

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
		addr    kitNet.Addr
		secured bool
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
			name: "IPv4-secure",
			args: args{
				addr:    kitNet.MakeAddr("coap", "127.0.0.1", 5683),
				secured: true,
			},
			want: schema.Endpoints{
				schema.Endpoint{URI: "coaps://127.0.0.1:5683"},
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
		{
			name: "IPv6-secure",
			args: args{
				addr:    kitNet.MakeAddr("coap", "fe80::1", 5683),
				secured: true,
			},
			want: schema.Endpoints{
				schema.Endpoint{URI: "coaps://[fe80::1]:5683"},
				schema.Endpoint{URI: "coap+tcp://[fe80::1]:5683"},
				schema.Endpoint{URI: "coaps+tcp://[fe80::1]:5684"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := schema.ResourceLink{
				Policy: &schema.Policy{
					UDPPort:    5683,
					TCPPort:    5683,
					TCPTLSPort: 5684,
					Secured:    tt.args.secured,
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

func TestResourceLinkGetTCPAddr(t *testing.T) {
	resourceLink := schema.ResourceLink{
		Endpoints: schema.Endpoints{
			schema.Endpoint{URI: "coap://exampleUDP.com:78", Priority: 4},
			schema.Endpoint{URI: "coaps+tcp://exampleTLS.com:12", Priority: 1},
			schema.Endpoint{URI: "coap+tcp://exampleTCP1.com:34", Priority: 2},
			schema.Endpoint{URI: "coap+tcp://exampleTCP2.com:56", Priority: 3},
		},
	}

	addr, err := resourceLink.GetTCPAddr()
	require.NoError(t, err)
	require.Equal(t, kitNet.MakeAddr("coap+tcp", "exampleTCP1.com", 34), addr)
}

func TestResourceLinkGetTCPSecureAddr(t *testing.T) {
	resourceLink := schema.ResourceLink{
		Endpoints: schema.Endpoints{
			schema.Endpoint{URI: "coap+tcp://exampleTCP.com:12", Priority: 1},
			schema.Endpoint{URI: "coaps+tcp://exampleTLS1.com:34", Priority: 2},
			schema.Endpoint{URI: "coaps+tcp://exampleTLS2.com:56", Priority: 3},
			schema.Endpoint{URI: "coap://exampleUDP.com:78", Priority: 4},
		},
	}

	addr, err := resourceLink.GetTCPSecureAddr()
	require.NoError(t, err)
	require.Equal(t, kitNet.MakeAddr("coaps+tcp", "exampleTLS1.com", 34), addr)
}

func TestGetUDPAddr(t *testing.T) {
	resourceLink := schema.ResourceLink{
		Endpoints: schema.Endpoints{
			schema.Endpoint{URI: "coap://exampleUDP.com:12", Priority: 1},
			schema.Endpoint{URI: "coaps+tcp://exampleTLS1.com:34", Priority: 2},
			schema.Endpoint{URI: "coap+tcp://exampleTCP.com:56", Priority: 3},
			schema.Endpoint{URI: "coaps+tcp://exampleTLS2.com:78", Priority: 4},
		},
	}

	addr, err := resourceLink.GetUDPAddr()
	require.NoError(t, err)
	require.Equal(t, kitNet.MakeAddr("coap", "exampleUDP.com", 12), addr)
}

func TestGetUDPSecureAddr(t *testing.T) {
	resourceLink := schema.ResourceLink{
		Endpoints: schema.Endpoints{
			schema.Endpoint{URI: "coap+tcp://exampleTCP.com:34", Priority: 2},
			schema.Endpoint{URI: "coaps+tcp://exampleTLS.com:56", Priority: 3},
			schema.Endpoint{URI: "coap://exampleUDP.com:78", Priority: 4},
			schema.Endpoint{URI: "coaps://exampleDTLS.com:12", Priority: 1},
		},
	}

	addr, err := resourceLink.GetUDPSecureAddr()
	require.NoError(t, err)
	require.Equal(t, kitNet.MakeAddr("coaps", "exampleDTLS.com", 12), addr)
}

func TestResourceLinkGetDeviceID(t *testing.T) {
	rl := schema.ResourceLink{}
	require.Equal(t, "", rl.GetDeviceID())

	rl = schema.ResourceLink{
		DeviceID: "device-id",
	}
	require.Equal(t, "device-id", rl.GetDeviceID())

	rl = schema.ResourceLink{
		Anchor: "ocf://device-id",
	}
	require.Equal(t, "device-id", rl.GetDeviceID())
}

func TestResourceLinksEndpointsSort(t *testing.T) {
	endpoints := schema.Endpoints{}
	sortedEndpoints := endpoints.Sort()
	require.Equal(t, 0, len(sortedEndpoints))

	endpoints = schema.Endpoints{
		schema.Endpoint{URI: "tcp://example.com:12345", Priority: 2},
		schema.Endpoint{URI: "udp://example.com:54321", Priority: 1},
		schema.Endpoint{URI: "coap://example.com:5683", Priority: 3},
	}

	sortedEndpoints = endpoints.Sort()
	require.Equal(t, 3, len(sortedEndpoints))
	require.Equal(t, "udp://example.com:54321", sortedEndpoints[0].URI)
	require.Equal(t, "tcp://example.com:12345", sortedEndpoints[1].URI)
	require.Equal(t, "coap://example.com:5683", sortedEndpoints[2].URI)
}

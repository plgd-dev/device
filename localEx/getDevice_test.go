package localEx_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/go-ocf/sdk/localEx"
	"github.com/go-ocf/sdk/schema/cloud"

	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

type sortResourcesByHref []schema.ResourceLink

func (a sortResourcesByHref) Len() int      { return len(a) }
func (a sortResourcesByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortResourcesByHref) Less(i, j int) bool {
	return a[i].Href < a[j].Href
}

func sortResources(s []schema.ResourceLink) []schema.ResourceLink {
	v := sortResourcesByHref(s)
	sort.Sort(v)
	return v
}

var TestDeviceSimulator = localEx.DeviceDetails{
	ID: test.TestDeviceID,
	Device: schema.Device{
		ID:   test.TestDeviceID,
		Name: test.TestDeviceName,
	},
	CloudConfiguration: &cloud.Configuration{
		ResourceTypes:      cloud.ConfigurationResourceTypes,
		Interfaces:         []string{"oic.if.rw", "oic.if.baseline"},
		ProvisioningStatus: cloud.ProvisioningStatus_UNINITIALIZED,
		CloudID:            "00000000-0000-0000-0000-000000000000",
		URL:                "coaps+tcp://127.0.0.1",
		LastErrorCode:      2,
	},
	Resources: sortResources(append(test.TestDevsimResources, test.TestDevsimPrivateResources...)),
}

var TestSecureDeviceSimulator = localEx.DeviceDetails{
	ID: test.TestSecureDeviceID,
	Device: schema.Device{
		ID:   test.TestSecureDeviceID,
		Name: test.TestSecureDeviceName,
	},
	IsSecured: true,
	Ownership: &schema.Doxm{
		ResourceOwner:                 "00000000-0000-0000-0000-000000000000",
		SupportedOwnerTransferMethods: []schema.OwnerTransferMethod{schema.JustWorks, schema.ManufacturerCertificate},
		DeviceOwner:                   "00000000-0000-0000-0000-000000000000",
		DeviceID:                      test.TestSecureDeviceID,
		SupportedCredentialTypes:      schema.CredentialType(schema.CredentialType_SYMMETRIC_PAIR_WISE | schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE),
		SelectedOwnerTransferMethod:   schema.JustWorks,
		Interfaces:                    []string{"oic.if.baseline"},
		ResourceTypes:                 []string{"oic.r.doxm"},
	},
	Resources: sortResources(append(append(test.TestDevsimResources, test.TestDevsimPrivateResources...), test.TestDevsimSecResources...)),
}

func cleanUpResources(s []schema.ResourceLink) []schema.ResourceLink {
	a := make([]schema.ResourceLink, 0, len(s))
	for _, l := range s {
		l.Endpoints = nil
		l.Policy = schema.Policy{}
		l.Anchor = ""
		a = append(a, l)
	}
	return a
}

func TestClient_GetDevice(t *testing.T) {
	type args struct {
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		want    localEx.DeviceDetails
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: test.TestDeviceID,
			},
			want: TestDeviceSimulator,
		},
		{
			name: "valid - secure",
			args: args{
				deviceID: test.TestSecureDeviceID,
			},
			want: TestSecureDeviceSimulator,
		},
		{
			name: "not-found",
			args: args{
				deviceID: "not-found",
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	c := NewTestClient()
	defer c.Close(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			got, err := c.GetDevice(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got.Resources = cleanUpResources(sortResources(got.Resources))
			got.DeviceRaw = nil
			got.Endpoints = nil
			require.Equal(t, tt.want, got)
		})
	}
}
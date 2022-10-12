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

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func addDirectDeviceToCache(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context {
	_, _ = c.deviceCache.UpdateOrStoreDevice(core.NewDevice(core.DeviceConfiguration{}, deviceID, []string{}, nil))
	return ctx
}

func makeMockObservationHandler() *mockObservationHandler {
	return &mockObservationHandler{res: make(chan coap.DecodeFunc, 1), close: make(chan struct{})}
}

type mockObservationHandler struct {
	res   chan coap.DecodeFunc
	close chan struct{}
}

func (h *mockObservationHandler) Handle(ctx context.Context, body coap.DecodeFunc) {
	h.res <- body
}

func (h *mockObservationHandler) Error(err error) { fmt.Println(err) }

func (h *mockObservationHandler) OnClose() { close(h.close) }

func (h *mockObservationHandler) waitForNotification(ctx context.Context) (coap.DecodeFunc, error) {
	select {
	case e := <-h.res:
		return e, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.close:
		return nil, fmt.Errorf("unexpected close")
	}
}

func (h *mockObservationHandler) waitForClose(ctx context.Context) error {
	select {
	case e := <-h.res:
		var d interface{}
		if err := e(d); err != nil {
			return fmt.Errorf("unexpected notification: cannot decode: %w", err)
		}
		return fmt.Errorf("unexpected notification %v", d)
	case <-ctx.Done():
		return ctx.Err()
	case <-h.close:
		return nil
	}
}

func makeMockDeviceResourcesObservationHandler() *mockDeviceResourcesObservationHandler {
	return &mockDeviceResourcesObservationHandler{
		res:   make(chan schema.ResourceLinks, 100),
		close: make(chan struct{}),
	}
}

type mockDeviceResourcesObservationHandler struct {
	res   chan schema.ResourceLinks
	close chan struct{}
}

func (h *mockDeviceResourcesObservationHandler) Handle(ctx context.Context, body schema.ResourceLinks) error {
	h.res <- body
	return nil
}

func (h *mockDeviceResourcesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *mockDeviceResourcesObservationHandler) OnClose() {
	close(h.close)
}

func (h *mockDeviceResourcesObservationHandler) waitForNotification(ctx context.Context) (schema.ResourceLinks, error) {
	select {
	case e := <-h.res:
		return e, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.close:
		return nil, fmt.Errorf("unexpected close")
	}
}

func (h *mockDeviceResourcesObservationHandler) waitForClose(ctx context.Context) error {
	select {
	case e := <-h.res:
		return fmt.Errorf("unexpected notification %v", e)
	case <-ctx.Done():
		return ctx.Err()
	case <-h.close:
		return nil
	}
}

type testSetupSecureClient struct {
	ca      []*x509.Certificate
	mfgCA   []*x509.Certificate
	mfgCert tls.Certificate
}

func (c *testSetupSecureClient) GetManufacturerCertificate() (tls.Certificate, error) {
	if c.mfgCert.PrivateKey == nil {
		return c.mfgCert, fmt.Errorf("private key not set")
	}
	return c.mfgCert, nil
}

func (c *testSetupSecureClient) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.mfgCA) == 0 {
		return nil, fmt.Errorf("certificate authority not set")
	}
	return c.mfgCA, nil
}

func (c *testSetupSecureClient) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.ca) == 0 {
		return nil, fmt.Errorf("certificate authorities not set")
	}
	return c.ca, nil
}

var certIdentity = "00000000-0000-0000-0000-000000000001"

func NewTestSecureClient() (*Client, error) {
	mfgTrustedCABlock, _ := pem.Decode(test.RootCACrt)
	if mfgTrustedCABlock == nil {
		return nil, fmt.Errorf("mfgTrustedCABlock is empty")
	}
	mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
	if err != nil {
		return nil, err
	}
	mfgCert, err := tls.X509KeyPair(test.MfgCert, test.MfgKey)
	if err != nil {
		return nil, fmt.Errorf("cannot X509KeyPair: %w", err)
	}
	cfg := Config{
		DeviceOwnershipSDK: &DeviceOwnershipSDKConfig{
			ID:               certIdentity,
			Cert:             string(test.IdentityIntermediateCA),
			CertKey:          string(test.IdentityIntermediateCAKey),
			CreateSignerFunc: test.NewIdentityCertificateSigner,
		},
	}

	client, err := NewClientFromConfig(&cfg, &testSetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
	}, core.NewNilLogger(),
	)
	if err != nil {
		return nil, err
	}
	err = client.Initialization(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}

func TestClientDeleteDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	var ctxValueKeyMockHandler struct{}
	type args struct {
		checkForSkip func(ctx context.Context, t *testing.T, c *Client, deviceID string)
		addDevice    func(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context
		cleanUp      func(ctx context.Context, t *testing.T, c *Client, deviceID string)
		deviceID     string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not found",
			args: args{
				addDevice: nil,
				deviceID:  "not-found",
			},
			want: false,
		},
		{
			name: "found",
			args: args{
				addDevice: addDirectDeviceToCache,
				deviceID:  "found",
			},
			want: true,
		},
		{
			name: "try to delete twice",
			args: args{
				deviceID: "found",
			},
			want: false,
		},
		{
			name: "delete device with resource observation",
			args: args{
				addDevice: func(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context {
					_, _, err := c.GetDeviceByMulticast(ctx, deviceID)
					require.NoError(t, err)
					h := makeMockObservationHandler()
					_, err = c.ObserveResource(ctx, deviceID, device.ResourceURI, h)
					require.NoError(t, err)
					_, err = h.waitForNotification(ctx)
					require.NoError(t, err)
					return context.WithValue(ctx, &ctxValueKeyMockHandler, h)
				},
				cleanUp: func(ctx context.Context, t *testing.T, c *Client, deviceID string) {
					h := ctx.Value(&ctxValueKeyMockHandler).(*mockObservationHandler)
					err := h.waitForClose(ctx)
					require.NoError(t, err)
				},
				deviceID: deviceID,
			},
			want: true,
		},
		{
			name: "delete device with device resources observation",
			args: args{
				checkForSkip: func(ctx context.Context, t *testing.T, c *Client, deviceID string) {
					_, links, err := c.GetDeviceByMulticast(ctx, deviceID)
					require.NoError(t, err)
					ok := c.DeleteDevice(ctx, deviceID)
					require.True(t, ok)

					res := links.GetResourceLinks(resources.ResourceType)
					require.NotEmpty(t, res)
					if !res[0].Policy.BitMask.Has(schema.Observable) {
						t.Skip("device does not support observable resources")
					}
				},
				addDevice: func(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context {
					_, _, err := c.GetDeviceByMulticast(ctx, deviceID)
					require.NoError(t, err)
					h := makeMockDeviceResourcesObservationHandler()
					_, err = c.ObserveDeviceResources(ctx, deviceID, h)
					require.NoError(t, err)
					e, err := h.waitForNotification(ctx)
					require.NoError(t, err)
					test.CheckResourceLinks(t, test.DefaultDevsimResourceLinks(), e)
					return context.WithValue(ctx, &ctxValueKeyMockHandler, h)
				},
				cleanUp: func(ctx context.Context, t *testing.T, c *Client, deviceID string) {
					h := ctx.Value(&ctxValueKeyMockHandler).(*mockDeviceResourcesObservationHandler)
					err := h.waitForClose(ctx)
					require.NoError(t, err)
				},
				deviceID: deviceID,
			},
			want: true,
		},
		{
			name: "delete device added by device discovery",
			args: args{
				addDevice: func(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context {
					_, _, err := c.GetDeviceByMulticast(ctx, deviceID)
					require.NoError(t, err)
					return ctx
				},
				deviceID: deviceID,
			},
			want: true,
		},
		{
			name: "delete device added by ip",
			args: args{
				addDevice: func(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context {
					_, _, err := c.GetDeviceByIP(ctx, ip)
					require.NoError(t, err)
					return ctx
				},
				deviceID: deviceID,
			},
			want: true,
		},
	}
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer func() {
		err := c.DisownDevice(ctx, deviceID)
		require.NoError(t, err)
	}()
	ok := c.DeleteDevice(ctx, deviceID)
	require.True(t, ok)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := ctx
			if tt.args.checkForSkip != nil {
				tt.args.checkForSkip(testCtx, t, c, tt.args.deviceID)
			}
			_, ok := c.deviceCache.GetDevice(deviceID)
			require.False(t, ok)
			if tt.args.addDevice != nil {
				testCtx = tt.args.addDevice(ctx, t, c, tt.args.deviceID)
			}
			got := c.DeleteDevice(testCtx, tt.args.deviceID)
			require.Equal(t, tt.want, got)
			if tt.args.cleanUp != nil {
				tt.args.cleanUp(testCtx, t, c, tt.args.deviceID)
			}
		})
	}
}

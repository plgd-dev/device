package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/device/test"
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
		res:   make(chan DeviceResourcesObservationEvent, 100),
		close: make(chan struct{}),
	}
}

type mockDeviceResourcesObservationHandler struct {
	res   chan DeviceResourcesObservationEvent
	close chan struct{}
}

func (h *mockDeviceResourcesObservationHandler) Handle(ctx context.Context, body DeviceResourcesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *mockDeviceResourcesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *mockDeviceResourcesObservationHandler) OnClose() {
	close(h.close)
}

func (h *mockDeviceResourcesObservationHandler) waitForNotification(ctx context.Context) (DeviceResourcesObservationEvent, error) {
	select {
	case e := <-h.res:
		return e, nil
	case <-ctx.Done():
		return DeviceResourcesObservationEvent{}, ctx.Err()
	case <-h.close:
		return DeviceResourcesObservationEvent{}, fmt.Errorf("unexpected close")
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

var CertIdentity = "00000000-0000-0000-0000-000000000001"

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
			ID:               CertIdentity,
			Cert:             string(test.IdentityIntermediateCA),
			CertKey:          string(test.IdentityIntermediateCAKey),
			CreateSignerFunc: test.NewIdentityCertificateSigner,
		},
	}

	client, err := NewClientFromConfig(&cfg, &testSetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
	}, func(err error) { fmt.Print(err) },
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
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "not found",
			args: args{
				addDevice: nil,
				deviceID:  "not-found",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "found",
			args: args{
				addDevice: addDirectDeviceToCache,
				deviceID:  "found",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "try to delete twice",
			args: args{
				deviceID: "found",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "delete device with resource observation",
			args: args{
				addDevice: func(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context {
					_, _, err := c.GetDevice(ctx, deviceID)
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
					_, links, err := c.GetDevice(ctx, deviceID)
					require.NoError(t, err)
					ok, err := c.DeleteDevice(ctx, deviceID)
					require.NoError(t, err)
					require.True(t, ok)

					res := links.GetResourceLinks(resources.ResourceType)
					require.NotEmpty(t, res)
					if !res[0].Policy.BitMask.Has(schema.Observable) {
						t.Skip("device does not support observable resources")
					}
				},
				addDevice: func(ctx context.Context, t *testing.T, c *Client, deviceID string) context.Context {
					_, _, err := c.GetDevice(ctx, deviceID)
					require.NoError(t, err)
					h := makeMockDeviceResourcesObservationHandler()
					_, err = c.ObserveDeviceResources(ctx, deviceID, h)
					require.NoError(t, err)
					expLinks := make(map[string]bool)
					for _, l := range test.TestDevsimResources {
						expLinks[l.Href] = true
					}
					for _, l := range test.TestDevsimSecResources {
						expLinks[l.Href] = true
					}
					for _, l := range test.TestDevsimPrivateResources {
						expLinks[l.Href] = true
					}
					for len(expLinks) > 0 {
						e, err := h.waitForNotification(ctx)
						require.NoError(t, err)
						require.Equal(t, DeviceResourcesObservationEvent_ADDED, e.Event)
						if _, ok := expLinks[e.Link.Href]; ok {
							delete(expLinks, e.Link.Href)
						} else {
							require.FailNowf(t, "unexpected link", e.Link.Href)
						}
					}
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
					_, _, err := c.GetDevice(ctx, deviceID)
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
					_, err := c.GetDeviceByIP(ctx, ip)
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
	ok, err := c.DeleteDevice(ctx, deviceID)
	require.NoError(t, err)
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
			got, err := c.DeleteDevice(testCtx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			if tt.args.cleanUp != nil {
				tt.args.cleanUp(testCtx, t, c, tt.args.deviceID)
			}
		})
	}
}

package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/client/core/otm"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema/credential"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/doxm"
	"github.com/plgd-dev/device/test"
	kitNet "github.com/plgd-dev/kit/v2/net"
	"github.com/stretchr/testify/require"
)

type InvalidOtmClient struct{}

func (InvalidOtmClient) Type() doxm.OwnerTransferMethod {
	return doxm.OwnerTransferMethod(-1)
}

func (InvalidOtmClient) Dial(ctx context.Context, addr kitNet.Addr, opts ...coap.DialOptionFunc) (*coap.ClientCloseHandler, error) {
	return nil, fmt.Errorf("invalid client")
}

func TestClientOwnDeviceMfg(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	signer, err := NewTestSigner()
	require.NoError(t, err)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()

	deviceID := secureDeviceID
	require := require.New(t)
	timeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()

	dev, err := c.GetDeviceByMulticast(timeout, deviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(err)
	defer func() {
		errClose := dev.Close(timeout)
		require.NoError(errClose)
	}()
	eps := dev.GetEndpoints()
	links, err := dev.GetResourceLinks(timeout, eps)
	require.NoError(err)

	err = dev.Own(timeout, links, []otm.Client{c.mfgOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(err)
	links, err = dev.GetResourceLinks(timeout, eps)
	require.NoError(err)
	err = dev.Disown(timeout, links)
	require.NoError(err)

	time.Sleep(time.Second)

	// try disown second time
	secureDeviceID = test.MustFindDeviceByName(test.DevsimName)
	dev, err = c.GetDeviceByMulticast(timeout, secureDeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(err)
	defer func() {
		errClose := dev.Close(timeout)
		require.NoError(errClose)
	}()
	eps = dev.GetEndpoints()
	links, err = dev.GetResourceLinks(timeout, eps)
	require.NoError(err)
	err = dev.Disown(timeout, links)
	require.NoError(err)

	secureDeviceID = test.MustFindDeviceByName(test.DevsimName)
	dev, err = c.GetDeviceByMulticast(timeout, secureDeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(err)
	eps = dev.GetEndpoints()
	links, err = dev.GetResourceLinks(timeout, eps)
	require.NoError(err)
	err = dev.Own(timeout, links, []otm.Client{c.mfgOtm}, core.WithActionDuringOwn(func(ctx context.Context, client *coap.ClientCloseHandler) (string, error) {
		var d device.Device
		err := client.GetResource(ctx, device.ResourceURI, &d)
		if err != nil {
			return "", core.MakeInternal(fmt.Errorf("cannot get device resource for owned device(%v): %w", secureDeviceID, err))
		}
		setDeviceOwned := doxm.DoxmUpdate{
			DeviceID: &d.ProtocolIndependentID,
		}
		/*doxm doesn't send any content for select OTM*/
		err = client.UpdateResource(ctx, doxm.ResourceURI, setDeviceOwned, nil)
		if err != nil {
			return "", core.MakeInternal(fmt.Errorf("cannot set device id %v for owned device(%v): %w", d.ProtocolIndependentID, secureDeviceID, err))
		}
		return d.ProtocolIndependentID, nil
	}), core.WithSetupCertificates(signer.Sign))
	require.NoError(err)
	require.NotEqual(t, secureDeviceID, dev.DeviceID())

	dev, err = c.GetDeviceByMulticast(timeout, dev.DeviceID(), core.DefaultDiscoveryConfiguration())
	require.NoError(err)
	eps = dev.GetEndpoints()
	links, err = dev.GetResourceLinks(timeout, eps)
	require.NoError(err)
	err = dev.Disown(timeout, links)
	require.NoError(err)
}

func TestClientOwnDeviceJustWorks(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	signer, err := NewTestSigner()
	require.NoError(t, err)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()
	require := require.New(t)
	timeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()

	device, err := c.GetDeviceByMulticast(timeout, secureDeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(err)
	defer func() {
		errClose := device.Close(timeout)
		require.NoError(errClose)
	}()
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(timeout, eps)
	require.NoError(err)

	err = device.Own(timeout, links, []otm.Client{c.justWorksOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(err)
	links, err = device.GetResourceLinks(timeout, eps)
	require.NoError(err)
	err = device.Disown(timeout, links)
	require.NoError(err)
}

func TestClientOwnDeviceInvalidClient(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	signer, err := NewTestSigner()
	require.NoError(t, err)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()
	require := require.New(t)
	timeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()

	device, err := c.GetDeviceByMulticast(timeout, secureDeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(err)
	defer func() {
		errClose := device.Close(timeout)
		require.NoError(errClose)
	}()
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(timeout, eps)
	require.NoError(err)

	err = device.Own(timeout, links, []otm.Client{InvalidOtmClient{}}, core.WithSetupCertificates(signer.Sign))
	require.Error(err)
}

func TestClientOwnDeviceWithPreviouslyBrokenConnection(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	signer, err := NewTestSigner()
	require.NoError(t, err)

	c, err := NewTestSecureClientWithTLS(true, false)
	require.NoError(t, err)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	links, err := got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)
	link, ok := links.GetResourceLink(credential.ResourceURI)
	require.True(t, ok)
	var v interface{}

	getResource := func() {
		resourceCtx, cancel := context.WithTimeout(ctx, time.Second*1)
		defer cancel()
		err = got.GetResource(resourceCtx, link, &v)
		require.Error(t, err)
	}
	getResource()

	err = got.Own(ctx, links, []otm.Client{c.justWorksOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(t, err)
	links, err = got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)
	err = got.Disown(ctx, links)
	require.NoError(t, err)

}

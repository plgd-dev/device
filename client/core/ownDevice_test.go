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

package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	justworks "github.com/plgd-dev/device/v2/client/core/otm/just-works"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/test"
	kitNet "github.com/plgd-dev/kit/v2/net"
	"github.com/stretchr/testify/require"
)

type InvalidOtmClient struct{}

func (InvalidOtmClient) Type() doxm.OwnerTransferMethod {
	return doxm.OwnerTransferMethod(-1)
}

func (InvalidOtmClient) Dial(ctx context.Context, addr kitNet.Addr) (*coap.ClientCloseHandler, error) {
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

func TestClientOwnDeviceWithFailSetupCertificates(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	dev, err := c.GetDeviceByMulticast(ctx, deviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	links, err := dev.GetResourceLinks(ctx, dev.GetEndpoints())
	require.NoError(t, err)
	err = dev.Own(ctx, links, []otm.Client{justworks.NewClient()}, core.WithSetupCertificates(func(ctx context.Context, csr []byte) ([]byte, error) {
		return nil, fmt.Errorf("invalid")
	}))
	require.Error(t, err)
	deviceID2 := test.MustFindDeviceByName(test.DevsimName)
	require.NotEqual(t, deviceID, deviceID2)
}

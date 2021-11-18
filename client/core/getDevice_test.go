package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func TestClientGetDeviceByIPWithIP4(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	links, err := got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)
	link, ok := links.GetResourceLink(platform.ResourceURI)
	require.True(t, ok)
	var v interface{}
	err = got.GetResource(ctx, link, &v)
	require.NoError(t, err)
}

func TestClientGetDeviceByIPWithIP6(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP6)

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	links, err := got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)

	err = got.Own(ctx, links, c.justWorksOtm)
	require.NoError(t, err)
	links, err = got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)
	defer func() {
		err := got.Disown(ctx, links)
		require.NoError(t, err)
	}()
}

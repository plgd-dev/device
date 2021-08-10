package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/sdk/test"
	"github.com/stretchr/testify/require"
)

func TestClient_GetDeviceByIP(t *testing.T) {
	ip := test.MustFindDeviceIP(test.TestSecureDeviceName)

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
	err = got.Disown(ctx, links)
	require.NoError(t, err)
}

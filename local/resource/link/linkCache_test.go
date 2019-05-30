package link_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCached(t *testing.T) {
	c := link.NewCache(nil, nil)
	c.Update(testDeviceID, testLink)

	l, ok := c.Get(testDeviceID, testHref)
	require.True(t, ok)
	assert.Equal(t, testLink, l)
}

func TestUncached(t *testing.T) {
	c := link.NewCache(nil, nil)

	_, ok := c.Get(testDeviceID, testHref)
	require.False(t, ok)
}

func TestDeleted(t *testing.T) {
	c := link.NewCache(nil, nil)
	c.Update(testDeviceID, testLink)
	c.Delete(testDeviceID, testHref)

	_, ok := c.Get(testDeviceID, testHref)
	require.False(t, ok)
}

func TestCreated(t *testing.T) {
	create := func(ctx context.Context, deviceID, href string) (schema.ResourceLink, error) {
		assert.Equal(t, testDeviceID, deviceID)
		assert.Equal(t, testHref, href)
		return schema.ResourceLink{
			Href: href,
		}, nil
	}
	c := link.NewCache(create, nil)

	l, err := c.GetOrCreate(context.Background(), testDeviceID, testHref)
	require.NoError(t, err)
	assert.Equal(t, testLink, l)
}

func TestCreationNotNeeded(t *testing.T) {
	c := link.NewCache(failingCreate, nil)
	c.Update(testDeviceID, testLink)

	l, err := c.GetOrCreate(context.Background(), testDeviceID, testHref)
	require.NoError(t, err)
	assert.Equal(t, testLink, l)
}

func TestCreationFailure(t *testing.T) {
	c := link.NewCache(failingCreate, nil)

	_, err := c.GetOrCreate(context.Background(), testDeviceID, testHref)
	assert.Error(t, err)
}

var (
	testDeviceID = "deviceID"
	testHref     = "/test/href"
	testLink     = schema.ResourceLink{Href: testHref}
)

func failingCreate(ctx context.Context, deviceID, href string) (res schema.ResourceLink, _ error) {
	return res, fmt.Errorf("unexpected create")
}

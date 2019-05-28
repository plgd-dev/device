package link_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCached(t *testing.T) {
	c := link.NewCache(10*time.Millisecond, nil, nil)
	c.Put(testDeviceID, testLink)

	l, ok := c.Get(testDeviceID, testHref)
	require.True(t, ok)
	assert.Equal(t, testLink, l)
}

func TestUncached(t *testing.T) {
	c := link.NewCache(10*time.Millisecond, nil, nil)

	_, ok := c.Get(testDeviceID, testHref)
	require.False(t, ok)
}

func TestExpired(t *testing.T) {
	c := link.NewCache(10*time.Millisecond, nil, nil)
	c.Put(testDeviceID, testLink)
	time.Sleep(20 * time.Millisecond)

	_, ok := c.Get(testDeviceID, testHref)
	require.False(t, ok)
}

func TestDeleted(t *testing.T) {
	c := link.NewCache(10*time.Millisecond, nil, nil)
	c.Put(testDeviceID, testLink)
	c.Delete(testDeviceID, testHref)

	_, ok := c.Get(testDeviceID, testHref)
	require.False(t, ok)
}

func TestCreated(t *testing.T) {
	create := func(ctx context.Context, c *link.Cache, deviceID, href string) error {
		assert.Equal(t, testDeviceID, deviceID)
		assert.Equal(t, testHref, href)
		c.Put(testDeviceID, testLink)
		return nil
	}
	c := link.NewCache(10*time.Millisecond, create, nil)

	l, err := c.GetOrCreate(context.Background(), testDeviceID, testHref)
	require.NoError(t, err)
	assert.Equal(t, testLink, l)
}

func TestCreationNotNeeded(t *testing.T) {
	c := link.NewCache(10*time.Millisecond, failingCreate, nil)
	c.Put(testDeviceID, testLink)

	l, err := c.GetOrCreate(context.Background(), testDeviceID, testHref)
	require.NoError(t, err)
	assert.Equal(t, testLink, l)
}

func TestCreationFailure(t *testing.T) {
	c := link.NewCache(10*time.Millisecond, failingCreate, nil)

	_, err := c.GetOrCreate(context.Background(), testDeviceID, testHref)
	assert.Error(t, err)
}

func TestMissAfterCreate(t *testing.T) {
	create := func(ctx context.Context, c *link.Cache, deviceID, href string) error {
		return nil
	}
	c := link.NewCache(10*time.Millisecond, create, nil)

	_, err := c.GetOrCreate(context.Background(), testDeviceID, testHref)
	assert.Error(t, err)
}

var (
	testDeviceID = "deviceID"
	testHref     = "/test/href"
	testLink     = schema.ResourceLink{Href: testHref}
)

func failingCreate(ctx context.Context, c *link.Cache, deviceID, href string) error {
	return fmt.Errorf("unexpected create")
}

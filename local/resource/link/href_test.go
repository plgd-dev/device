package link_test

import (
	"strings"
	"testing"

	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var deviceID = "12345678-90ab-cdef-1234-123456789012"

func TestHrefParser(t *testing.T) {
	uri := "/test/uri"
	s := "/" + deviceID + uri
	href, err := link.ParseHref(s)
	require.NoError(t, err)
	assert.Equal(t, deviceID, href.DeviceID)
	assert.Equal(t, uri, href.Href)
}

func TestUppercaseID(t *testing.T) {
	uri := "/test/uri"
	s := "/" + strings.ToUpper(deviceID) + uri
	href, err := link.ParseHref(s)
	require.NoError(t, err)
	assert.Equal(t, deviceID, href.DeviceID)
	assert.Equal(t, uri, href.Href)
}

func TestInvalidID(t *testing.T) {
	_, err := link.ParseHref("/invalidID/test/uri")
	require.Error(t, err)
}

func TestNoHref(t *testing.T) {
	_, err := link.ParseHref("/" + deviceID)
	require.Error(t, err)
}

func TestHrefString(t *testing.T) {
	href := link.Href{DeviceID: deviceID, Href: "/test/uri"}
	s := href.String()
	assert.Equal(t, "/"+deviceID+"/test/uri", s)
}

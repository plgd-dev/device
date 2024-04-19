package resources_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/stretchr/testify/assert"
)

func TestSupportedOperationHasOperation(t *testing.T) {
	tests := []struct {
		name       string
		operation  resources.SupportedOperation
		check      resources.SupportedOperation
		wantResult bool
	}{
		{"Test Read", resources.SupportedOperationRead, resources.SupportedOperationRead, true},
		{"Test Write", resources.SupportedOperationWrite, resources.SupportedOperationWrite, true},
		{"Test Observe", resources.SupportedOperationObserve, resources.SupportedOperationObserve, true},
		{"Test Read and Write", resources.SupportedOperationRead | resources.SupportedOperationWrite, resources.SupportedOperationRead, true},
		{"Test Write and Observe", resources.SupportedOperationWrite | resources.SupportedOperationObserve, resources.SupportedOperationObserve, true},
		{"Test All", resources.SupportedOperationRead | resources.SupportedOperationWrite | resources.SupportedOperationObserve, resources.SupportedOperationRead, true},
		{"Test None", 0, resources.SupportedOperationRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operation.HasOperation(tt.check)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestResourceSupportsOperations(t *testing.T) {
	tests := []struct {
		name           string
		getHandler     resources.GetHandlerFunc
		postHandler    resources.PostHandlerFunc
		policyBitMask  schema.BitMask
		wantOperations resources.SupportedOperation
	}{
		{"Only Read", func(*net.Request) (*pool.Message, error) { return &pool.Message{}, nil }, nil, 0, resources.SupportedOperationRead},
		{"Only Write", nil, func(*net.Request) (*pool.Message, error) { return &pool.Message{}, nil }, 0, resources.SupportedOperationWrite},
		{"Read and Observe", func(*net.Request) (*pool.Message, error) { return &pool.Message{}, nil }, nil, schema.Observable, resources.SupportedOperationRead | resources.SupportedOperationObserve},
		{"All Operations", func(*net.Request) (*pool.Message, error) { return &pool.Message{}, nil }, func(*net.Request) (*pool.Message, error) { return &pool.Message{}, nil }, schema.Observable, resources.SupportedOperationRead | resources.SupportedOperationWrite | resources.SupportedOperationObserve},
		{"No Operations", nil, nil, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resources.NewResource("/test", tt.getHandler, tt.postHandler, nil, nil)
			r.PolicyBitMask = tt.policyBitMask
			operations := r.SupportsOperations()
			assert.Equal(t, tt.wantOperations, operations)
		})
	}
}

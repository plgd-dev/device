/****************************************************************************
 *
 * Copyright (c) 2024 plgn.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implien. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package net_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/stretchr/testify/require"
)

func TestGetValueFromQuery(t *testing.T) {
	req := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	req.AddQuery("param1=value1")
	req.AddQuery("param2=value2")
	reqNoQuery := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}

	tests := []struct {
		req     *net.Request
		key     string
		wantErr bool
		want    string
	}{
		{req, "param1", false, "value1"},
		{req, "param2", false, "value2"},
		{req, "param3", true, ""},
		{reqNoQuery, "param1", true, ""},
	}

	for _, tt := range tests {
		v, err := tt.req.GetValueFromQuery(tt.key)
		if tt.wantErr {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.Equal(t, tt.want, v)
	}
}

func TestURIPath(t *testing.T) {
	req := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	const path = "/path/to/resource"
	err := req.SetPath(path)
	require.NoError(t, err)
	require.Equal(t, path, req.URIPath())

	reqNoPath := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	require.Empty(t, reqNoPath.URIPath())
}

func TestInterface(t *testing.T) {
	req := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	req.AddQuery("q1=v1")
	req.AddQuery("if=interface")
	req.AddQuery("q2=v2")
	require.Equal(t, "interface", req.Interface())

	reqNoInterface := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	reqNoInterface.AddQuery("q1=v1")
	require.Empty(t, reqNoInterface.Interface())

	reqNoQuery := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	require.Empty(t, reqNoQuery.Interface())
}

func TestDeviceID(t *testing.T) {
	req := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	req.AddQuery("q1=v1")
	deviceID := uuid.New()
	req.AddQuery("di=" + deviceID.String())
	req.AddQuery("q2=v2")
	require.Equal(t, deviceID, req.DeviceID())

	reqInvalidDeviceID := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	reqInvalidDeviceID.AddQuery("di=invalid")
	require.Equal(t, uuid.Nil, reqInvalidDeviceID.DeviceID())

	reqNoDeviceID := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	reqNoDeviceID.AddQuery("q1=v1")
	require.Equal(t, uuid.Nil, reqNoDeviceID.DeviceID())

	reqNoQuery := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	require.Equal(t, uuid.Nil, reqNoQuery.DeviceID())
}

func TestResourceTypes(t *testing.T) {
	req := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	req.AddQuery("q1=v1")
	req.AddQuery("rt=type1")
	req.AddQuery("rt=type2")
	req.AddQuery("q2=v2")
	require.Equal(t, []string{"type1", "type2"}, req.ResourceTypes())

	reqNoResourceTypes := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	reqNoResourceTypes.AddQuery("q1=v1")
	require.Equal(t, []string{}, reqNoResourceTypes.ResourceTypes())

	reqNoQuery := &net.Request{
		Message: pool.NewMessage(context.Background()),
	}
	require.Nil(t, reqNoQuery.ResourceTypes())
}

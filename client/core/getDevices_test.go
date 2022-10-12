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
	"github.com/stretchr/testify/require"
)

func TestDeviceDiscovery(t *testing.T) {
	c := core.NewClient()
	timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	h := testDeviceHandler{}
	err := c.GetDevicesByMulticast(timeout, core.DefaultDiscoveryConfiguration(), &h)
	require.NoError(t, err)
}

type testDeviceHandler struct{}

func (h *testDeviceHandler) Handle(ctx context.Context, d *core.Device) {
	defer func() {
		if errClose := d.Close(ctx); errClose != nil {
			h.Error(fmt.Errorf("testDeviceHandler.Handle: %w", errClose))
		}
	}()
}

func (h *testDeviceHandler) Error(err error) {
	fmt.Printf("testDeviceHandler.Error: %v\n", err)
}

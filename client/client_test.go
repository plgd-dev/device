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

package client_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/stretchr/testify/require"
)

var (
	ETagSupported                   = false
	ETagBatchSupported              = false
	ETagIncrementalChangesSupported = false
)

func checkIfBatchAndIncrementalChangesSupported(ctx context.Context, d *core.Device, link schema.ResourceLink) (bool, bool) {
	v1 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
	err := d.GetResourceWithCodec(ctx, link, codecOcf.VNDOCFCBORCodec{}, &v1, coap.WithInterface(interfaces.OC_IF_B))
	if err != nil {
		fmt.Printf("cannot check if incremental changes supported: first request failed: %v\n", err)
		return false, false
	}
	if v1.ETag == nil {
		log.Printf("cannot check if incremental changes supported: ETag is nil\n")
		return false, false
	}
	opts := make([]coap.OptionFunc, 0, 2)
	opts = append(opts, coap.WithInterface(interfaces.OC_IF_B))
	etags := make([][]byte, 0, v1.Body.Len())
	for _, v := range v1.Body {
		if len(v.ETag) != 0 {
			etags = append(etags, v.ETag)
		}
	}
	queries := coap.EncodeETagsForIncrementalChanges(etags)
	for _, q := range queries {
		opts = append(opts, coap.WithQuery(q))
	}
	v2 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
	err = d.GetResourceWithCodec(ctx, link, codecOcf.VNDOCFCBORCodec{}, &v2, opts...)
	if err != nil {
		log.Fatalf("cannot check if incremental changes supported: second request failed: %v\n", err)
		return true, false
	}
	return true, v2.Code == codes.Valid
}

func init() {
	panicIfErr := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	deviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := testClient.NewTestSecureClient()
	panicIfErr(err)
	defer func() {
		errC := c.Close(context.Background())
		panicIfErr(errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()

	deviceID, err = c.OwnDevice(ctx, deviceID)
	panicIfErr(err)
	defer func() {
		ctxD, cancelD := context.WithTimeout(context.Background(), time.Second*2)
		defer cancelD()
		errD := c.DisownDevice(ctxD, deviceID)
		panicIfErr(errD)
	}()

	d, links, err := c.GetDevice(ctx, deviceID, client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration()))
	panicIfErr(err)
	link, err := core.GetResourceLink(links, resources.ResourceURI)
	panicIfErr(err)
	// force the use of a secure endpoint
	secureEndpoints := link.Endpoints.FilterSecureEndpoints()
	if (len(secureEndpoints)) != 0 {
		link.Endpoints = secureEndpoints
	}

	v := coap.DetailedResponse[interface{}]{}
	err = d.GetResourceWithCodec(ctx, link, codecOcf.VNDOCFCBORCodec{}, &v)
	panicIfErr(err)
	if v.ETag != nil {
		ETagSupported = true
		fmt.Println("ETags supported")

		ETagBatchSupported, ETagIncrementalChangesSupported = checkIfBatchAndIncrementalChangesSupported(ctx, d, link)
		if ETagBatchSupported {
			fmt.Println("ETags for batch interface supported")
		}
		if ETagIncrementalChangesSupported {
			fmt.Println("ETags incremental changes supported")
		}
	}
}

func disown(t *testing.T, c *client.Client, deviceID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	err := c.DisownDevice(ctx, deviceID)
	require.NoError(t, err)
}

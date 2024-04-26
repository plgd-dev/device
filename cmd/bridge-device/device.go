package main

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/bridge/service"
	bridgeDevice "github.com/plgd-dev/device/v2/cmd/bridge-device/device"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
)

type resourceData struct {
	Name string `json:"name,omitempty"`
}

type resourceDataSync struct {
	resourceData
	lock sync.Mutex
}

func (r *resourceDataSync) setName(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Name = name
}

func (r *resourceDataSync) copy() resourceData {
	r.lock.Lock()
	defer r.lock.Unlock()
	return resourceData{
		Name: r.Name,
	}
}

func addResources(d service.Device, numResources int) {
	if numResources <= 0 {
		return
	}
	obsWatcher := coapSync.NewMap[uint64, func()]()
	for i := 0; i < numResources; i++ {
		addResource(d, i, obsWatcher)
	}
	go func() {
		// notify observers every 500ms
		for {
			time.Sleep(time.Millisecond * 500)
			obsWatcher.Range(func(_ uint64, h func()) bool {
				h()
				return true
			})
		}
	}()
}

func addResource(d service.Device, idx int, obsWatcher *coapSync.Map[uint64, func()]) {
	rds := resourceDataSync{
		resourceData: resourceData{
			Name: fmt.Sprintf("test-%v", idx),
		},
	}

	resHandler := func(req *net.Request) (*pool.Message, error) {
		resp := pool.NewMessage(req.Context())
		switch req.Code() {
		case codes.GET:
			resp.SetCode(codes.Content)
		case codes.POST:
			resp.SetCode(codes.Changed)
		default:
			return nil, fmt.Errorf("invalid method %v", req.Code())
		}
		resp.SetContentFormat(message.AppOcfCbor)
		data, err := cbor.Encode(rds.copy())
		if err != nil {
			return nil, err
		}
		resp.SetBody(bytes.NewReader(data))
		return resp, nil
	}

	resPostHandler := func(req *net.Request) (*pool.Message, error) {
		codec := codecOcf.VNDOCFCBORCodec{}
		var newData resourceData
		err := codec.Decode(req.Message, &newData)
		if err != nil {
			return nil, err
		}
		rds.setName(newData.Name)
		return resHandler(req)
	}

	var subID atomic.Uint64
	res := resources.NewResource(bridgeDevice.GetTestResourceHref(idx), resHandler, resPostHandler, []string{bridgeDevice.TestResourceType}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW})
	res.SetObserveHandler(d.GetLoop(), func(req *net.Request, handler func(msg *pool.Message, err error)) (cancel func(), err error) {
		sub := subID.Add(1)
		obsWatcher.Store(sub, func() {
			resp, err := resHandler(req)
			if err != nil {
				handler(nil, err)
				return
			}
			handler(resp, nil)
		})
		return func() {
			obsWatcher.Delete(sub)
		}, nil
	})
	d.AddResources(res)
}

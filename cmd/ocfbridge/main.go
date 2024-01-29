package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"gopkg.in/yaml.v3"
)

func loadConfig(configFile string) (Config, error) {
	// Sanitize the configFile variable to ensure it only contains a valid file path
	configFile = filepath.Clean(configFile)
	f, err := os.Open(configFile)
	if err != nil {
		return Config{}, err
	}
	defer func() {
		_ = f.Close()
	}()
	var cfg Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

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
		for range time.After(time.Millisecond * 500) {
			obsWatcher.Range(func(key uint64, h func()) bool {
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

	var subID atomic.Uint64
	res := resources.NewResource(fmt.Sprintf("/test/%d", idx), resHandler, func(req *net.Request) (*pool.Message, error) {
		codec := codecOcf.VNDOCFCBORCodec{}
		var newData resourceData
		err := codec.Decode(req.Message, &newData)
		if err != nil {
			return nil, err
		}
		rds.setName(newData.Name)
		return resHandler(req)
	}, []string{"x.plgd.test"}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW})
	res.SetObserveHandler(func(req *net.Request, handler func(msg *pool.Message, err error)) (cancel func(), err error) {
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
	d.AddResource(res)
}

func main() {
	configFile := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()
	cfg, err := loadConfig(*configFile)
	if err != nil {
		panic(err)
	}
	s, err := service.New(cfg.Config)
	if err != nil {
		panic(err)
	}
	for i := 0; i < cfg.NumGeneratedBridgedDevices; i++ {
		newDevice := func(id uuid.UUID, piid uuid.UUID) service.Device {
			d := device.New(device.Config{
				Name:                  fmt.Sprintf("bridged-device-%d", i),
				ResourceTypes:         []string{"oic.d.virtual"},
				ID:                    id,
				ProtocolIndependentID: piid,
				MaxMessageSize:        cfg.Config.API.CoAP.MaxMessageSize,
				Cloud: device.CloudConfig{
					Enabled: true,
				},
			}, nil, func() map[string]interface{} {
				return map[string]interface{}{
					"my-property": "my-value",
				}
			})
			return d
		}
		d, ok := s.CreateDevice(uuid.New(), newDevice)
		if ok {
			addResources(d, cfg.NumResourcesPerDevice)
			d.Init()
		}
	}
	err = s.Serve()
	if err != nil {
		panic(err)
	}
}

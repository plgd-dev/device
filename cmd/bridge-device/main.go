package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	thingDescriptionResource "github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/log"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/plgd-dev/device/v2/schema"
	deviceResource "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
	"gopkg.in/yaml.v3"
)

const myPropertyKey = "my-property"

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
	if err = cfg.Validate(); err != nil {
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

func getCloudTLS(cfg CloudConfig, credentialEnabled bool) (cloud.CAPool, *tls.Certificate, error) {
	var ca []*x509.Certificate
	var err error
	if cfg.TLS.CAPoolPath == "" && !credentialEnabled {
		return cloud.CAPool{}, nil, errors.New("cannot load ca: caPoolPath is empty")
	}
	if cfg.TLS.CAPoolPath != "" {
		ca, err = pkgX509.ReadPemCertificates(cfg.TLS.CAPoolPath)
		if err != nil {
			return cloud.CAPool{}, nil, fmt.Errorf("cannot load ca('%v'): %w", cfg.TLS.CAPoolPath, err)
		}
	}
	caPool := cloud.MakeCAPool(func() []*x509.Certificate {
		return ca
	}, cfg.TLS.UseSystemCAPool)

	if cfg.TLS.KeyPath == "" {
		return caPool, nil, nil
	}

	cert, err := tls.LoadX509KeyPair(cfg.TLS.CertPath, cfg.TLS.KeyPath)
	if err != nil {
		return cloud.CAPool{}, nil, fmt.Errorf("cannot load cert(%v, %v): %w", cfg.TLS.CertPath, cfg.TLS.KeyPath, err)
	}
	return caPool, &cert, nil
}

func handleSignals(s *service.Service) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGINT:
			os.Exit(0)
			return
		case syscall.SIGTERM:
			_ = s.Shutdown()
			return
		}
	}
}

func getCloudOpts(cfg Config) ([]device.Option, error) {
	caPool, cert, err := getCloudTLS(cfg.Cloud, cfg.Credential.Enabled)
	if err != nil {
		return nil, err
	}
	opts := []device.Option{device.WithCAPool(caPool)}
	if cert != nil {
		opts = append(opts, device.WithGetCertificates(func(string) []tls.Certificate {
			return []tls.Certificate{*cert}
		}))
	}
	return opts, nil
}

func getTDOpts(cfg Config) ([]device.Option, error) {
	tdJson, err := os.ReadFile(cfg.ThingDescription.File)
	if err != nil {
		return nil, err
	}
	td, err := wotTD.UnmarshalThingDescription(tdJson)
	if err != nil {
		return nil, err
	}
	return []device.Option{device.WithThingDescription(func(_ context.Context, dev *device.Device, endpoints schema.Endpoints) *wotTD.ThingDescription {
		endpoint := ""
		if len(endpoints) > 0 {
			endpoint = endpoints[0].URI
		}
		newTD := thingDescription.PatchThingDescription(td, dev, endpoint, func(resourceHref string, resource thingDescription.Resource) (wotTD.PropertyElement, bool) {
			propElement, ok := td.Properties[resourceHref]
			if !ok {
				propElement, ok = thingDescriptionResource.GetOCFResourcePropertyElement(resourceHref)
				if ok && resourceHref == deviceResource.ResourceURI && propElement.Properties != nil && propElement.Properties.DataSchemaMap != nil {
					stringType := wotTD.String
					readOnly := true
					propElement.Properties.DataSchemaMap[myPropertyKey] = wotTD.DataSchema{
						DataSchemaType: &stringType,
						ReadOnly:       &readOnly,
					}
				}
			}
			if !ok {
				return wotTD.PropertyElement{}, false
			}
			propElement = thingDescription.PatchPropertyElement(propElement, dev.GetID(), resource, endpoint != "")
			return propElement, true
		})
		return &newTD
	})}, nil
}

func getOpts(cfg Config) ([]device.Option, error) {
	opts := []device.Option{
		device.WithGetAdditionalPropertiesForResponse(func() map[string]interface{} {
			return map[string]interface{}{
				myPropertyKey: "my-value",
			}
		}),
	}
	if cfg.Cloud.Enabled {
		cloudOpts, err := getCloudOpts(cfg)
		if err != nil {
			return nil, err
		}
		opts = append(opts, cloudOpts...)
	}
	if cfg.ThingDescription.Enabled {
		tdOpts, err := getTDOpts(cfg)
		if err != nil {
			return nil, err
		}
		opts = append(opts, tdOpts...)
	}
	return opts, nil
}

func main() {
	configFile := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()
	cfg, err := loadConfig(*configFile)
	if err != nil {
		panic(err)
	}
	s, err := service.New(cfg.Config, service.WithLogger(log.NewStdLogger(cfg.Log.Level)))
	if err != nil {
		panic(err)
	}

	opts, err := getOpts(cfg)
	if err != nil {
		panic(err)
	}

	for i := 0; i < cfg.NumGeneratedBridgedDevices; i++ {
		newDevice := func(id uuid.UUID, piid uuid.UUID) (service.Device, error) {
			return device.New(device.Config{
				Name:                  fmt.Sprintf("bridged-device-%d", i),
				ResourceTypes:         []string{"oic.d.virtual"},
				ID:                    id,
				ProtocolIndependentID: piid,
				MaxMessageSize:        cfg.Config.API.CoAP.MaxMessageSize,
				Cloud: device.CloudConfig{
					Enabled: cfg.Cloud.Enabled,
				},
				Credential: device.CredentialConfig{
					Enabled: cfg.Credential.Enabled,
				},
			}, append(opts, device.WithLogger(device.NewLogger(id, cfg.Log.Level)))...)
		}
		d, errC := s.CreateDevice(uuid.New(), newDevice)
		if errC == nil {
			addResources(d, cfg.NumResourcesPerDevice)
			d.Init()
		}
	}

	go func() {
		handleSignals(s)
	}()

	if err = s.Serve(); err != nil {
		panic(err)
	}
}

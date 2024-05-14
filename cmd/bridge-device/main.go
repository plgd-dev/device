package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	thingDescriptionResource "github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	"github.com/plgd-dev/device/v2/bridge/service"
	bridgeDevice "github.com/plgd-dev/device/v2/cmd/bridge-device/device"
	"github.com/plgd-dev/device/v2/pkg/log"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/plgd-dev/device/v2/schema"
	deviceResource "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

func getCloudTLS(cfg bridgeDevice.CloudConfig, credentialEnabled bool) (cloud.CAPool, *tls.Certificate, error) {
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

func getCloudOpts(cfg bridgeDevice.Config) ([]device.Option, error) {
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

func patchPropertyElement(td wotTD.ThingDescription, dev *device.Device, endpoint string, resourceHref string, resource thingDescription.Resource) (wotTD.PropertyElement, bool) {
	propElement, ok := td.Properties[resourceHref]
	if !ok {
		propElement, ok = thingDescriptionResource.GetOCFResourcePropertyElement(resourceHref)
		if ok && resourceHref == deviceResource.ResourceURI && propElement.Properties != nil && propElement.Properties.DataSchemaMap != nil {
			addProps := bridgeDevice.GetDataSchemaForAdditionalProperties()
			for key, prop := range addProps {
				propElement.Properties.DataSchemaMap[key] = prop
			}
		}
	}
	if !ok {
		return wotTD.PropertyElement{}, false
	}
	var f thingDescription.CreateFormsFunc
	if endpoint != "" {
		f = thingDescription.CreateCOAPForms
	}
	propElement, err := thingDescription.PatchPropertyElement(propElement, resource.GetResourceTypes(), dev.GetID(), resource.GetHref(), resource.SupportsOperations(), message.AppCBOR, f)
	return propElement, err == nil
}

func getTDOpts(cfg bridgeDevice.Config) ([]device.Option, error) {
	td, err := bridgeDevice.GetThingDescription(cfg.ThingDescription.File, cfg.NumResourcesPerDevice)
	if err != nil {
		return nil, err
	}
	return []device.Option{device.WithThingDescription(func(_ context.Context, dev *device.Device, endpoints schema.Endpoints) *wotTD.ThingDescription {
		endpoint := ""
		if len(endpoints) > 0 {
			endpoint = endpoints[0].URI
		}
		newTD := thingDescription.PatchThingDescription(td, dev, endpoint, func(resourceHref string, resource thingDescription.Resource) (wotTD.PropertyElement, bool) {
			return patchPropertyElement(td, dev, endpoint, resourceHref, resource)
		})
		return &newTD
	})}, nil
}

func getOpts(cfg bridgeDevice.Config) ([]device.Option, error) {
	opts := []device.Option{
		device.WithGetAdditionalPropertiesForResponse(bridgeDevice.GetAdditionalProperties),
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
	cfg, err := bridgeDevice.LoadConfig(*configFile)
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
				ResourceTypes:         []string{bridgeDevice.DeviceResourceType},
				ID:                    id,
				ProtocolIndependentID: piid,
				MaxMessageSize:        cfg.Config.API.CoAP.MaxMessageSize,
				Cloud: device.CloudConfig{
					Enabled: cfg.Cloud.Enabled,
					Config: cloud.Config{
						CloudID: cfg.Cloud.CloudID,
					},
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

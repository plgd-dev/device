package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/service"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
)

const (
	numGeneratedBridgedDevices = 3
)

func testConfig() service.Config {
	return service.Config{
		API: service.APIConfig{
			CoAP: service.CoAPConfig{
				ID: uuid.New().String(),
				Config: net.Config{
					ExternalAddresses: []string{"127.0.0.1:15683", "[::1]:15683"},
					MaxMessageSize:    2097152,
				},
			},
		},
	}
}

func getCloudTLS() (cloud.CAPool, *tls.Certificate, error) {
	caPath := os.Getenv("CA_POOL")
	fmt.Printf("Loading CA(%s)\n", caPath)
	ca, err := pkgX509.ReadPemCertificates(caPath)
	if err != nil {
		return cloud.CAPool{}, nil, fmt.Errorf("cannot load ca: %w", err)
	}
	caPool := cloud.MakeCAPool(func() []*x509.Certificate {
		return ca
	}, false)

	certPath := os.Getenv("CERT_FILE")
	keyPath := os.Getenv("KEY_FILE")
	if keyPath != "" && certPath != "" {
		fmt.Printf("Loading certificate(%s) and key(%s)\n", certPath, keyPath)
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return cloud.CAPool{}, nil, fmt.Errorf("cannot load cert: %w", err)
		}
		return caPool, &cert, nil
	}
	return caPool, nil, nil
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

func main() {
	cfg := testConfig()
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	s, err := service.New(cfg)
	if err != nil {
		panic(err)
	}

	opts := []device.Option{}
	caPool, cert, errC := getCloudTLS()
	if errC != nil {
		panic(errC)
	}
	opts = append(opts, device.WithCAPool(caPool))
	if cert != nil {
		opts = append(opts, device.WithGetCertificates(func(string) []tls.Certificate {
			return []tls.Certificate{*cert}
		}))
	}

	for i := 0; i < numGeneratedBridgedDevices; i++ {
		newDevice := func(id uuid.UUID, piid uuid.UUID) service.Device {
			d, errD := device.New(device.Config{
				Name:                  fmt.Sprintf("bridged-device-%d", i),
				ResourceTypes:         []string{"oic.d.virtual"},
				ID:                    id,
				ProtocolIndependentID: piid,
				MaxMessageSize:        cfg.API.CoAP.MaxMessageSize,
				Cloud: device.CloudConfig{
					Enabled: true,
					Config: cloud.Config{
						CloudID: os.Getenv("CLOUD_SID"),
					},
				},
			}, opts...)
			if errD != nil {
				panic(errD)
			}
			return d
		}
		d, ok := s.CreateDevice(uuid.New(), newDevice)
		if ok {
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

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/service"
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
					ExternalAddress: "127.0.0.1:15683",
					MaxMessageSize:  2097152,
				},
			},
		},
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
	for i := 0; i < numGeneratedBridgedDevices; i++ {
		newDevice := func(id uuid.UUID, piid uuid.UUID) service.Device {
			d := device.New(device.Config{
				Name:                  fmt.Sprintf("bridged-device-%d", i),
				ResourceTypes:         []string{"oic.d.virtual"},
				ID:                    id,
				ProtocolIndependentID: piid,
				MaxMessageSize:        cfg.API.CoAP.MaxMessageSize,
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
			d.Init()
		}
	}

	// Signal handling.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		for sig := range sigCh {
			log.Printf("Trapped \"%v\" signal\n", sig)
			switch sig {
			case syscall.SIGINT:
				log.Println("Exiting...")
				os.Exit(0)
				return
			case syscall.SIGTERM:
				_ = s.Shutdown()
				return
			}
		}
	}()

	err = s.Serve()
	if err != nil {
		panic(err)
	}
}

package test

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/schema"
)

func MustGetHostname() string {
	n, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return n
}

func MustFindDeviceByName(name string) (deviceID string) {
	var err error
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		deviceID, err = FindDeviceByName(ctx, name)
		if err == nil {
			return deviceID
		}
	}
	panic(err)
}

type findDeviceIDByNameHandler struct {
	id     atomic.Value
	name   string
	cancel context.CancelFunc
}

func (h *findDeviceIDByNameHandler) Handle(ctx context.Context, device *core.Device) {
	defer device.Close(ctx)
	eps := device.GetEndpoints()
	var d schema.Device
	err := device.GetResource(ctx, schema.ResourceLink{
		Href:      "/oic/d",
		Endpoints: eps,
	}, &d)
	if err != nil {
		return
	}
	if d.Name == h.name {
		h.id.Store(d.ID)
		h.cancel()
	}
}

func (h *findDeviceIDByNameHandler) Error(err error) {}

func FindDeviceByName(ctx context.Context, name string) (deviceID string, _ error) {
	client := core.NewClient()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h := findDeviceIDByNameHandler{
		name:   name,
		cancel: cancel,
	}

	err := client.GetDevicesV2(ctx, core.DefaultDiscoveryConfiguration(), &h)
	if err != nil {
		return "", fmt.Errorf("could not find the device named %s: %w", name, err)
	}
	id, ok := h.id.Load().(string)
	if !ok || id == "" {
		return "", fmt.Errorf("could not find the device named %s: not found", name)
	}
	return id, nil
}

func FindDeviceIP(ctx context.Context, deviceName string) (string, error) {
	deviceID := MustFindDeviceByName(deviceName)
	client := core.NewClient()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	device, err := client.GetDeviceByMulticast(ctx, deviceID, core.DefaultDiscoveryConfiguration())
	if err != nil {
		return "", err
	}
	defer device.Close(ctx)

	if len(device.GetEndpoints()) == 0 {
		return "", fmt.Errorf("endpoints are not set for device %v", device)
	}
	addr, err := device.GetEndpoints().GetAddr("coap")
	if err != nil {
		return "", fmt.Errorf("cannot get coap endpoint %v", device)
	}
	return addr.GetHostname(), nil
}

func MustFindDeviceIP(name string) (ip string) {
	var err error
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		ip, err = FindDeviceIP(ctx, name)
		if err == nil {
			return ip
		}
	}
	panic(err)
}

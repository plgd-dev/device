package test

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/pkg/security/signer"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/kit/v2/log"
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

func (h *findDeviceIDByNameHandler) Handle(ctx context.Context, dev *core.Device) {
	defer func() {
		if errClose := dev.Close(ctx); errClose != nil {
			h.Error(errClose)
		}
	}()
	eps := dev.GetEndpoints()
	var d device.Device
	err := dev.GetResource(ctx, schema.ResourceLink{
		Href:      device.ResourceURI,
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

func (h *findDeviceIDByNameHandler) Error(err error) {
	log.Debug(err)
}

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

func NewIdentityCertificateSigner(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore time.Time, validNotAfter time.Time) core.CertificateSigner {
	return signer.NewOCFIdentityCertificate(caCert, caKey, validNotBefore, validNotAfter)
}

type IPType int

const (
	ANY IPType = 0
	IP4 IPType = 1
	IP6 IPType = 2
)

func getDiscoveryConfiguration(ipType IPType) core.DiscoveryConfiguration {
	discoveryCfg := core.DefaultDiscoveryConfiguration()
	switch ipType {
	case IP4:
		discoveryCfg.MulticastAddressUDP6 = nil
	case IP6:
		discoveryCfg.MulticastAddressUDP4 = nil
	}
	return discoveryCfg
}

func getDeviceIP(device *core.Device, ipType IPType) (string, error) {
	if len(device.GetEndpoints()) == 0 {
		return "", fmt.Errorf("endpoints are not set for device %v", device)
	}
	eps := device.GetEndpoints().FilterUnsecureEndpoints()
	if ipType == ANY {
		addr, err := eps.GetAddr(schema.UDPScheme)
		if err != nil {
			return "", fmt.Errorf("cannot get coap endpoint %v", device)
		}
		return addr.GetHostname(), nil
	}
	for _, e := range eps {
		addr, err := e.GetAddr()
		if err != nil {
			continue
		}
		if schema.Scheme(addr.GetScheme()) != schema.UDPScheme {
			continue
		}
		if strings.Contains(addr.GetHostname(), ":") && ipType == IP6 {
			return addr.GetHostname(), nil
		}
		if ipType == IP4 {
			return addr.GetHostname(), nil
		}
	}
	return "", fmt.Errorf("ipType(%v) not found in %v", ipType, eps)
}

func FindDeviceIP(ctx context.Context, deviceName string, ipType IPType) (string, error) {
	deviceID := MustFindDeviceByName(deviceName)
	client := core.NewClient()
	discoveryCfg := getDiscoveryConfiguration(ipType)
	device, err := client.GetDeviceByMulticast(ctx, deviceID, discoveryCfg)
	if err != nil {
		return "", err
	}
	defer func() {
		if errClose := device.Close(ctx); errClose != nil {
			log.Errorf("FindDeviceIP: %w", errClose)
		}
	}()
	return getDeviceIP(device, ipType)
}

func MustFindDeviceIP(name string, ipType IPType) (ip string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ip, err := FindDeviceIP(ctx, name, ipType)
	if err == nil {
		return ip
	}
	panic(err)
}

func DefaultSwitchResourceLink(id string) schema.ResourceLink {
	return schema.ResourceLink{
		Href:          TestResourceSwitchesInstanceHref(id),
		ResourceTypes: []string{types.BINARY_SWITCH},
		Interfaces:    []string{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
		Policy: &schema.Policy{
			BitMask: schema.Discoverable | schema.Observable,
		},
	}
}

func MakeSwitchResourceDefaultData() map[string]interface{} {
	s := DefaultSwitchResourceLink("")
	rif := make([]interface{}, 0, len(s.Interfaces))
	for _, i := range s.Interfaces {
		rif = append(rif, i)
	}
	rt := make([]interface{}, 0, len(s.ResourceTypes))
	for _, i := range s.ResourceTypes {
		rt = append(rt, i)
	}
	return map[string]interface{}{
		"if": rif,
		"rt": rt,
		"rep": map[interface{}]interface{}{
			"value": false,
		},
		"p": map[interface{}]interface{}{
			"bm": uint64(s.Policy.BitMask),
		},
	}
}

func MakeSwitchResourceData(overrides map[string]interface{}) map[string]interface{} {
	data := MakeSwitchResourceDefaultData()
	for k, v := range overrides {
		data[k] = v
	}
	return data
}

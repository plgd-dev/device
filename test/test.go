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

package test

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/security/signer"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/stretchr/testify/require"
)

const TestTimeout = time.Second * 8

func MustGetHostname() string {
	n, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return n
}

func findDeviceByNameWithRetry(name string, tries int) (string, error) {
	for i := 0; i < tries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		deviceID, err := FindDeviceByName(ctx, name)
		if err == nil {
			return deviceID, nil
		}
	}
	return "", fmt.Errorf("device named %s not found", name)
}

func MustFindDeviceByName(name string) string {
	deviceID, err := findDeviceByNameWithRetry(name, 3)
	if err == nil {
		return deviceID
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
		if errC := dev.Close(ctx); errC != nil {
			h.Error(errC)
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
	fmt.Printf("%v\n", err)
}

func FindDeviceByName(ctx context.Context, name string) (deviceID string, _ error) {
	client := core.NewClient()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	h := findDeviceIDByNameHandler{
		name:   name,
		cancel: cancel,
	}

	err := client.GetDevicesByMulticast(ctx, core.DefaultDiscoveryConfiguration(), &h)
	if err != nil {
		return "", fmt.Errorf("could not find the device named %s: %w", name, err)
	}
	id, ok := h.id.Load().(string)
	if !ok || id == "" {
		return "", fmt.Errorf("could not find the device named %s: not found", name)
	}
	return id, nil
}

func NewIdentityCertificateSigner(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore, validNotAfter time.Time, crlDistributionPoints []string) (core.CertificateSigner, error) {
	return signer.NewOCFIdentityCertificate(caCert, caKey, validNotBefore, validNotAfter, crlDistributionPoints)
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

func findDevice(ctx context.Context, deviceName string, ipType IPType) (*core.Device, error) {
	deviceID, err := findDeviceByNameWithRetry(deviceName, 3)
	if err != nil {
		return nil, err
	}
	client := core.NewClient()
	discoveryCfg := getDiscoveryConfiguration(ipType)
	device, err := client.GetDeviceByMulticast(ctx, deviceID, discoveryCfg)
	if err != nil {
		return nil, err
	}
	return device, nil
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
		isIpV6 := strings.Contains(addr.GetHostname(), ":")
		if isIpV6 && ipType == IP6 {
			return addr.GetHostname(), nil
		}
		if !isIpV6 && ipType == IP4 {
			return addr.GetHostname(), nil
		}
	}
	return "", fmt.Errorf("ipType(%v) not found in %v", ipType, eps)
}

func FindDeviceIP(ctx context.Context, deviceName string, ipType IPType) (string, error) {
	device, err := findDevice(ctx, deviceName, ipType)
	if err != nil {
		return "", err
	}
	defer func() {
		if errC := device.Close(ctx); errC != nil {
			fmt.Printf("cannot close device: %v\n", errC)
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

func FindDeviceEndpoints(ctx context.Context, deviceName string, ipType IPType) (schema.Endpoints, error) {
	device, err := findDevice(ctx, deviceName, ipType)
	if err != nil {
		return nil, err
	}
	defer func() {
		if errC := device.Close(ctx); errC != nil {
			fmt.Printf("cannot close device: %v\n", errC)
		}
	}()
	return device.GetEndpoints(), nil
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

func MakeNonDiscoverableSwitchData() map[string]interface{} {
	// create a non-discoverable switch resource
	overrideParameters := map[string]interface{}{
		"if": []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
		"p": map[interface{}]interface{}{
			"bm": uint64(schema.Observable), // let's make the resource only observable
		},
	}
	return MakeSwitchResourceData(overrideParameters)
}

func DefaultDevsimResourceLinks() schema.ResourceLinks {
	res := TestDevsimResources
	res = append(res, TestDevsimSecResources...)
	res = append(res, TestDevsimPrivateResources...)
	return res
}

func CheckResourceLinks(t *testing.T, expected, actual schema.ResourceLinks) {
	require.Len(t, actual, len(expected))
	expLinks := make(map[string]bool)
	for _, l := range expected {
		expLinks[l.Href] = true
	}
	for _, l := range actual {
		if _, ok := expLinks[l.Href]; ok {
			delete(expLinks, l.Href)
		} else {
			require.FailNowf(t, "unexpected link", l.Href)
		}
	}
	require.Empty(t, expLinks)
}

func CloudSID() string {
	return os.Getenv("CLOUD_SID")
}

func GetRootCApem(t *testing.T) []byte {
	certPath := os.Getenv("ROOT_CA_CRT")
	require.NotEmpty(t, certPath)
	data, err := os.ReadFile(filepath.Clean(certPath))
	require.NoError(t, err)
	return data
}

func GetRootCA(t *testing.T) []*x509.Certificate {
	certPem := GetRootCApem(t)
	cas, err := pkgX509.ParsePemCertificates(certPem)
	require.NoError(t, err)
	return cas
}

func GetMfgCertificate(t *testing.T) tls.Certificate {
	ca, err := tls.X509KeyPair(MfgCert, MfgKey)
	require.NoError(t, err)
	return ca
}

func GetCoapCertificate(t *testing.T) tls.Certificate {
	certPath := os.Getenv("COAP_CRT")
	require.NotEmpty(t, certPath)
	keyPath := os.Getenv("COAP_KEY")
	require.NotEmpty(t, keyPath)
	ca, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)
	return ca
}

package test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/kit/security"
	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/pkg/net/coap"
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

type IdentityCertificateSigner struct {
	caCert         []*x509.Certificate
	caKey          crypto.PrivateKey
	validNotBefore time.Time
	validNotAfter  time.Time
}

func NewIdentityCertificateSigner(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore time.Time, validNotAfter time.Time) core.CertificateSigner {
	return &IdentityCertificateSigner{caCert: caCert, caKey: caKey, validNotBefore: validNotBefore, validNotAfter: validNotAfter}
}

func (s *IdentityCertificateSigner) Sign(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	csrBlock, _ := pem.Decode(csr)
	if csrBlock == nil {
		err = fmt.Errorf("pem not found")
		return
	}

	certificateRequest, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return
	}

	err = certificateRequest.CheckSignature()
	if err != nil {
		return
	}

	notBefore := s.validNotBefore
	notAfter := s.validNotAfter
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return
	}

	template := x509.Certificate{
		SerialNumber:       serialNumber,
		NotBefore:          notBefore,
		NotAfter:           notAfter,
		Subject:            certificateRequest.Subject,
		PublicKeyAlgorithm: certificateRequest.PublicKeyAlgorithm,
		PublicKey:          certificateRequest.PublicKey,
		SignatureAlgorithm: s.caCert[0].SignatureAlgorithm,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		UnknownExtKeyUsage: []asn1.ObjectIdentifier{coap.ExtendedKeyUsage_IDENTITY_CERTIFICATE},
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	if len(s.caCert) == 0 {
		return nil, fmt.Errorf("cannot sign with empty signer CA certificates")
	}
	signedCsr, err = x509.CreateCertificate(rand.Reader, &template, s.caCert[0], certificateRequest.PublicKey, s.caKey)
	if err != nil {
		return
	}
	return security.CreatePemChain(s.caCert, signedCsr)
}

type IPType int

const (
	ANY IPType = 0
	IP4 IPType = 1
	IP6 IPType = 2
)

func FindDeviceIP(ctx context.Context, deviceName string, ipType IPType) (string, error) {
	deviceID := MustFindDeviceByName(deviceName)
	client := core.NewClient()

	discoveryCfg := core.DefaultDiscoveryConfiguration()
	switch ipType {
	case IP4:
		discoveryCfg.MulticastAddressUDP6 = nil
	case IP6:
		discoveryCfg.MulticastAddressUDP4 = nil
	}

	device, err := client.GetDeviceByMulticast(ctx, deviceID, discoveryCfg)
	if err != nil {
		return "", err
	}
	defer device.Close(ctx)

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

func MustFindDeviceIP(name string, ipType IPType) (ip string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ip, err := FindDeviceIP(ctx, name, ipType)
	if err == nil {
		return ip
	}
	panic(err)
}

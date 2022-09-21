package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	enjson "encoding/json"
	"fmt"
	"time"

	"github.com/plgd-dev/device/client"
	local "github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/kit/v2/codec/json"
)

const Timeout = time.Second * 10

type (
	// OCF Client for working with devices
	OCFClient struct {
		client  *local.Client
		devices []local.DeviceDetails
	}
)

type SetupSecureClient struct {
	ca      []*x509.Certificate
	mfgCA   []*x509.Certificate
	mfgCert tls.Certificate
}

func (c *SetupSecureClient) GetManufacturerCertificate() (tls.Certificate, error) {
	if c.mfgCert.PrivateKey == nil {
		return c.mfgCert, fmt.Errorf("private key not set")
	}
	return c.mfgCert, nil
}

func (c *SetupSecureClient) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.mfgCA) == 0 {
		return nil, fmt.Errorf("certificate authority not set")
	}
	return c.mfgCA, nil
}

func (c *SetupSecureClient) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.ca) == 0 {
		return nil, fmt.Errorf("certificate authorities not set")
	}
	return c.ca, nil
}

// Discover devices in the local area
func (c *OCFClient) Discover(discoveryTimeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()
	res, err := c.client.GetDevices(ctx, local.WithError(func(error) {}))
	if err != nil {
		return "", err
	}

	deviceInfo := []interface{}{}
	devices := []local.DeviceDetails{}
	for _, device := range res {
		if device.IsSecured && device.Ownership != nil {
			devices = append(devices, device)
			devInfo := map[string]interface{}{
				"id": device.ID, "name": device.Ownership.Name, "owned": device.Ownership.Owned,
				"ownerID": device.Ownership.OwnerID, "details": device.Details,
			}
			deviceInfo = append(deviceInfo, devInfo)
		}
	}
	c.devices = devices

	devicesJSON, err := enjson.MarshalIndent(deviceInfo, "", "    ")
	if err != nil {
		return "", err
	}
	return string(devicesJSON), nil
}

// Observe device online/offline status
func (c *OCFClient) ObserveDevices() {

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Minute)
	defer cancel()
    c.client.GetDeviceByIP(ctx, "192.168.197.27") 
    fmt.Println("starting observation")
    
	h := makeDevicesObservationHandler()
	_, err := c.client.ObserveDevices(ctx, h)
    if err != nil {
        fmt.Println(err)
    }
    finished := false

	for !finished {
		select {
		case devs := <-h.devs:
            fmt.Println(devs)
            continue
		case <-ctx.Done():
            fmt.Println("Timeout reached")
            finished = true
		}
	}
}

func makeDevicesObservationHandler() *devicesObservationHandler {
	return &devicesObservationHandler{devs: make(chan client.DevicesObservationEvent, 100)}
}

type devicesObservationHandler struct {
	devs chan client.DevicesObservationEvent
}

func (h *devicesObservationHandler) Handle(ctx context.Context, body client.DevicesObservationEvent) error {
	h.devs <- body
	return nil
}

func (h *devicesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *devicesObservationHandler) OnClose() {
	fmt.Println("devices observation was closed")
}
// OwnDevice transfers the ownership of the device to user represented by the token
func (c *OCFClient) OwnDevice(deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.client.OwnDevice(ctx, deviceID, local.WithOTMs([]client.OTMType{client.OTMType_JustWorks}))
}

// Get all resource Info of the device
func (c *OCFClient) GetResources(deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	_, links, err := c.client.GetRefDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	resourcesInfo := []map[string]interface{}{}
	for _, link := range links {
		info := map[string]interface{}{"href": link.Href} // , "rt":link.ResourceTypes, "if":link.Interfaces}
		resourcesInfo = append(resourcesInfo, info)
	}

	linksJSON, err := enjson.MarshalIndent(resourcesInfo, "", "    ")
	if err != nil {
		return "", err
	}
	return string(linksJSON), nil
}

// Get a resource Info of the device
func (c *OCFClient) GetResource(deviceID, href string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	var got interface{} // map[string]interface{}
	opts := []local.GetOption{local.WithInterface(interfaces.OC_IF_BASELINE)}
	err := c.client.GetResource(ctx, deviceID, href, &got, opts...)
	if err != nil {
		return "", err
	}

	var resourceJSON bytes.Buffer
	resourceBytes, err := json.Encode(got)
	if err != nil {
		return "", err
	}
	err = enjson.Indent(&resourceJSON, resourceBytes, "", "    ")
	if err != nil {
		return "", err
	}
	return resourceJSON.String(), nil
}

// Update a resource of the device
func (c *OCFClient) UpdateResource(deviceID string, href string, data interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	opts := []local.UpdateOption{local.WithInterface(interfaces.OC_IF_RW)}
	return c.client.UpdateResource(ctx, deviceID, href, data, nil, opts...)
}

// DisownDevice removes the current ownership
func (c *OCFClient) DisownDevice(deviceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.client.DisownDevice(ctx, deviceID)
}

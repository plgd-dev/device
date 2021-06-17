package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	enjson "encoding/json"
	"fmt"
	"time"

	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/sdk/local"
)

const Timeout = time.Second * 10

type (
	// OCF Client for working with devices
	OCFClient struct {
		client        *local.Client
		devices	  []local.DeviceDetails
	}
)

type SetupSecureClient struct {
	ca      []*x509.Certificate
	mfgCA   []*x509.Certificate
	mfgCert tls.Certificate
}

func (c *SetupSecureClient) GetManufacturerCertificate() (tls.Certificate, error) {
	if c.mfgCert.PrivateKey == nil {
		return c.mfgCert, fmt.Errorf("not set")
	}
	return c.mfgCert, nil
}

func (c *SetupSecureClient) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.mfgCA) == 0 {
		return nil, fmt.Errorf("not set")
	}
	return c.mfgCA, nil
}

func (c *SetupSecureClient) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.ca) == 0 {
		return nil, fmt.Errorf("not set")
	}
	return c.ca, nil
}

// Discover devices in the local area
func (c *OCFClient) Discover(timeoutSeconds int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	res, err := c.client.GetDevices(ctx)
	if err != nil {
		return "", err
	}

	deviceInfo := []interface{}{}
	devices := []local.DeviceDetails{}
	for _, device := range res {
		if device.IsSecured {
			devices = append(devices, device)
			deviceInfo = append(deviceInfo, device.Details)
		}
	}
	c.devices = devices

	devicesJSON, err := enjson.MarshalIndent(deviceInfo, "", "    ")
	//devicesJSON, err := enjson.MarshalIndent(devices, "", "    ")
	if err != nil {
		return "", err
	}
	return string(devicesJSON), nil
}

// OwnDevice transfers the ownership of the device to user represented by the token
func (c *OCFClient) OwnDevice(deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.client.OwnDevice(ctx, deviceID, local.WithOTM(local.OTMType_JustWorks))
}

// Get all resource Info of the device
func (c *OCFClient) GetResources(deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	_, links, err := c.client.GetRefDevice(ctx, deviceID)

	resourcesInfo := []map[string]interface{}{}
	for _, link := range links {
		info := map[string]interface{}{"Href":link.Href} //, "rt":link.ResourceTypes, "if":link.Interfaces}
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
	opts := []local.GetOption{local.WithInterface("oic.if.baseline")}
	err := c.client.GetResource(ctx, deviceID, href, &got, opts...)
	if err != nil {
		return "", err
	}

	var resourceJSON bytes.Buffer
	resourceBytes, err := json.Encode(got)
	err = enjson.Indent(&resourceJSON, resourceBytes, "", "    ")
	if err != nil {
		return "", err
	}
	return string(resourceJSON.Bytes()), nil
}

// Update a resource of the device
func (c *OCFClient) UpdateResource(deviceID string, href string, data map[string]interface{}) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	var got interface{}
	opts := []local.UpdateOption{local.WithInterface("oic.if.rw")}
	err := c.client.UpdateResource(ctx, deviceID, href, data, &got, opts...)
	if err != nil {
		return "", err
	}

	var resourceJSON bytes.Buffer
	resourceBytes, err := json.Encode(got)
	err = enjson.Indent(&resourceJSON, resourceBytes, "", "    ")
	if err != nil {
		return "", err
	}
	return string(resourceJSON.Bytes()), nil

}

// DisownDevice removes the current ownership
func (c *OCFClient) DisownDevice(deviceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.client.DisownDevice(ctx, deviceID)
}

package local

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/go-ocf/kit/net/coap"

	kitNet "github.com/go-ocf/kit/net"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

// DisownDevice remove ownership of device
func (c *Client) DisownDevice(
	ctx context.Context,
	deviceID string,
) error {
	const errMsg = "cannot disown device %v: %v"

	client, err := c.ownDeviceFindClient(ctx, deviceID, resource.DiscoverAllDevices)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	ownership := client.GetOwnership()
	if !ownership.Owned {
		return fmt.Errorf(errMsg, deviceID, "device is not owned")
	}

	deviceClient, err := c.GetDevice(ctx, deviceID, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	links := deviceClient.GetResourceLinks()
	if len(links) == 0 {
		return fmt.Errorf(errMsg, deviceID, "device links are empty")
	}
	var tlsAddr kitNet.Addr
	var tlsAddrFound bool
	for _, link := range deviceClient.GetResourceLinks() {
		if tlsAddr, err = link.GetTCPSecureAddr(); err == nil {
			tlsAddrFound = true
			break
		}
	}
	//tlsAddr, err := deviceClient.GetResourceLinks()[0].GetTCPSecureAddr()
	if !tlsAddrFound {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get tcp secure address: not found"))
	}
	cert, err := c.GetCertificate()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get identity certificate: %v", err))
	}
	cas, err := c.GetCertificateAuthorities()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get identity certificate: %v", err))
	}
	tlsConn, err := coap.DialTcpTls(ctx, tlsAddr.String(), cert, cas, func(*x509.Certificate) error { return nil })
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot create connection: %v", err))
	}
	defer tlsConn.Close()

	sdkID, err := c.GetSdkDeviceID()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	if ownership.DeviceOwner != sdkID {
		return fmt.Errorf(errMsg, deviceID, fmt.Sprintf("device is owned by %v, not by %v", ownership.DeviceOwner, sdkID))
	}

	setResetProvisionState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RESET,
		},
	}

	err = tlsConn.UpdateResource(ctx, "/oic/sec/pstat", setResetProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	defer c.CloseConnections(deviceClient.GetDeviceLinks())

	return nil
}

package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	enjson "encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/sdk/local"
	"github.com/plgd-dev/sdk/test"
)

const Timeout = time.Second * 8

type (
	// OCFClient for working with devices
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

func NewSDKClient() (*local.Client, error) {
	mfgTrustedCABlock, _ := pem.Decode(MfgTrustedCA)
	if mfgTrustedCABlock == nil {
		return nil, fmt.Errorf("mfgTrustedCABlock is empty")
	}
	mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
	if err != nil {
		return nil, err
	}
	mfgCert, err := tls.X509KeyPair(MfgCert, MfgKey)
	if err != nil {
		return nil, fmt.Errorf("cannot X509KeyPair: %w", err)
	}

	identityTrustedCABlock, _ := pem.Decode(IdentityTrustedCA)
	if identityTrustedCABlock == nil {
		return nil, fmt.Errorf("identityTrustedCABlock is empty")
	}
	identityTrustedCACert, err := x509.ParseCertificates(identityTrustedCABlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("cannot parse cert: %w", err)
	}

	cfg := local.Config{
		DisablePeerTCPSignalMessageCSMs: true,
		DeviceOwnershipSDK: &local.DeviceOwnershipSDKConfig{
			ID:      CertIdentity,
			Cert:    string(IdentityIntermediateCA),
			CertKey: string(IdentityIntermediateCAKey),
		},
	}

	client, err := local.NewClientFromConfig(&cfg, &SetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
		ca:      append(identityTrustedCACert),
	}, test.NewIdentityCertificateSigner, func(err error) { fmt.Print(err) },
	)
	if err != nil {
		return nil, err
	}
	err = client.Initialization(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}


// Initialize creates and initializes new local client
func (c *OCFClient) Initialize() error {

	localClient, err := NewSDKClient()
	if err != nil {
		return err
	}

	c.client = localClient
	return nil
}

func (c *OCFClient) Close() error {
	if c.client != nil {
		return c.client.Close(context.Background())
	}
	return nil
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

	//devicesJSON, err := json.Encode(deviceInfo)
	devicesJSON, err := enjson.MarshalIndent(deviceInfo, "", "    ")
	if err != nil {
		return "", err
	}
	return string(devicesJSON), nil
}

// OwnDevice transfers the ownership of the device to user represented by the token
func (c *OCFClient) OwnDevice(deviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

	//linksJSON, err := json.Encode(resourcesInfo)
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

	resourceJSON, err := json.Encode(got)
	//resourceJSON, err := enjson.MarshalIndent(string(resourceBytes), "", "    ")
	if err != nil {
		return "", err
	}
	return string(resourceJSON), nil
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

	resourceJSON, err := json.Encode(got)
	if err != nil {
		return "", err
	}
	return string(resourceJSON), nil

}

// DisownDevice removes the current ownership
func (c *OCFClient) DisownDevice(deviceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return c.client.DisownDevice(ctx, deviceID)
}


var (
	CertIdentity = "00000000-0000-0000-0000-000000000001"

	MfgCert = []byte(`-----BEGIN CERTIFICATE-----
MIIB9zCCAZygAwIBAgIRAOwIWPAt19w7DswoszkVIEIwCgYIKoZIzj0EAwIwEzER
MA8GA1UEChMIVGVzdCBPUkcwHhcNMTkwNTAyMjAwNjQ4WhcNMjkwMzEwMjAwNjQ4
WjBHMREwDwYDVQQKEwhUZXN0IE9SRzEyMDAGA1UEAxMpdXVpZDpiNWEyYTQyZS1i
Mjg1LTQyZjEtYTM2Yi0wMzRjOGZjOGVmZDUwWTATBgcqhkjOPQIBBggqhkjOPQMB
BwNCAAQS4eiM0HNPROaiAknAOW08mpCKDQmpMUkywdcNKoJv1qnEedBhWne7Z0jq
zSYQbyqyIVGujnI3K7C63NRbQOXQo4GcMIGZMA4GA1UdDwEB/wQEAwIDiDAzBgNV
HSUELDAqBggrBgEFBQcDAQYIKwYBBQUHAwIGCCsGAQUFBwMBBgorBgEEAYLefAEG
MAwGA1UdEwEB/wQCMAAwRAYDVR0RBD0wO4IJbG9jYWxob3N0hwQAAAAAhwR/AAAB
hxAAAAAAAAAAAAAAAAAAAAAAhxAAAAAAAAAAAAAAAAAAAAABMAoGCCqGSM49BAMC
A0kAMEYCIQDuhl6zj6gl2YZbBzh7Th0uu5izdISuU/ESG+vHrEp7xwIhANCA7tSt
aBlce+W76mTIhwMFXQfyF3awWIGjOcfTV8pU
-----END CERTIFICATE-----
`)

	MfgKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMPeADszZajrkEy4YvACwcbR0pSdlKG+m8ALJ6lj/ykdoAoGCCqGSM49
AwEHoUQDQgAEEuHojNBzT0TmogJJwDltPJqQig0JqTFJMsHXDSqCb9apxHnQYVp3
u2dI6s0mEG8qsiFRro5yNyuwutzUW0Dl0A==
-----END EC PRIVATE KEY-----
`)

	MfgTrustedCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBaTCCAQ+gAwIBAgIQR33gIB75I7Vi/QnMnmiWvzAKBggqhkjOPQQDAjATMREw
DwYDVQQKEwhUZXN0IE9SRzAeFw0xOTA1MDIyMDA1MTVaFw0yOTAzMTAyMDA1MTVa
MBMxETAPBgNVBAoTCFRlc3QgT1JHMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
xbwMaS8jcuibSYJkCmuVHfeV3xfYVyUq8Iroz7YlXaTayspW3K4hVdwIsy/5U+3U
vM/vdK5wn2+NrWy45vFAJqNFMEMwDgYDVR0PAQH/BAQDAgEGMBMGA1UdJQQMMAoG
CCsGAQUFBwMBMA8GA1UdEwEB/wQFMAMBAf8wCwYDVR0RBAQwAoIAMAoGCCqGSM49
BAMCA0gAMEUCIBWkxuHKgLSp6OXDJoztPP7/P5VBZiwLbfjTCVRxBvwWAiEAnzNu
6gKPwtKmY0pBxwCo3NNmzNpA6KrEOXE56PkiQYQ=
-----END CERTIFICATE-----
`)
	MfgTrustedCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICzfC16AqtSv3wt+qIbrgM8dTqBhHANJhZS5xCpH6P2roAoGCCqGSM49
AwEHoUQDQgAExbwMaS8jcuibSYJkCmuVHfeV3xfYVyUq8Iroz7YlXaTayspW3K4h
VdwIsy/5U+3UvM/vdK5wn2+NrWy45vFAJg==
-----END EC PRIVATE KEY-----
`)

	IdentityTrustedCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBaDCCAQ6gAwIBAgIRANpzWRKheR25RH0CgYYwLzQwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTEzMTA1M1oYDzIxMTkwNjI1MTMxMDUz
WjARMQ8wDQYDVQQDEwZSb290Q0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASQ
TLfEiNgEfqyWmtW1RV9UKgxsMddrNlYFt/+ZpqaJpBQ+hvtGwJenLEv5jzeEcMXr
gOR4EwjjJSzELk6IibC+o0UwQzAOBgNVHQ8BAf8EBAMCAQYwEwYDVR0lBAwwCgYI
KwYBBQUHAwEwDwYDVR0TAQH/BAUwAwEB/zALBgNVHREEBDACggAwCgYIKoZIzj0E
AwIDSAAwRQIhAOUfsOKyjIgYmDd2G46ge+PEPAZ9DS67Q5RjJvLk/lf3AiA6yMxJ
msmj2nz8VeEkxpKq3gYwJUdJ9jMklTzP+Dcenw==
-----END CERTIFICATE-----
`)
	IdentityTrustedCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIFe+pAuS4dEt1gRZ6Mq1xrgkEHxL191AFzEsNNvTEWOYoAoGCCqGSM49
AwEHoUQDQgAEkEy3xIjYBH6slprVtUVfVCoMbDHXazZWBbf/maamiaQUPob7RsCX
pyxL+Y83hHDF64DkeBMI4yUsxC5OiImwvg==
-----END EC PRIVATE KEY-----
`)
	IdentityIntermediateCA = []byte(`
-----BEGIN CERTIFICATE-----
MIIBczCCARmgAwIBAgIRANntjEpzu9krzL0EG6fcqqgwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTIwMzczOVoYDzIxMTkwNjI1MjAzNzM5
WjAZMRcwFQYDVQQDEw5JbnRlcm1lZGlhdGVDQTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABKw1/6WHFcWtw67hH5DzoZvHgA0suC6IYLKms4IP/pds9wU320eDaENo
5860TOyKrGn7vW/cj/OVe2Dzr4KSFVijSDBGMA4GA1UdDwEB/wQEAwIBBjATBgNV
HSUEDDAKBggrBgEFBQcDATASBgNVHRMBAf8ECDAGAQH/AgEAMAsGA1UdEQQEMAKC
ADAKBggqhkjOPQQDAgNIADBFAiEAgPtnYpgwxmPhN0Mo8VX582RORnhcdSHMzFjh
P/li1WwCIFVVWBOrfBnTt7A6UfjP3ljAyHrJERlMauQR+tkD/aqm
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBaDCCAQ6gAwIBAgIRANpzWRKheR25RH0CgYYwLzQwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTEzMTA1M1oYDzIxMTkwNjI1MTMxMDUz
WjARMQ8wDQYDVQQDEwZSb290Q0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASQ
TLfEiNgEfqyWmtW1RV9UKgxsMddrNlYFt/+ZpqaJpBQ+hvtGwJenLEv5jzeEcMXr
gOR4EwjjJSzELk6IibC+o0UwQzAOBgNVHQ8BAf8EBAMCAQYwEwYDVR0lBAwwCgYI
KwYBBQUHAwEwDwYDVR0TAQH/BAUwAwEB/zALBgNVHREEBDACggAwCgYIKoZIzj0E
AwIDSAAwRQIhAOUfsOKyjIgYmDd2G46ge+PEPAZ9DS67Q5RjJvLk/lf3AiA6yMxJ
msmj2nz8VeEkxpKq3gYwJUdJ9jMklTzP+Dcenw==
-----END CERTIFICATE-----
`)
	IdentityIntermediateCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIPF4DPvFeiRL1G0ROd6MosoUGnvIG/2YxH0CbHwnLKxqoAoGCCqGSM49
AwEHoUQDQgAErDX/pYcVxa3DruEfkPOhm8eADSy4Lohgsqazgg/+l2z3BTfbR4No
Q2jnzrRM7Iqsafu9b9yP85V7YPOvgpIVWA==
-----END EC PRIVATE KEY-----
`)
	IdentityCert = []byte(`-----BEGIN CERTIFICATE-----
MIIBsTCCAVagAwIBAgIQaxAoemzXSnFWCq/DmVwQIDAKBggqhkjOPQQDAjAZMRcw
FQYDVQQDEw5JbnRlcm1lZGlhdGVDQTAgFw0xOTA3MTkyMDM3NTFaGA8yMTE5MDYy
NTIwMzc1MVowNDEyMDAGA1UEAxMpdXVpZDowMDAwMDAwMC0wMDAwLTAwMDAtMDAw
MC0wMDAwMDAwMDAwMDEwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAS/gWdMe96z
qsKMOfWsGJtH0wQCRYcwbu0dr+IkQv4/tSv+wO0EVhfvaIAr8lM2xZ6X+uGMcg/Y
muqOL/nFhadlo2MwYTAOBgNVHQ8BAf8EBAMCA4gwMwYDVR0lBCwwKgYIKwYBBQUH
AwEGCCsGAQUFBwMCBggrBgEFBQcDAQYKKwYBBAGC3nwBBjAMBgNVHRMBAf8EAjAA
MAwGA1UdEQQFMAOCATowCgYIKoZIzj0EAwIDSQAwRgIhAJwukCJJtkbgrgrS96uR
RILQxW0aF8K6+5j+CBeEkwYNAiEAguOX+W1WEu1gAf6pIxMOIF83/X4adJd4KEYs
7gMgO3Y=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBczCCARmgAwIBAgIRANntjEpzu9krzL0EG6fcqqgwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTIwMzczOVoYDzIxMTkwNjI1MjAzNzM5
WjAZMRcwFQYDVQQDEw5JbnRlcm1lZGlhdGVDQTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABKw1/6WHFcWtw67hH5DzoZvHgA0suC6IYLKms4IP/pds9wU320eDaENo
5860TOyKrGn7vW/cj/OVe2Dzr4KSFVijSDBGMA4GA1UdDwEB/wQEAwIBBjATBgNV
HSUEDDAKBggrBgEFBQcDATASBgNVHRMBAf8ECDAGAQH/AgEAMAsGA1UdEQQEMAKC
ADAKBggqhkjOPQQDAgNIADBFAiEAgPtnYpgwxmPhN0Mo8VX582RORnhcdSHMzFjh
P/li1WwCIFVVWBOrfBnTt7A6UfjP3ljAyHrJERlMauQR+tkD/aqm
-----END CERTIFICATE-----
`)
	IdentityKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICLgYlcG6V0LbI3IqUENYuVLR2s0Tyqkxz0t1+QP2KVLoAoGCCqGSM49
AwEHoUQDQgAEv4FnTHves6rCjDn1rBibR9MEAkWHMG7tHa/iJEL+P7Ur/sDtBFYX
72iAK/JTNsWel/rhjHIP2Jrqji/5xYWnZQ==
-----END EC PRIVATE KEY-----
`)
)

func main() {
	client := OCFClient{}
	err := client.Initialize()
	if err != nil {
		fmt.Errorf("OCF Client was failed to initialize")
	}

	// Console Input
	scanner(client)
}


func scanner(client OCFClient) {

	scanner := bufio.NewScanner(os.Stdin)
	printMenu()
	var selMenu int64 = 0
	for scanner.Scan() {
		selMenu, _ = strconv.ParseInt(scanner.Text(), 10, 32)
		switch selMenu {
		case 0 :
			printMenu()
			break
		case 1 :
			res, err := client.Discover(20)
			if err != nil {
				println("\nDiscover devices was failed : " + err.Error())
				break
			}
			println("\nDiscovered devices : \n" + res)
			break
		case 2 :
			print("\nInput device ID : ")
			scanner.Scan()
			deviceID := scanner.Text()
			res, err := client.OwnDevice(deviceID)
			if err != nil {
				println("\nTransfer Ownership was failed : " + err.Error())
				break
			}
			println("\nTransfer Ownership of "+deviceID+" was successful  : \n" + res)
			break
		case 3 :
			print("\nInput device ID : ")
			scanner.Scan()
			deviceID := scanner.Text()
			res, err := client.GetResources(deviceID)
			if err != nil {
				println("\nGet Resources was failed : " + err.Error())
				break
			}
			println("\nResources of "+deviceID+" : \n" + res)
			break
		case 4 :
			// Select Device
			print("\nInput device ID : ")
			scanner.Scan()
			deviceID := scanner.Text()
			res, err := client.GetResources(deviceID)
			if err != nil {
				println("\nGet Resources was failed : " + err.Error())
				break
			}
			println("\nResources of "+deviceID+" : \n" + res)

			// Select Resource
			print("\nInput resource href : ")
			scanner.Scan()
			href := scanner.Text()
			aRes, err := client.GetResource(deviceID, href)
			if err != nil {
				println("\nGet Resource was failed : " + err.Error())
				break
			}
			println("\nResource properties of "+deviceID+href+" : \n" + aRes)
			break
		case 5 :
			// Select Device
			print("\nInput device ID : ")
			scanner.Scan()
			deviceID := scanner.Text()
			res, err := client.GetResources(deviceID)
			if err != nil {
				println("\nGet Resources was failed : " + err.Error())
				break
			}
			println("\nResources of "+deviceID+" : \n" + res)

			// Select Resource
			print("\nInput resource href : ")
			scanner.Scan()
			href := scanner.Text()
			aRes, err := client.GetResource(deviceID, href)
			if err != nil {
				println("\nGet Resource was failed : " + err.Error())
				break
			}
			println("\nResource properties of "+deviceID+href+" : \n" + aRes)

			// Select Property
			print("\nInput property name : ")
			scanner.Scan()
			key := scanner.Text()
			// Input Value of the property
			print("\nInput property value : ")
			scanner.Scan()
			value := scanner.Text()

			// Update Property of the Resource
			jsonString := "{\""+key+"\": "+value+"}"
			var data map[string]interface{}
			err = enjson.Unmarshal([]byte(jsonString), &data)
			upRes, err := client.UpdateResource(deviceID, href, data)
			if err != nil {
				println("\nUpdate resource property was failed : " + err.Error())
				break
			}
			println("\nUpdated resource property of "+deviceID+href+" : \n" + upRes)
			break

		case 6 :
			// Select Device
			print("\nInput device ID : ")
			scanner.Scan()
			deviceID := scanner.Text()
			err := client.DisownDevice(deviceID)
			if err != nil {
				println("\nOff-boarding was failed : " + err.Error())
				break
			}
			println("\nOff-boarding "+deviceID+" was successful" )
			break
		case 99 :
			// Close Client
			client.Close()
			os.Exit(0)
			break
		}
		printMenu()
	}
}

func printMenu() {
	fmt.Println("\n#################### OCF Client for D2D ####################")
	fmt.Println("[0] Display this menu")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("[1] Discover devices")
	fmt.Println("[2] Transfer ownership to the device (On-boarding)")
	fmt.Println("[3] Retrieve resources of the device")
	fmt.Println("[4] Retrieve a resource of the device")
	fmt.Println("[5] Update a resource of the device")
	fmt.Println("[6] Reset ownership of the device (Off-boarding)")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("[99] Exit")
	fmt.Println("############################################################")
	fmt.Print("\nSelect menu : ")
}

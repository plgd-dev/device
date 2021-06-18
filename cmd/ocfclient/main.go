package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/plgd-dev/kit/security"
	"github.com/plgd-dev/kit/security/generateCertificate"
	"github.com/plgd-dev/sdk/app"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/plgd-dev/kit/codec/json"
	"github.com/plgd-dev/sdk/local"
	"github.com/plgd-dev/sdk/test"
)

type Options struct {
	CertIdentity string                          `long:"certIdentity"`

	MfgCert     string                           `long:"mfgCert"`
	MfgKey      string                           `long:"mfgKey"`
	MfgTrustCA	 	string                       `long:"mfgTrustCA"`
	MfgTrustCAKey  	string                       `long:"mfgTrustCAKey"`

	IdentityCert string                          `long:"identityCert"`
	IdentityKey string                           `long:"identityKey"`
	IdentityIntermediateCA string                `long:"identityIntermediateCA"`
	IdentityIntermediateCAKey string             `long:"identityIntermediateCAKey"`
	IdentityTrustCA string                       `long:"identityTrustCA"`
	IdentityTrustCAKey string                    `long:"identityTrustCAKey"`
}

func ReadCommandOptions(opts Options) {

	fmt.Println("Usage of OCF Client Options :")
	fmt.Println("    --certIdentity=<Device UUID>             i.e. 00000000-0000-0000-0000-000000000001")
	fmt.Println("    --mfgCert=<Manufacturer Certificate>     i.e. mfg_cert.crt")
	fmt.Println("    --mfgKey=<Manufacturer Private Key>      i.e. mfg_cert.key")
	fmt.Println("    --mfgTrustCA=<Manufacturer Trusted CA Certificate>         i.e. mfg_rootca.crt")
	fmt.Println("    --mfgTrustCAKey=<Manufacturer Trusted CA Private Key>      i.e. mfg_rootca.key")
	fmt.Println("    --identityCert=<Identity Certificate>      i.e. end_cert.crt")
	fmt.Println("    --identityKey=<Identity Certificate>       i.e. end_cert.key")
	fmt.Println("    --identityIntermediateCA=<Identity Intermediate CA Certificate>     i.e. subca_cert.crt")
	fmt.Println("    --identityIntermediateCAKey=<Identity Intermediate CA Private Key>  i.e. subca_cert.key")
	fmt.Println("    --identityTrustCA=<Identity Trusted CA Certificate>     i.e. rootca_cert.crt")
	fmt.Println("    --identityTrustCA=<Identity Trusted CA Private Key>     i.e. rootca_cert.key")
	fmt.Println()

	// Certificate Identity
	if opts.CertIdentity != "" {
		CertIdentity = opts.CertIdentity
	} else {
		opts.CertIdentity = CertIdentity
	}

	// Mfg Certificates
	if opts.MfgTrustCA != "" {
		mfgTrustCA, err := ioutil.ReadFile(opts.MfgTrustCA)
		if err != nil {
			fmt.Println("Reading MfgTrustCA was failed : " + err.Error())
		} else {
			fmt.Println("Reading MfgTrustCA from file (" + opts.MfgTrustCA + ") was successful")
			MfgTrustedCA = mfgTrustCA
		}
	} else {
		filename := "mfg_rootca.crt"
		if !fileExists(filename) {
			err := ioutil.WriteFile(filename, MfgTrustedCA, 0644)
			if err != nil {
				fmt.Println("Writing MfgTrustCA was failed : " + err.Error())
			} else {
				fmt.Println("Writing MfgTrustCA to " + filename + " for sample use ...")
			}
		}
		opts.MfgTrustCA = filename
	}

	if opts.MfgTrustCAKey != "" {
		mfgTrustCAKey, err := ioutil.ReadFile(opts.MfgTrustCAKey)
		if err != nil {
			fmt.Println("Reading MfgTrustCAKey was failed : " + err.Error())
		} else {
			fmt.Println("Reading MfgTrustCAKey from file (" + opts.MfgTrustCAKey + ") was successful")
			MfgTrustedCAKey = mfgTrustCAKey
		}
	} else {
		filename := "mfg_rootca.key"
		if !fileExists(filename) {
			err := ioutil.WriteFile(filename, MfgTrustedCAKey, 0644)
			if err != nil {
				fmt.Println("Writing MfgTrustedCAKey was failed : " + err.Error())
			} else {
				fmt.Println("Writing MfgTrustedCAKey to " + filename + " for sample use ...")
			}
		}
		opts.MfgTrustCAKey = filename
	}

	if opts.MfgCert != "" && opts.MfgKey != ""{
		mfgCert, err := ioutil.ReadFile(opts.MfgCert)
		if err != nil {
			fmt.Println("Reading MfgCert was failed : " + err.Error())
		} else {
			fmt.Println("Reading MfgCert from file (" + opts.MfgCert + ") was successful")
			MfgCert = mfgCert
		}
		mfgKey, err := ioutil.ReadFile(opts.MfgKey)
		if err != nil {
			fmt.Println("Reading MfgKey was failed : " + err.Error())
		} else {
			fmt.Println("Reading MfgKey from file (" + opts.MfgKey+") was successful")
			MfgKey = mfgKey
		}
	}

	if opts.MfgCert == "" || opts.MfgKey == "" {
		outCert := "mfg_cert.crt"
		outKey := "mfg_cert.key"
		err := generateIdentityCertificate(opts.CertIdentity, opts.MfgTrustCA, opts.MfgTrustCAKey, outCert, outKey)
		if err != nil {
			fmt.Println("Generating MfgCert and MfgKey was failed : " + err.Error())
		} else {
			fmt.Println("Generating MfgCert and MfgKey was successful")

			mfgCert, err := ioutil.ReadFile(outCert)
			if err != nil {
				fmt.Println("Reading MfgCert was failed : " + err.Error())
			} else {
				fmt.Println("Reading MfgCert from file (" + outCert + ") was successful")
				MfgCert = mfgCert
			}

			mfgKey, err := ioutil.ReadFile(outKey)
			if err != nil {
				fmt.Println("Reading MfgKey was failed : " + err.Error())
			} else {
				fmt.Println("Reading MfgKey from file (" + outKey+") was successful")
				MfgKey = mfgKey
			}
		}
	}

	// Identity Certificates
	if opts.IdentityIntermediateCA != "" {
		identityIntermediateCA, err := ioutil.ReadFile(opts.IdentityIntermediateCA)
		if err != nil {
			fmt.Println("Reading IdentityIntermediateCA was failed : " + err.Error())
		} else {
			fmt.Println("Reading IdentityIntermediateCA from file (" + opts.IdentityIntermediateCA + ") was successful")
			IdentityIntermediateCA = identityIntermediateCA
		}
	} else {
		filename := "subca_cert.crt"
		if !fileExists(filename) {
			err := ioutil.WriteFile(filename, IdentityIntermediateCA, 0644)
			if err != nil {
				fmt.Println("Writing IdentityIntermediateCA was failed : " + err.Error())
			} else {
				fmt.Println("Writing IdentityIntermediateCA to " + filename + " for sample use ...")
			}
		}
		opts.IdentityIntermediateCA = filename
	}

	if opts.IdentityIntermediateCAKey != "" {
		identityIntermediateCAKey, err := ioutil.ReadFile(opts.IdentityIntermediateCAKey)
		if err != nil {
			fmt.Println("Reading IdentityIntermediateCAKey was failed : " + err.Error())
		} else {
			fmt.Println("Reading IdentityIntermediateCAKey from file (" + opts.IdentityIntermediateCAKey + ") was successful")
			IdentityIntermediateCAKey = identityIntermediateCAKey
		}
	} else {
		filename := "subca_cert.key"
		if !fileExists(filename) {
			err := ioutil.WriteFile(filename, IdentityIntermediateCAKey, 0644)
			if err != nil {
				fmt.Println("Writing IdentityIntermediateCAKey was failed : " + err.Error())
			} else {
				fmt.Println("Writing IdentityIntermediateCAKey to " + filename + " for sample use ...")
			}
		}
		opts.IdentityIntermediateCAKey = filename
	}

	if opts.IdentityTrustCA != "" {
		identityTrustCA, err := ioutil.ReadFile(opts.IdentityTrustCA)
		if err != nil {
			fmt.Println("Reading IdentityTrustCA was failed : " + err.Error())
		} else {
			fmt.Println("Reading IdentityTrustCA from file (" + opts.IdentityTrustCA + ") was successful")
			IdentityTrustedCA = identityTrustCA
		}
	} else {
		filename := "rootca_cert.crt"
		if !fileExists(filename) {
			err := ioutil.WriteFile(filename, IdentityTrustedCA, 0644)
			if err != nil {
				fmt.Println("Writing IdentityTrustCA was failed : " + err.Error())
			} else {
				fmt.Println("Writing IdentityTrustCA to " + filename + " for sample use ...")
			}
		}
		opts.IdentityTrustCA = filename
	}

	if opts.IdentityTrustCAKey != "" {
		identityTrustCAKey, err := ioutil.ReadFile(opts.IdentityTrustCAKey)
		if err != nil {
			fmt.Println("Reading IdentityTrustCAKey was failed : " + err.Error())
		} else {
			fmt.Println("Reading IdentityTrustCAKey from file (" + opts.IdentityTrustCAKey + ") was successful")
			IdentityTrustedCAKey = identityTrustCAKey
		}
	} else {
		filename := "rootca_cert.key"
		if !fileExists(filename) {
			err := ioutil.WriteFile(filename, IdentityTrustedCAKey, 0644)
			if err != nil {
				fmt.Println("Writing IdentityTrustedCAKey was failed : " + err.Error())
			} else {
				fmt.Println("Writing IdentityTrustedCAKey to " + filename + " for sample use ...")
			}
		}
		opts.IdentityTrustCAKey = filename
	}

	if opts.IdentityCert != ""  && opts.IdentityKey != "" {
		identityCert, err := ioutil.ReadFile(opts.IdentityCert)
		if err != nil {
			fmt.Println("Reading IdentityCert was failed : " + err.Error())
		} else {
			fmt.Println("Reading IdentityCert from file (" + opts.IdentityCert + ") was successful")
			IdentityCert = identityCert
		}
		identityKey, err := ioutil.ReadFile(opts.IdentityKey)
		if err != nil {
			fmt.Println("Reading IdentityKey was failed : " + err.Error())
		} else {
			fmt.Println("Reading IdentityKey from file (" + opts.IdentityKey + ") was successful")
			IdentityKey = identityKey
		}
	}

	if opts.IdentityCert == "" || opts.IdentityKey == "" {
		outCert := "end_cert.crt"
		outKey := "end_cert.key"
		err := generateIdentityCertificate(opts.CertIdentity, opts.IdentityIntermediateCA, opts.IdentityIntermediateCAKey, outCert, outKey)
		if err != nil {
			fmt.Println("Generating IdentityCert and IdentityKey was failed : " + err.Error())
		} else {
			fmt.Println("Generating IdentityCert and IdentityKey was successful")

			identityCert, err := ioutil.ReadFile(outCert)
			if err != nil {
				fmt.Println("Reading IdentityCert was failed : " + err.Error())
			} else {
				fmt.Println("Reading IdentityCert from file (" + outCert + ") was successful")
				IdentityCert = identityCert
			}

			identityKey, err := ioutil.ReadFile(outKey)
			if err != nil {
				fmt.Println("Reading IdentityKey was failed : " + err.Error())
			} else {
				fmt.Println("Reading IdentityKey from file (" + outKey + ") was successful")
				IdentityKey = identityKey
			}
		}
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func generateIdentityCertificate(identity, signCert, signKey, outCert, outKey string) error {
	certConfig := generateCertificate.Configuration{}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	signerCert, err := security.LoadX509(signCert)
	if err != nil {
		return err
	}
	signerKey, err := security.LoadX509PrivateKey(signKey)
	if err != nil {
		return err
	}
	cert, err := generateCertificate.GenerateIdentityCert(certConfig, identity, priv, signerCert, signerKey)
	if err != nil {
		return err
	}
	WriteCertOut(outCert, cert)
	if err != nil {
		return err
	}
	WritePrivateKey(outKey, priv)
	if err != nil {
		return err
	}
	return nil
}

func WriteCertOut(filename string, cert []byte) error {
	certOut, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to open %v for writing: %s", filename, err)
	}
	_, err = certOut.Write(cert)
	if err != nil {
		return fmt.Errorf("failed to write %v: %s", filename, err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("error closing %v: %s", filename, err)
	}
	return nil
}

func WritePrivateKey(filename string, priv *ecdsa.PrivateKey) error {
	privBlock, err := pemBlockForKey(priv)
	if err != nil {
		return fmt.Errorf("failed to encode priv key %v for writing: %v", filename, err)
	}

	keyOut, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %v for writing: %v", filename, err)
	}

	if err := pem.Encode(keyOut, privBlock); err != nil {
		return fmt.Errorf("failed to write data to %v: %s", filename, err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("error closing %v: %s", filename, err)
	}
	return nil
}

func pemBlockForKey(k *ecdsa.PrivateKey) (*pem.Block, error) {
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil, err
	}
	return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
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
	}, test.NewIdentityCertificateSigner, func(err error) {},)
	if err != nil {
		return nil, err
	}
	err = client.Initialization(context.Background())
	if err != nil {
		return nil, err
	}

	return client, nil
}

func NewClient() *local.Client {
	appCallback, err := app.NewApp(nil)
	if err != nil {
		panic(err)
	}
	c, err := local.NewClientFromConfig(&local.Config{
		KeepAliveConnectionTimeoutSeconds: 3600,
		ObserverPollingIntervalSeconds:    10,
	}, appCallback, test.NewIdentityCertificateSigner, func(error) {})
	if err != nil {
		panic(err)
	}
	return c
}

func NewSecureClient() (*local.Client, error) {
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
	cfg := local.Config{
		DeviceOwnershipSDK: &local.DeviceOwnershipSDKConfig{
			ID:      CertIdentity,
			Cert:    string(IdentityIntermediateCA),
			CertKey: string(IdentityIntermediateCAKey),
		},
	}

	client, err := local.NewClientFromConfig(&cfg, &SetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
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

func main() {
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		fmt.Println("Parsing command options was failed : " + err.Error())
	}

	// Read Command Options
	ReadCommandOptions(opts)

	client := OCFClient{}
	err = client.Initialize()
	if err != nil {
		fmt.Println("OCF Client was failed to initialize : " + err.Error())
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
			res, err := client.Discover(30)
			if err != nil {
				println("\nDiscovering devices was failed : " + err.Error())
				break
			}
			println("\nDiscovered devices : \n" + res)
			break
		case 2 :
			// Select Device
			print("\nInput device ID : ")
			scanner.Scan()
			deviceID := scanner.Text()
			res, err := client.OwnDevice(deviceID)
			if err != nil {
				println("\nTransferring Ownership was failed : " + err.Error())
				break
			}
			println("\nTransferring Ownership of "+deviceID+" was successful  : \n" + res)
			break
		case 3 :
			// Select Device
			print("\nInput device ID : ")
			scanner.Scan()
			deviceID := scanner.Text()
			res, err := client.GetResources(deviceID)
			if err != nil {
				println("\nGetting Resources was failed : " + err.Error())
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
				println("\nGetting Resources was failed : " + err.Error())
				break
			}
			println("\nResources of "+deviceID+" : \n" + res)

			// Select Resource
			print("\nInput resource href : ")
			scanner.Scan()
			href := scanner.Text()
			aRes, err := client.GetResource(deviceID, href)
			if err != nil {
				println("\nGetting Resource was failed : " + err.Error())
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
				println("\nGetting Resources was failed : " + err.Error())
				break
			}
			println("\nResources of "+deviceID+" : \n" + res)

			// Select Resource
			print("\nInput resource href : ")
			scanner.Scan()
			href := scanner.Text()
			aRes, err := client.GetResource(deviceID, href)
			if err != nil {
				println("\nGetting Resource was failed : " + err.Error())
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
			var data interface{}
			err = json.Decode([]byte(jsonString), &data)
			dataBytes, err := json.Encode(data)
			println("\nProperty data to update : " + string(dataBytes))
			_, err = client.UpdateResource(deviceID, href, data)
			if err != nil {
				println("\nUpdating resource property was failed : " + err.Error())
				break
			}
			println("\nUpdating resource property of "+deviceID+href+" was successful")
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
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("[1] Discover devices")
	fmt.Println("[2] Transfer ownership to the device (On-boarding)")
	fmt.Println("[3] Retrieve resources of the device")
	fmt.Println("[4] Retrieve a resource of the device")
	fmt.Println("[5] Update a resource of the device")
	fmt.Println("[6] Reset ownership of the device (Off-boarding)")
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("[99] Exit")
	fmt.Println("##############################################################")
	fmt.Print("\nSelect menu : ")
}

var (
	CertIdentity = "00000000-0000-0000-0000-000000000001"

	MfgCert = []byte{}

	MfgKey = []byte{}

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
	IdentityCert = []byte{}

	IdentityKey = []byte{}
)

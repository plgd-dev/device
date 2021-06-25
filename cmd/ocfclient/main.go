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
	"time"

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
	fmt.Println("    --certIdentity=<Device UUID>                               i.e. 00000000-0000-0000-0000-000000000001")
	fmt.Println("    --mfgCert=<Manufacturer Certificate>                       i.e. mfg_cert.crt")
	fmt.Println("    --mfgKey=<Manufacturer Private Key>                        i.e. mfg_cert.key")
	fmt.Println("    --mfgTrustCA=<Manufacturer Trusted CA Certificate>         i.e. mfg_rootca.crt")
	fmt.Println("    --mfgTrustCAKey=<Manufacturer Trusted CA Private Key>      i.e. mfg_rootca.key")
	fmt.Println("    --identityCert=<Identity Certificate>                      i.e. end_cert.crt")
	fmt.Println("    --identityKey=<Identity Certificate>                       i.e. end_cert.key")
	fmt.Println("    --identityIntermediateCA=<Identity Intermediate CA Certificate>     i.e. subca_cert.crt")
	fmt.Println("    --identityIntermediateCAKey=<Identity Intermediate CA Private Key>  i.e. subca_cert.key")
	fmt.Println("    --identityTrustCA=<Identity Trusted CA Certificate>        i.e. rootca_cert.crt")
	fmt.Println("    --identityTrustCA=<Identity Trusted CA Private Key>        i.e. rootca_cert.key")
	fmt.Println()

	// Load certificate identity
	if opts.CertIdentity != "" {
		CertIdentity = opts.CertIdentity
	} else {
		opts.CertIdentity = CertIdentity
	}

	// Load mfg root CA certificate and private key
	if opts.MfgTrustCA != "" {
		mfgTrustCA, err := ioutil.ReadFile(opts.MfgTrustCA)
		if err != nil {
			fmt.Println("Unable to read Manufacturer Trust CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Manufacturer Trust CA's Certificate from " + opts.MfgTrustCA + " was successful.")
			MfgTrustedCA = mfgTrustCA
		}
	}
	if opts.MfgTrustCAKey != "" {
		mfgTrustCAKey, err := ioutil.ReadFile(opts.MfgTrustCAKey)
		if err != nil {
			fmt.Println("Unable to read Manufacturer Trust CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Manufacturer Trust CA's Private Key from " + opts.MfgTrustCAKey + " was successful.")
			MfgTrustedCAKey = mfgTrustCAKey
		}
	}

	// Generate mfg root CA certificate and private key if not exists
	if opts.MfgTrustCA == "" || opts.MfgTrustCAKey == "" {
		outCert := "mfg_rootca.crt"
		outKey := "mfg_rootca.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.Subject.CommonName = "TEST Mfg ROOT CA"
			cfg.BasicConstraints.MaxPathLen = -1
			cfg.ValidFrom = "now"
			cfg.ValidFor = 8760 * time.Hour

			err := generateRootCA(cfg, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Manufacturer Trust CA: " + err.Error())
			} else {
				fmt.Println("Generating Manufacturer Trust CA to " + outCert + ", " + outKey + " was successful.")
			}
		}
	}

	// Load mfg certificate and private key
	if opts.MfgCert != "" && opts.MfgKey != ""{
		mfgCert, err := ioutil.ReadFile(opts.MfgCert)
		if err != nil {
			fmt.Println("Unable to read Manufacturer Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Manufacturer Certificate from " + opts.MfgCert + " was successful.")
			MfgCert = mfgCert
		}
		mfgKey, err := ioutil.ReadFile(opts.MfgKey)
		if err != nil {
			fmt.Println("Unable to read Manufacturer Certificate's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Manufacturer Certificate's Private Key from " + opts.MfgKey+" was successful.")
			MfgKey = mfgKey
		}
	}

	// Generate mfg certificate and private key if not exists
	if opts.MfgCert == "" || opts.MfgKey == "" {
		outCert := "mfg_cert.crt"
		outKey := "mfg_cert.key"
		signerCert := "mfg_rootca.crt"
		signerKey := "mfg_rootca.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.ValidFrom = "now"
			cfg.ValidFor = 8760 * time.Hour

			err := generateIdentityCertificate(cfg, opts.CertIdentity, signerCert, signerKey, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Manufacturer Certificate: " + err.Error())
			} else {
				fmt.Println("Generating Manufacturer Certificate to " + outCert + ", " + outKey + " was successful.")
			}
		}
	}

	// Load identity trust CA certificate and private key
	if opts.IdentityTrustCA != "" {
		identityTrustCA, err := ioutil.ReadFile(opts.IdentityTrustCA)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Certificate from " + opts.IdentityTrustCA + " was successful.")
			IdentityTrustedCA = identityTrustCA
		}
	}

	if opts.IdentityTrustCAKey != "" {
		identityTrustCAKey, err := ioutil.ReadFile(opts.IdentityTrustCAKey)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Private Key from " + opts.IdentityTrustCAKey + " was successful.")
			IdentityTrustedCAKey = identityTrustCAKey
		}
	}

	// Generate identity trust CA certificate and private key if not exists
	if opts.IdentityTrustCA == "" || opts.IdentityTrustCAKey == ""  {
		outCert := "rootca_cert.crt"
		outKey := "rootca_cert.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.Subject.CommonName = "TEST ROOT CA"
			cfg.BasicConstraints.MaxPathLen = -1
			cfg.ValidFrom = "now"
			cfg.ValidFor = 8760 * time.Hour

			err := generateRootCA(cfg, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Identity Trust CA: " + err.Error())
			} else {
				fmt.Println("Generating Identity Trust CA to " + outCert + ", " + outKey + " was successful.")
			}
		}

		identityTrustCA, err := ioutil.ReadFile(opts.IdentityTrustCA)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Certificate from " + opts.IdentityTrustCA + " was successful.")
			IdentityTrustedCA = identityTrustCA
		}
		identityTrustCAKey, err := ioutil.ReadFile(opts.IdentityTrustCAKey)
		if err != nil {
			fmt.Println("Unable to read Identity Trust CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Trust CA's Private Key from " + opts.IdentityTrustCAKey + " was successful.")
			IdentityTrustedCAKey = identityTrustCAKey
		}
	}

	// Load identity intermediate CA certificate and private key
	if opts.IdentityIntermediateCA != "" {
		identityIntermediateCA, err := ioutil.ReadFile(opts.IdentityIntermediateCA)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Certificate from " + opts.IdentityIntermediateCA + " was successful.")
			IdentityIntermediateCA = identityIntermediateCA
		}
	}

	if opts.IdentityIntermediateCAKey != "" {
		identityIntermediateCAKey, err := ioutil.ReadFile(opts.IdentityIntermediateCAKey)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Private Key from " + opts.IdentityIntermediateCAKey + " was successful.")
			IdentityIntermediateCAKey = identityIntermediateCAKey
		}
	}

	// Generate identity intermediate CA certificate and private key if not exists
	if opts.IdentityIntermediateCA == "" || opts.IdentityIntermediateCAKey == "" {
		outCert := "subca_cert.crt"
		outKey := "subca_cert.key"
		signerCert := "rootca_cert.crt"
		signerKey := "rootca_cert.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			cfg := generateCertificate.Configuration{}
			cfg.Subject.Organization = []string{"TEST"}
			cfg.Subject.CommonName = "TEST Intermediate CA"
			cfg.BasicConstraints.MaxPathLen = -1
			cfg.ValidFrom = "now"
			cfg.ValidFor = 8760 * time.Hour
			err := generateIntermediateCertificate(cfg, signerCert, signerKey, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Identity Intermediate CA: " + err.Error())
			} else {
				fmt.Println("Generating Identity Intermediate CA to " + outCert + ", " + outKey + " was successful.")

			}
		}

		identityIntermediateCA, err := ioutil.ReadFile(outCert)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Certificate from " + outCert + " was successful.")
			IdentityIntermediateCA = identityIntermediateCA
		}
		identityIntermediateCAKey, err := ioutil.ReadFile(outKey)
		if err != nil {
			fmt.Println("Unable to read Identity Intermediate CA's Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Intermediate CA's Private Key from " + outKey + " was successful.")
			IdentityIntermediateCAKey = identityIntermediateCAKey
		}
	}

	// Load identity certificate and private key
	if opts.IdentityCert != ""  && opts.IdentityKey != "" {
		identityCert, err := ioutil.ReadFile(opts.IdentityCert)
		if err != nil {
			fmt.Println("Unable to read Identity Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Certificate from " + opts.IdentityCert + " was successful.")
			IdentityCert = identityCert
		}
		identityKey, err := ioutil.ReadFile(opts.IdentityKey)
		if err != nil {
			fmt.Println("Unable to read Identity Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Private Key from " + opts.IdentityKey + " was successful.")
			IdentityKey = identityKey
		}
	}

	// Generate identity certificate and private key if not exists
	if opts.IdentityCert == "" || opts.IdentityKey == "" {
		outCert := "end_cert.crt"
		outKey := "end_cert.key"
		signerCert := "subca_cert.crt"
		signerKey := "subca_cert.key"

		if !fileExists(outCert) || !fileExists(outKey) {
			certConfig := generateCertificate.Configuration{}
			err := generateIdentityCertificate(certConfig, opts.CertIdentity, signerCert, signerKey, outCert, outKey)
			if err != nil {
				fmt.Println("Unable to generate Identity Certificate: " + err.Error())
			} else {
				fmt.Println("Generating Identity Certificate to " + outCert + ", " + outKey + " was successful.")
			}
		}

		identityCert, err := ioutil.ReadFile(outCert)
		if err != nil {
			fmt.Println("Unable to read Identity Certificate: " + err.Error())
		} else {
			fmt.Println("Reading Identity Certificate from " + outCert + " was successful.")
			IdentityCert = identityCert
		}
		identityKey, err := ioutil.ReadFile(outKey)
		if err != nil {
			fmt.Println("Unable to read Identity Private Key: " + err.Error())
		} else {
			fmt.Println("Reading Identity Private Key from " + outKey + " was successful.")
			IdentityKey = identityKey
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

func generateRootCA(certConfig generateCertificate.Configuration, outCert, outKey string) error {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	cert, err := generateCertificate.GenerateRootCA(certConfig, priv)
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

func generateIntermediateCertificate(certConfig generateCertificate.Configuration, signCert, signKey, outCert, outKey string) error {
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
	cert, err := generateCertificate.GenerateIntermediateCA(certConfig, priv, signerCert, signerKey)
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


func generateIdentityCertificate(certConfig generateCertificate.Configuration, identity, signCert, signKey, outCert, outKey string) error {
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

	var cfg local.Config
	if len(IdentityIntermediateCA) > 0 && len(IdentityIntermediateCAKey) > 0 {
		cfg = local.Config{
			//DisablePeerTCPSignalMessageCSMs: true,
			DeviceOwnershipSDK: &local.DeviceOwnershipSDKConfig{
				ID:      CertIdentity,
				Cert:    string(IdentityIntermediateCA),
				CertKey: string(IdentityIntermediateCAKey),
			},
		}
	} else {
		cfg = local.Config{
			//DisablePeerTCPSignalMessageCSMs: true,
			DeviceOwnershipSDK: &local.DeviceOwnershipSDKConfig{
				ID: CertIdentity,
			},
		}
	}

	var setupSecureClient = SetupSecureClient{}
	if len(MfgTrustedCA) > 0 {
		mfgTrustedCABlock, _ := pem.Decode(MfgTrustedCA)
		if mfgTrustedCABlock != nil {
			mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
			if err != nil {
				return nil, err
			}
			setupSecureClient.mfgCA = mfgCA
		} else {
			fmt.Errorf("mfgTrustedCABlock is empty")
		}
	}

	if len(MfgCert) > 0 && len(MfgKey) > 0 {
		mfgCert, err := tls.X509KeyPair(MfgCert, MfgKey)
		if err != nil {
			fmt.Errorf("cannot X509KeyPair: %w", err)
		} else {
			setupSecureClient.mfgCert = mfgCert
		}
	}

	if len(IdentityTrustedCA) > 0 {
		identityTrustedCABlock, _ := pem.Decode(IdentityTrustedCA)
		if identityTrustedCABlock == nil {
			return nil, fmt.Errorf("identityTrustedCABlock is empty")
		}
		identityTrustedCACert, err := x509.ParseCertificates(identityTrustedCABlock.Bytes)
		if err != nil {
			fmt.Errorf("cannot parse cert: %w", err)
		} else {
			setupSecureClient.ca = append(identityTrustedCACert)
		}
	}

	client, err := local.NewClientFromConfig(&cfg, &setupSecureClient, test.NewIdentityCertificateSigner, func(err error) {},)
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

	localClient, err := NewSecureClient()
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

	// Create OCF Client
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
			println("\nTransferring Ownership of "+deviceID+" was successful : \n" + res)
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
			println("\nUpdating resource property of "+deviceID+href+" was successful.")
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
			println("\nOff-boarding "+deviceID+" was successful." )
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
	MfgTrustedCA = []byte{}
	MfgTrustedCAKey = []byte{}

	IdentityTrustedCA = []byte{}
	IdentityTrustedCAKey = []byte{}
	IdentityIntermediateCA = []byte{}
	IdentityIntermediateCAKey = []byte{}
	IdentityCert = []byte{}
	IdentityKey = []byte{}
)

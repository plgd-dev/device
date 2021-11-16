package test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"time"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/acl"
	"github.com/plgd-dev/device/schema/ael"
	"github.com/plgd-dev/device/schema/cloud"
	"github.com/plgd-dev/device/schema/collection"
	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/credential"
	"github.com/plgd-dev/device/schema/csr"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/doxm"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/introspection"
	"github.com/plgd-dev/device/schema/maintenance"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/device/schema/pstat"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/device/schema/roles"
	"github.com/plgd-dev/device/schema/sdi"
	"github.com/plgd-dev/device/schema/sp"
	testTypes "github.com/plgd-dev/device/test/resource/types"
	"github.com/plgd-dev/kit/v2/security"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
)

var (
	DevsimNetBridge string
	DevsimNetHost   string

	TestDevsimResources        []schema.ResourceLink
	TestDevsimPrivateResources []schema.ResourceLink
	TestDevsimSecResources     []schema.ResourceLink
)

const (
	TestResourceSwitchesHref = "/switches"
)

func TestResourceSwitchesInstanceHref(id string) string {
	return TestResourceSwitchesHref + "/" + id
}

func TestResourceLightInstanceHref(id string) string {
	return "/light/" + id
}

func init() {
	DevsimNetHost = "devsim-net-host-" + MustGetHostname()
	DevsimNetBridge = "devsim-net-bridge-" + MustGetHostname()

	TestDevsimResources = []schema.ResourceLink{
		{
			Href:          platform.ResourceURI,
			ResourceTypes: []string{platform.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		},

		{
			Href:          device.ResourceURI,
			ResourceTypes: []string{testTypes.DEVICE_CLOUD, device.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		},

		{
			Href:          resources.ResourceURI,
			ResourceTypes: []string{resources.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_LL, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
		},

		{
			Href:          configuration.ResourceURI,
			ResourceTypes: []string{configuration.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},

		{
			Href:          TestResourceLightInstanceHref("1"),
			ResourceTypes: []string{testTypes.CORE_LIGHT},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},

		{
			Href:          TestResourceSwitchesHref,
			ResourceTypes: []string{collection.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
		},
	}

	TestDevsimPrivateResources = []schema.ResourceLink{
		{
			Href:          cloud.ConfigurationResourceURI,
			ResourceTypes: []string{cloud.ConfigurationResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          introspection.ResourceURI,
			ResourceTypes: []string{introspection.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          maintenance.ResourceURI,
			ResourceTypes: []string{maintenance.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
	}

	TestDevsimSecResources = []schema.ResourceLink{
		{
			Href:          sp.ResourceURI,
			ResourceTypes: []string{sp.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          roles.ResourceURI,
			ResourceTypes: []string{roles.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          pstat.ResourceURI,
			ResourceTypes: []string{pstat.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          doxm.ResourceURI,
			ResourceTypes: []string{doxm.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          csr.ResourceURI,
			ResourceTypes: []string{csr.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          credential.ResourceURI,
			ResourceTypes: []string{credential.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          acl.ResourceURI,
			ResourceTypes: []string{acl.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          ael.ResourceURI,
			ResourceTypes: []string{ael.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
		{
			Href:          sdi.ResourceURI,
			ResourceTypes: []string{sdi.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
	}
}

func loadFileFromEnv(env string) []byte {
	v, err := os.ReadFile(os.Getenv(env))
	if err != nil {
		log.Fatalf("cannot log file from env %v(%v): %v", env, os.Getenv(env), err)
	}
	return v
}

var MfgCert = loadFileFromEnv("MFG_CRT")
var MfgKey = loadFileFromEnv("MFG_KEY")
var RootCACrt = loadFileFromEnv("ROOT_CA_CRT")
var RootCAKey = loadFileFromEnv("ROOT_CA_KEY")
var IdentityIntermediateCA = loadFileFromEnv("INTERMEDIATE_CA_CRT")
var IdentityIntermediateCAKey = loadFileFromEnv("INTERMEDIATE_CA_KEY")

func pemBlockForKey(k *ecdsa.PrivateKey) (*pem.Block, error) {
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil, err
	}
	return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
}

func GenerateIdentityCert(deviceID string) tls.Certificate {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	signerCA, err := security.ParseX509FromPEM(RootCACrt)
	if err != nil {
		log.Fatal(err)
	}
	signerCAKey, err := security.LoadX509PrivateKey(os.Getenv("ROOT_CA_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	data, err := generateCertificate.GenerateIdentityCert(generateCertificate.Configuration{
		ValidFor: time.Hour * 24 * 365,
	}, deviceID, priv, signerCA, signerCAKey)
	if err != nil {
		log.Fatal(err)
	}
	dataKey, err := pemBlockForKey(priv)
	if err != nil {
		log.Fatal(err)
	}

	pair, err := tls.X509KeyPair(data, pem.EncodeToMemory(dataKey))
	if err != nil {
		log.Fatal(err)
	}
	return pair
}

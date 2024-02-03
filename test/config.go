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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/ael"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/collection"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/introspection"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/device/v2/schema/pstat"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/schema/roles"
	"github.com/plgd-dev/device/v2/schema/sdi"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/schema/sp"
	testTypes "github.com/plgd-dev/device/v2/test/resource/types"
)

var (
	DevsimName string

	TestDevsimResources        schema.ResourceLinks
	TestDevsimPrivateResources schema.ResourceLinks
	TestDevsimSecResources     schema.ResourceLinks
)

const (
	TestResourceSwitchesHref = "/switches"
	DockerDevsimName         = "devsim-net-host"
)

func TestResourceSwitchesInstanceHref(id string) string {
	return TestResourceSwitchesHref + "/" + id
}

func TestResourceLightInstanceHref(id string) string {
	return "/light/" + id
}

func init() {
	DevsimName = "devsim-" + MustGetHostname()

	TestDevsimResources = schema.ResourceLinks{
		{
			Href:          platform.ResourceURI,
			ResourceTypes: []string{platform.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		},

		{
			Href:          plgdtime.ResourceURI,
			ResourceTypes: []string{plgdtime.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
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

		{
			Href:          softwareupdate.ResourceURI,
			ResourceTypes: []string{softwareupdate.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},
	}

	TestDevsimPrivateResources = []schema.ResourceLink{
		{
			Href:          cloud.ResourceURI,
			ResourceTypes: []string{cloud.ResourceType},
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
		log.Printf("cannot load file from env %v(%v): %v", env, os.Getenv(env), err)
	}
	return v
}

var (
	MfgCert                   = loadFileFromEnv("MFG_CRT")
	MfgKey                    = loadFileFromEnv("MFG_KEY")
	RootCACrt                 = loadFileFromEnv("ROOT_CA_CRT")
	RootCAKey                 = loadFileFromEnv("ROOT_CA_KEY")
	IdentityIntermediateCA    = loadFileFromEnv("INTERMEDIATE_CA_CRT")
	IdentityIntermediateCAKey = loadFileFromEnv("INTERMEDIATE_CA_KEY")
)

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
	signerCA, err := pkgX509.ParsePemCertificates(RootCACrt)
	if err != nil {
		log.Fatal(err)
	}
	signerCAKey, err := pkgX509.ReadPemEcdsaPrivateKey(os.Getenv("ROOT_CA_KEY"))
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

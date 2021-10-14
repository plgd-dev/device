package test

import (
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/acl"
	"github.com/plgd-dev/device/schema/ael"
	"github.com/plgd-dev/device/schema/cloud"
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
)

var (
	TestSecureDeviceName string
	TestDeviceName       string

	TestDevsimResources        []schema.ResourceLink
	TestDevsimPrivateResources []schema.ResourceLink
	TestDevsimSecResources     []schema.ResourceLink
)

func init() {
	TestDeviceName = "devsim-" + MustGetHostname()
	TestSecureDeviceName = "devsimsec-" + MustGetHostname()

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
			Href:          "/light/1",
			ResourceTypes: []string{testTypes.CORE_LIGHT},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
		},

		{
			Href:          "/light/2",
			ResourceTypes: []string{testTypes.CORE_LIGHT},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
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

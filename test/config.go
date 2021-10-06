package test

import (
	"github.com/plgd-dev/sdk/v2/schema"
	"github.com/plgd-dev/sdk/v2/schema/cloud"
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
			Href:          "/oic/p",
			ResourceTypes: []string{"oic.wk.p"},
			Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
		},

		{
			Href:          "/oic/d",
			ResourceTypes: []string{"oic.d.cloudDevice", "oic.wk.d"},
			Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
		},

		{
			Href:          "/oic/res",
			ResourceTypes: []string{"oic.wk.res"},
			Interfaces:    []string{"oic.if.ll", "oic.if.b", "oic.if.baseline"},
		},

		{
			Href:          "/oc/con",
			ResourceTypes: []string{"oic.wk.con"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},

		{
			Href:          "/light/1",
			ResourceTypes: []string{"core.light"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},

		{
			Href:          "/light/2",
			ResourceTypes: []string{"core.light"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
	}

	TestDevsimPrivateResources = []schema.ResourceLink{
		{
			Href:          cloud.ConfigurationResourceHref,
			ResourceTypes: cloud.ConfigurationResourceTypes,
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oc/wk/introspection",
			ResourceTypes: []string{"oic.wk.introspection"},
			Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
		},
		{
			Href:          "/oic/mnt",
			ResourceTypes: []string{"oic.wk.mnt"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
	}

	TestDevsimSecResources = []schema.ResourceLink{
		{
			Href:          "/oic/sec/sp",
			ResourceTypes: []string{"oic.r.sp"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oic/sec/roles",
			ResourceTypes: []string{"oic.r.roles"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oic/sec/pstat",
			ResourceTypes: []string{"oic.r.pstat"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          schema.DoxmHref,
			ResourceTypes: []string{"oic.r.doxm"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oic/sec/csr",
			ResourceTypes: []string{"oic.r.csr"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oic/sec/cred",
			ResourceTypes: []string{"oic.r.cred"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oic/sec/acl2",
			ResourceTypes: []string{"oic.r.acl2"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oic/sec/ael",
			ResourceTypes: []string{"oic.r.ael"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		{
			Href:          "/oic/sec/sdi",
			ResourceTypes: []string{"oic.r.sdi"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
	}

}

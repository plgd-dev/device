package test

import (
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/cloud"
)

var (
	TestSecureDeviceName string

	TestDevsimPrivateResources []schema.ResourceLink
	TestDevsimSecResources     []schema.ResourceLink
)

func init() {
	TestSecureDeviceName = "devsimsec-" + grpcTest.MustGetHostname()

	TestDevsimPrivateResources = []schema.ResourceLink{
		schema.ResourceLink{
			Href:          cloud.ConfigurationResourceHref,
			ResourceTypes: cloud.ConfigurationResourceTypes,
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oc/wk/introspection",
			ResourceTypes: []string{"oic.wk.introspection"},
			Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oic/mnt",
			ResourceTypes: []string{"oic.wk.mnt"},
			Interfaces:    []string{"oic.if.rw", "oic.if.baseline"},
		},
	}

	TestDevsimSecResources = []schema.ResourceLink{
		schema.ResourceLink{
			Href:          "/oic/sec/sp",
			ResourceTypes: []string{"oic.r.sp"},
			Interfaces:    []string{"oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oic/sec/roles",
			ResourceTypes: []string{"oic.r.roles"},
			Interfaces:    []string{"oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oic/sec/pstat",
			ResourceTypes: []string{"oic.r.pstat"},
			Interfaces:    []string{"oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oic/sec/doxm",
			ResourceTypes: []string{"oic.r.doxm"},
			Interfaces:    []string{"oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oic/sec/csr",
			ResourceTypes: []string{"oic.r.csr"},
			Interfaces:    []string{"oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oic/sec/cred",
			ResourceTypes: []string{"oic.r.cred"},
			Interfaces:    []string{"oic.if.baseline"},
		},
		schema.ResourceLink{
			Href:          "/oic/sec/acl2",
			ResourceTypes: []string{"oic.r.acl2"},
			Interfaces:    []string{"oic.if.baseline"},
		},
	}

}

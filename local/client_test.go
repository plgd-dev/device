package local_test

import (
	"time"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/go-ocf/sdk/local/resource"
)

var testCfg = ocf.Config{
	Protocol: "tcp",
	Resource: resource.Config{
		ResourceHrefExpiration: time.Hour,
		DiscoveryTimeout:       time.Second,
		DiscoveryDelay:         100 * time.Millisecond,

		Errors: func(error) {},
	},
}

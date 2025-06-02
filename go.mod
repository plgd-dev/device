module github.com/plgd-dev/device/v2

go 1.23.0

// use export GOTOOLCHAIN=go1.23.0 before calling go mod tidy to avoid tidy adding
// the toolchain directive with your local go version

require (
	github.com/fredbi/uri v1.1.0
	github.com/fxamacker/cbor/v2 v2.8.0
	github.com/go-json-experiment/json v0.0.0-20240815174924-0599f16bf0e2
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jessevdk/go-flags v1.6.1
	github.com/karrick/tparse/v2 v2.8.2
	github.com/pion/dtls/v3 v3.0.6
	github.com/pion/logging v0.2.3
	github.com/plgd-dev/go-coap/v3 v3.3.7-0.20250702164925-f431046ea1ce
	github.com/plgd-dev/kit/v2 v2.0.0-20211006190727-057b33161b90
	github.com/stretchr/testify v1.10.0
	github.com/ugorji/go/codec v1.2.14
	github.com/web-of-things-open-source/thingdescription-go v0.0.0-20250521114616-3895cda67f5d
	go.uber.org/atomic v1.11.0
	golang.org/x/exp v0.0.0-20250531010427-b6e5de432a8b
	golang.org/x/sync v0.14.0
	google.golang.org/grpc v1.72.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dsnet/golib/memfile v1.0.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pion/transport/v3 v3.0.7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

// last versions for Go 1.23
replace github.com/go-json-experiment/json => github.com/go-json-experiment/json v0.0.0-20240815175050-ebd3a8989ca1

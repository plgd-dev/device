module github.com/plgd-dev/device/v2

go 1.22

toolchain go1.22.0

require (
	github.com/fredbi/uri v1.1.0
	github.com/fxamacker/cbor/v2 v2.7.0
	github.com/go-json-experiment/json v0.0.0-20240815174924-0599f16bf0e2
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jessevdk/go-flags v1.6.1
	github.com/karrick/tparse/v2 v2.8.2
	github.com/pion/dtls/v3 v3.0.2
	github.com/pion/logging v0.2.4
	github.com/plgd-dev/go-coap/v3 v3.3.5
	github.com/plgd-dev/kit/v2 v2.0.0-20211006190727-057b33161b90
	github.com/stretchr/testify v1.10.0
	github.com/ugorji/go/codec v1.2.12
	github.com/web-of-things-open-source/thingdescription-go v0.0.0-20240513190706-79b5f39190eb
	go.uber.org/atomic v1.11.0
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948
	golang.org/x/sync v0.8.0
	google.golang.org/grpc v1.66.2
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
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

// last versions for Go 1.22.0
replace (
	github.com/go-json-experiment/json => github.com/go-json-experiment/json v0.0.0-20240815174924-0599f16bf0e2
	golang.org/x/exp => golang.org/x/exp v0.0.0-20240823005443-9b4947da3948
)

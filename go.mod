module github.com/plgd-dev/device/v2

go 1.22

toolchain go1.22.0

require (
	github.com/fredbi/uri v1.1.0
	github.com/fxamacker/cbor/v2 v2.7.0
	github.com/go-json-experiment/json v0.0.0-20240524174822-2d9f40f7385b
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jessevdk/go-flags v1.6.1
	github.com/karrick/tparse/v2 v2.8.2
	github.com/pion/dtls/v2 v2.2.8-0.20240701035148-45e16a098c47
	github.com/pion/logging v0.2.2
	github.com/plgd-dev/go-coap/v3 v3.3.5-0.20240702113249-de76143e11ea
	github.com/plgd-dev/kit/v2 v2.0.0-20211006190727-057b33161b90
	github.com/stretchr/testify v1.9.0
	github.com/ugorji/go/codec v1.2.12
	github.com/web-of-things-open-source/thingdescription-go v0.0.0-20240513190706-79b5f39190eb
	go.uber.org/atomic v1.11.0
	golang.org/x/sync v0.7.0
	google.golang.org/grpc v1.64.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dsnet/golib/memfile v1.0.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pion/transport/v3 v3.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

// note: github.com/pion/dtls/v2/pkg/net package is not yet available in release branches,
// so we force to the use of the pinned master branch
replace github.com/pion/dtls/v2 => github.com/pion/dtls/v2 v2.2.8-0.20240701035148-45e16a098c47

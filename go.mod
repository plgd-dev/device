module github.com/plgd-dev/device/v2

go 1.20

require (
	github.com/fxamacker/cbor/v2 v2.6.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.1
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jessevdk/go-flags v1.5.0
	github.com/karrick/tparse/v2 v2.8.2
	github.com/pion/dtls/v2 v2.2.8-0.20240201071732-2597464081c8
	github.com/pion/logging v0.2.2
	github.com/plgd-dev/go-coap/v3 v3.3.4-0.20240303135359-ce5dcc5ef1ba
	github.com/plgd-dev/kit/v2 v2.0.0-20211006190727-057b33161b90
	github.com/stretchr/testify v1.9.0
	github.com/ugorji/go/codec v1.2.12
	go.uber.org/atomic v1.11.0
	golang.org/x/sync v0.6.0
	google.golang.org/grpc v1.62.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dsnet/golib/memfile v1.0.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pion/transport/v3 v3.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240304212257-790db918fca8 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)

// note: github.com/pion/dtls/v2/pkg/net package is not yet available in release branches,
// so we force to the use of the pinned master branch
replace github.com/pion/dtls/v2 => github.com/pion/dtls/v2 v2.2.8-0.20240201071732-2597464081c8

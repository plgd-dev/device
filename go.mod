module github.com/plgd-dev/device/v2

go 1.18

require (
	github.com/fxamacker/cbor/v2 v2.5.0
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/google/uuid v1.3.1
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.0-rc.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jessevdk/go-flags v1.5.0
	github.com/karrick/tparse/v2 v2.8.2
	github.com/pion/dtls/v2 v2.2.8-0.20231003120602-c8645455edb4
	github.com/pion/logging v0.2.2
	github.com/plgd-dev/go-coap/v3 v3.1.6-0.20231002093212-ea0ff53e8348
	github.com/plgd-dev/kit/v2 v2.0.0-20211006190727-057b33161b90
	github.com/stretchr/testify v1.8.4
	github.com/ugorji/go/codec v1.2.11
	go.uber.org/atomic v1.11.0
	golang.org/x/sync v0.3.0
	google.golang.org/grpc v1.58.2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dsnet/golib/memfile v1.0.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/pion/transport/v3 v3.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.24.0 // indirect
	golang.org/x/crypto v0.13.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230920204549-e6e6cdab5c13 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

exclude (
	// note: go.uber.org/multierr must be kept at v1.9.0 as long as golang1.18 is supported
	go.uber.org/multierr v1.10.0
	go.uber.org/multierr v1.11.0
	// note: go.uber.org/zap must be kept at v1.24.0 as long as golang1.18 is supported
	go.uber.org/zap v1.25.0
	go.uber.org/zap v1.26.0
)

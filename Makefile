default: test

dep:
	dep ensure -v -vendor-only
.PHONY: dep

secure:
	go generate ./vendor/github.com/go-ocf/kit/security
.PHONY: secure

insecure:
	OCF_INSECURE=TRUE go generate ./vendor/github.com/go-ocf/kit/security
.PHONY: insecure

simulator: simulator.stop
	docker build ./test --network=host -t device-simulator --target service
	docker build ./test -f ./test/Dockerfile.insecure --network=host -t device-simulator-insecure --target service
	docker network create devsimnet
	docker run -d --name devsim --network=devsimnet device-simulator /device-simulator
	docker run -d --name devsim-insecure --network=devsimnet device-simulator-insecure /device-simulator
.PHONY: simulator

simulator.stop:
	docker rm -f devsim || true
	docker rm -f devsim-insecure || true
	docker network rm devsimnet || true
.PHONY: simulator.stop

build: build
	docker build . --network=host -t sdk:build
.PHONY: build

docker: build simulator
	docker run -it --rm --mount type=bind,source="$(shell pwd)",target=/go/src/github.com/go-ocf/sdk --network=devsimnet sdk:build

.PHONY: docker

test: build simulator
	docker run --network=devsimnet sdk:build go test ./...
.PHONY: test


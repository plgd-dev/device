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
	docker build ./test --network=host -t device-simulator
	docker network create devsimnet
	docker run -d --name devsim --network=devsimnet device-simulator /device-simulator
.PHONY: simulator

simulator.stop:
	docker rm -f devsim || true
	docker network rm devsimnet || true
.PHONY: simulator.stop

test: simulator
	docker build . --network=host -t sdk:build
	docker run --network=devsimnet sdk:build go test ./...
.PHONY: test


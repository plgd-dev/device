default: dep insecure simulator test

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
	docker build test/ --network=host -t device-simulator
	docker run -d -t --network=host --name device-simulator device-simulator
.PHONY: simulator

simulator.stop:
	docker rm -f device-simulator || true
.PHONY: simulator.stop

test:
	go test -a ./...
.PHONY: test

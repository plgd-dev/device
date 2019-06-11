default: insecure test

secure:
	go generate ./vendor/github.com/go-ocf/kit/security
.PHONY: secure

insecure:
	OCF_INSECURE=TRUE go generate ./vendor/github.com/go-ocf/kit/security
.PHONY: insecure

test:
	docker build test/ --network=host -t device-simulator
	docker rm -f device-simulator || true
	docker run -d -t --network=host --name device-simulator device-simulator
	go test -a ./...
.PHONY: test

default: insecure test

secure:
	go generate ./vendor/github.com/go-ocf/kit/security
.PHONY: secure

insecure:
	OCF_INSECURE=TRUE go generate ./vendor/github.com/go-ocf/kit/security
.PHONY: insecure

test:
	go test -a ./...
.PHONY: test

SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG = vnext
VERSION_TAG = vnext-$(shell git rev-parse --short=7 --verify HEAD)
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)

default: build

define build-docker-image
	docker build \
		--network=host \
		--tag ocfcloud/$(SERVICE_NAME):$(VERSION_TAG) \
		--tag ocfcloud/$(SERVICE_NAME):$(LATEST_TAG) \
		--target $(1) \
		.
endef

build-testcontainer:
	$(call build-docker-image,build)

build: build-testcontainer

env: clean
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6'; \
	fi
	docker build ./device-simulator --network=host -t device-simulator --target service
	docker build ./device-simulator -f ./device-simulator/Dockerfile.insecure --network=host -t device-simulator-insecure --target service
	docker run -d --name devsimsec  --network=host device-simulator devsimsec-$(SIMULATOR_NAME_SUFFIX)
	docker run -d --name devsim --network=host device-simulator-insecure devsim-$(SIMULATOR_NAME_SUFFIX)

test: env build-testcontainer 
	docker run \
		--network=host \
		--mount type=bind,source="$(shell pwd)",target=/shared \
		ocfcloud/$(SERVICE_NAME):$(VERSION_TAG) \
		go test -p 1 -v ./... -covermode=atomic -coverprofile=/shared/coverage.txt

clean:
	docker rm -f devsimsec || true
	docker rm -f devsim|| true

.PHONY: build-testcontainer build test clean env

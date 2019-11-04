SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG = vnext
VERSION_TAG = vnext-$(shell git rev-parse --short=7 --verify HEAD)
DOCKER_NET = devsimnet-${TRAVIS_JOB_ID}-${TRAVIS_BUILD_ID}

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
	docker network create $(DOCKER_NET)
	docker run -d --name devsim --network=$(DOCKER_NET) device-simulator /device-simulator
	docker run -d --name devsim-insecure --network=$(DOCKER_NET) device-simulator-insecure /device-simulator

test: env build-testcontainer 
	docker run \
		--network=$(DOCKER_NET) \
		--mount type=bind,source="$(shell pwd)",target=/shared \
		ocfcloud/$(SERVICE_NAME):$(VERSION_TAG) \
		go test -v ./... -covermode=atomic -coverprofile=/shared/coverage.txt
	echo "---DEVSIM---"
	docker logs devsim
	echo "---DEVSIM-INSECURE---"
	docker logs devsim-insecure

clean:
	docker rm -f devsim || true
	docker rm -f devsim-insecure || true
	docker network rm $(DOCKER_NET) || true

.PHONY: build-testcontainer build test clean env

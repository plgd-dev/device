SHELL = /bin/bash
SERVICE_NAME = cloud-server-test
VERSION_TAG = vnext-$(shell git rev-parse --short=7 --verify HEAD)
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)
TMP_PATH = $(shell pwd)/.tmp
CERT_PATH = $(TMP_PATH)/pki_certs
DEVSIM_NET_HOST_PATH = $(shell pwd)/.tmp/devsim-net-host
CERT_TOOL_IMAGE ?= ghcr.io/plgd-dev/hub/cert-tool:vnext
DEVSIM_IMAGE ?= ghcr.io/iotivity/iotivity-lite/cloud-server-discovery-resource-observable-debug:master

default: build

define build-docker-image
	docker build \
		--network=host \
		--tag $(SERVICE_NAME):$(VERSION_TAG) \
		--target $(1) \
		-f test/cloud-server/Dockerfile \
		.
endef

build-testcontainer:
	$(call build-docker-image,service)

build: build-testcontainer

certificates:
	mkdir -p $(CERT_PATH)
	docker pull $(CERT_TOOL_IMAGE)
	docker run --rm -v $(CERT_PATH):/out $(CERT_TOOL_IMAGE) --outCert=/out/cloudca.pem --outKey=/out/cloudcakey.pem --cert.subject.cn="ca" --cmd.generateRootCA
	docker run --rm -v $(CERT_PATH):/out $(CERT_TOOL_IMAGE) --signerCert=/out/cloudca.pem --signerKey=/out/cloudcakey.pem  --outCert=/out/intermediatecacrt.pem --outKey=/out/intermediatecakey.pem --cert.basicConstraints.maxPathLen=0 --cert.subject.cn="intermediateCA" --cmd.generateIntermediateCA
	docker run --rm -v $(CERT_PATH):/out $(CERT_TOOL_IMAGE) --signerCert=/out/intermediatecacrt.pem --signerKey=/out/intermediatecakey.pem --outCert=/out/mfgcrt.pem --outKey=/out/mfgkey.pem --cert.san.domain=localhost --cert.san.ip=127.0.0.1 --cert.subject.cn="mfg" --cmd.generateCertificate
	sudo chmod -R 0777 $(CERT_PATH)

env: clean certificates
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6'; \
	fi
	mkdir -p $(DEVSIM_NET_HOST_PATH)
	docker pull $(DEVSIM_IMAGE)
	docker run -d \
		--privileged \
		--network=host \
		--name devsim-net-host \
		-v $(DEVSIM_NET_HOST_PATH):/tmp \
		-v $(CERT_PATH):/pki_certs \
		$(DEVSIM_IMAGE) devsim-$(SIMULATOR_NAME_SUFFIX)

unit-test:
	mkdir -p $(TMP_PATH)
	go test -race -v ./schema/... -covermode=atomic -coverprofile=$(TMP_PATH)/schema.coverage.txt
	go test -race -v ./pkg/... -covermode=atomic -coverprofile=$(TMP_PATH)/pkg.coverage.txt

test: env build-testcontainer
	docker run \
		--network=host \
		--rm \
		-v $(CERT_PATH):/pki_certs \
		-v $(TMP_PATH):/tmp \
		$(SERVICE_NAME):$(VERSION_TAG) -test.parallel 1 -test.v -test.coverprofile=/tmp/coverage.txt

clean:
	docker rm -f devsim-net-host || true
	sudo rm -rf .tmp/*

.PHONY: build-testcontainer build certificates clean env test unit-test

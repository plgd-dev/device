SHELL = /bin/bash
SERVICE_NAME = $(notdir $(CURDIR))
LATEST_TAG = vnext
VERSION_TAG = vnext-$(shell git rev-parse --short=7 --verify HEAD)
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)
TMP_PATH = $(shell pwd)/.tmp
CERT_PATH = $(TMP_PATH)/pki_certs
DEVSIM_NET_HOST_PATH = $(shell pwd)/.tmp/devsim-net-host
DEVSIM_NET_BRIDGE_PATH = $(shell pwd)/.tmp/devsim-net-bridge
CERT_TOOL_IMAGE ?= ghcr.io/plgd-dev/hub/cert-tool:vnext
DEVSIM_IMAGE ?= ghcr.io/iotivity/iotivity-lite/cloud-server-debug:master

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

certificates:
	mkdir -p $(CERT_PATH)
	docker pull $(CERT_TOOL_IMAGE)
	docker run --rm -v $(CERT_PATH):/out $(CERT_TOOL_IMAGE) --outCert=/out/cloudca.pem --outKey=/out/cloudcakey.pem --cert.subject.cn="ca" --cmd.generateRootCA
	docker run --rm -v $(CERT_PATH):/out $(CERT_TOOL_IMAGE) --signerCert=/out/cloudca.pem --signerKey=/out/cloudcakey.pem  --outCert=/out/intermediatecacrt.pem --outKey=/out/intermediatecakey.pem --cert.basicConstraints.maxPathLen=0 --cert.subject.cn="intermediateCA" --cmd.generateIntermediateCA
	docker run --rm -v $(CERT_PATH):/out $(CERT_TOOL_IMAGE) --signerCert=/out/intermediatecacrt.pem --signerKey=/out/intermediatecakey.pem --outCert=/out/mfgcrt.pem --outKey=/out/mfgkey.pem --cert.san.domain=localhost --cert.san.ip=127.0.0.1 --cert.subject.cn="mfg" --cmd.generateCertificate
	sudo chmod -R 0777 $(CERT_PATH)
.PHONY: certificates

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
		$(DEVSIM_IMAGE) devsim-net-host-$(SIMULATOR_NAME_SUFFIX)
	mkdir -p $(DEVSIM_NET_BRIDGE_PATH)
	docker run -d \
		--privileged \
		--name devsim-net-bridge \
		-v $(DEVSIM_NET_BRIDGE_PATH):/tmp \
		-v $(CERT_PATH):/pki_certs \
		$(DEVSIM_IMAGE) devsim-net-bridge-$(SIMULATOR_NAME_SUFFIX)

test: env build-testcontainer 
	docker run \
		--network=host \
		-e ROOT_CA_CRT="/pki_certs/cloudca.pem" \
        -e ROOT_CA_KEY="/pki_certs/cloudcakey.pem" \
        -e INTERMEDIATE_CA_CRT="/pki_certs/intermediatecacrt.pem" \
        -e INTERMEDIATE_CA_KEY="/pki_certs/intermediatecakey.pem" \
        -e MFG_CRT="/pki_certs/mfgcrt.pem" \
        -e MFG_KEY="/pki_certs/mfgkey.pem" \
		-e IDENTITY_CRT="/pki_certs/identitycrt.pem" \
        -e IDENTITY_KEY="/pki_certs/identitykey.pem" \
		-v $(CERT_PATH):/pki_certs \
		-v $(TMP_PATH):/shared \
		ocfcloud/$(SERVICE_NAME):$(VERSION_TAG) \
		go test -p 1 -race -v ./... -coverpkg=./... -covermode=atomic -coverprofile=/shared/coverage.txt

clean:
	docker rm -f devsim-net-host || true
	docker rm -f devsim-net-bridge || true
	sudo rm -rf .tmp/*

.PHONY: build-testcontainer build test clean env make-ca make-mongo make-nats

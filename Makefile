SHELL = /bin/bash
SERVICE_NAME = cloud-server-test
VERSION_TAG = vnext-$(shell git rev-parse --short=7 --verify HEAD)
SIMULATOR_NAME_SUFFIX ?= $(shell hostname)
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)
TMP_PATH = $(shell pwd)/.tmp
CERT_PATH = $(TMP_PATH)/pki_certs
CLOUD_SID ?= adebc667-1f2b-41e3-bf5c-6d6eabc68cc6
DEVSIM_NET_HOST_PATH = $(shell pwd)/.tmp/devsim-net-host
CERT_TOOL_IMAGE ?= ghcr.io/plgd-dev/hub/cert-tool:vnext
# supported values: ECDSA-SHA256, ECDSA-SHA384, ECDSA-SHA512
CERT_TOOL_SIGN_ALG ?= ECDSA-SHA256
# supported values: P256, P384, P521
CERT_TOOL_ELLIPTIC_CURVE ?= P256
DEVSIM_IMAGE ?= ghcr.io/iotivity/iotivity-lite/cloud-server-discovery-resource-observable-debug:vnext
HUB_TEST_DEVICE_IMAGE = ghcr.io/plgd-dev/hub/test-cloud-server:vnext-pr1202

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

ROOT_CA_CRT = $(CERT_PATH)/cloudca.pem
ROOT_CA_KEY = $(CERT_PATH)/cloudcakey.pem
INTERMEDIATE_CA_CRT = $(CERT_PATH)/intermediatecacrt.pem
INTERMEDIATE_CA_KEY = $(CERT_PATH)/intermediatecakey.pem
MFG_CRT = $(CERT_PATH)/mfgcrt.pem
MFG_KEY = $(CERT_PATH)/mfgkey.pem
COAP_CRT = $(CERT_PATH)/coapcrt.pem
COAP_KEY = $(CERT_PATH)/coapkey.pem

certificates:
	mkdir -p $(CERT_PATH)
	chmod 0777 $(CERT_PATH)
	docker pull $(CERT_TOOL_IMAGE)

	docker run \
		--rm -v $(CERT_PATH):/out \
		--user $(USER_ID):$(GROUP_ID) \
		$(CERT_TOOL_IMAGE) \
			--outCert=/out/cloudca.pem --outKey=/out/cloudcakey.pem \
			--cert.subject.cn="ca" --cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE) \
			--cmd.generateRootCA

	docker run \
		--rm -v $(CERT_PATH):/out \
		--user $(USER_ID):$(GROUP_ID) \
		$(CERT_TOOL_IMAGE) \
			--signerCert=/out/cloudca.pem --signerKey=/out/cloudcakey.pem \
			--outCert=/out/intermediatecacrt.pem --outKey=/out/intermediatecakey.pem \
			--cert.basicConstraints.maxPathLen=0 --cert.subject.cn="intermediateCA" \
			--cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE) --cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) \
			--cmd.generateIntermediateCA

	docker run \
		--rm -v $(CERT_PATH):/out \
		--user $(USER_ID):$(GROUP_ID) \
		$(CERT_TOOL_IMAGE) \
			--signerCert=/out/intermediatecacrt.pem --signerKey=/out/intermediatecakey.pem \
			--outCert=/out/mfgcrt.pem --outKey=/out/mfgkey.pem --cert.san.domain=localhost \
			--cert.san.ip=127.0.0.1 --cert.subject.cn="mfg" \
			--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE) \
			--cmd.generateCertificate

	docker run \
		--rm -v $(CERT_PATH):/out \
		--user $(USER_ID):$(GROUP_ID) \
		${CERT_TOOL_IMAGE} \
			--signerCert=/out/cloudca.pem --signerKey=/out/cloudcakey.pem \
			--outCert=/out/coapcrt.pem --outKey=/out/coapkey.pem \
			--cert.san.ip=127.0.0.1 --cert.san.domain=localhost \
			--cert.signatureAlgorithm=$(CERT_TOOL_SIGN_ALG) --cert.ellipticCurve=$(CERT_TOOL_ELLIPTIC_CURVE) \
			--cmd.generateCertificate --cert.subject.cn=uuid:$(CLOUD_SID)

	sudo chown -R $(shell whoami) $(CERT_PATH)
	chmod -R 0777 $(CERT_PATH)

env: clean certificates
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6'; \
	fi
	mkdir -p $(DEVSIM_NET_HOST_PATH)/creds

	docker pull $(DEVSIM_IMAGE)
	docker run -d \
		--privileged \
		--network=host \
		--name devsim-net-host \
		-v $(DEVSIM_NET_HOST_PATH):/tmp \
		-v $(DEVSIM_NET_HOST_PATH)/creds:/cloud_server_creds \
		-v $(CERT_PATH):/pki_certs \
		$(DEVSIM_IMAGE) devsim-$(SIMULATOR_NAME_SUFFIX)

unit-test: certificates
	mkdir -p $(TMP_PATH)
	ROOT_CA_CRT="$(ROOT_CA_CRT)" ROOT_CA_KEY="$(ROOT_CA_KEY)" \
	MFG_CRT="$(MFG_CRT)" MFG_KEY="$(MFG_KEY)" \
	INTERMEDIATE_CA_CRT="$(INTERMEDIATE_CA_CRT)" INTERMEDIATE_CA_KEY=$(INTERMEDIATE_CA_KEY) \
	COAP_CRT="$(COAP_CRT)" COAP_KEY="$(COAP_KEY)" \
	CLOUD_SID=$(CLOUD_SID) \
		go test -race -parallel 1 -v ./bridge/... -coverpkg=./... -covermode=atomic -coverprofile=$(TMP_PATH)/bridge.unit.coverage.txt
	go test -race -v ./schema/... -covermode=atomic -coverprofile=$(TMP_PATH)/schema.unit.coverage.txt
	ROOT_CA_CRT="$(ROOT_CA_CRT)" ROOT_CA_KEY="$(ROOT_CA_KEY)" \
		go test -race -v ./pkg/... -covermode=atomic -coverprofile=$(TMP_PATH)/pkg.unit.coverage.txt

test: env build-testcontainer
	docker run \
		--network=host \
		--rm \
		-v $(CERT_PATH):/pki_certs \
		-v $(TMP_PATH):/tmp \
		$(SERVICE_NAME):$(VERSION_TAG) -test.parallel 1 -test.v -test.coverprofile=/tmp/coverage.txt

test-bridge:
	sudo rm -rf $(TMP_PATH)/data || :
	mkdir -p $(TMP_PATH)/data
	# pull image
	docker pull $(HUB_TEST_DEVICE_IMAGE)
	# prepare environment
	docker run \
		--rm \
		--network=host \
		--name hub-device-tests-environment \
		--env PREPARE_ENV=true \
		--env RUN=false \
		--env COAP_GATEWAY_CLOUD_ID="$(CLOUD_SID)" \
		-v $(TMP_PATH):/tmp \
		-v $(TMP_PATH)/data:/data \
		$(HUB_TEST_DEVICE_IMAGE)

	# start device
	rm -rf $(TMP_PATH)/bridge || :
	mkdir -p $(TMP_PATH)/bridge
	go build -C ./test/bridge-device -cover -o ./bridge-device
	pkill -KILL bridge-device || :
	CLOUD_SID=$(CLOUD_SID) CA_POOL=$(TMP_PATH)/data/certs/root_ca.crt \
	CERT_FILE=$(TMP_PATH)/data/certs/external/coap-gateway.crt \
	KEY_FILE=$(TMP_PATH)/data/certs/external/coap-gateway.key \
	GOCOVERDIR=$(TMP_PATH)/bridge \
		./test/bridge-device/bridge-device &

	# run tests
	docker run \
		--rm \
		--network=host \
		--name hub-device-tests \
		--env PREPARE_ENV=false \
		--env RUN=true \
		--env COAP_GATEWAY_CLOUD_ID="$(CLOUD_SID)" \
		--env TEST_DEVICE_NAME="bridged-device-0" \
		--env TEST_DEVICE_TYPE="bridged" \
		--env GRPC_GATEWAY_TEST_DISABLED=1 \
		--env IOTIVITY_LITE_TEST_RUN="(TestOffboard|TestOffboardWithoutSignIn|TestOffboardWithRepeat|TestRepublishAfterRefresh)$$" \
		-v $(TMP_PATH):/tmp \
		-v $(TMP_PATH)/data:/data \
		$(HUB_TEST_DEVICE_IMAGE)

	# stop device
	pkill -TERM bridge-device || :
	while pgrep -x bridge-device > /dev/null; do \
		echo "waiting for bridge-device to exit"; \
		sleep 1; \
	done
	go tool covdata textfmt -i=$(TMP_PATH)/bridge -o $(TMP_PATH)/bridge.coverage.txt

clean:
	docker rm -f devsim-net-host || :
	docker rm -f hub-device-tests-environment || :
	docker rm -f hub-device-tests || :
	pkill -KILL bridge-device || :
	sudo rm -rf .tmp/*

.PHONY: build-testcontainer build certificates clean env test unit-test

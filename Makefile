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

make-ca:
	docker pull smallstep/step-ca
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo net.ipv4.ip_unprivileged_port_start=0 > /etc/sysctl.d/50-unprivileged-ports.conf'; \
		sudo sysctl --system; \
	fi
	mkdir -p ./test/step-ca/data/secrets
	echo "password" > ./test/step-ca/data/secrets/password
	docker run \
		-it \
		-v "$(shell pwd)"/test/step-ca/data:/home/step --user $(shell id -u):$(shell id -g) \
		smallstep/step-ca \
		/bin/bash -c "step ca init -dns localhost -address=:10443 -provisioner=test@localhost -name test -password-file ./secrets/password && step ca provisioner add acme --type ACME"
	docker run \
		-d \
		--network=host \
		--name=step-ca-test \
		-v /etc/nsswitch.conf:/etc/nsswitch.conf \
		-v "$(shell pwd)"/test/step-ca/data:/home/step --user $(shell id -u):$(shell id -g) \
		smallstep/step-ca

make-nats:
	sleep 1
	docker exec -it step-ca-test /bin/bash -c "mkdir -p certs/nats && step ca certificate localhost certs/nats/nats.crt certs/nats/nats.key --provisioner acme"
	docker run \
	    -d \
		--network=host \
		--name=nats \
		-v $(shell pwd)/test/step-ca/data/certs:/certs \
		nats --tls --tlsverify --tlscert=/certs/nats/nats.crt --tlskey=/certs/nats/nats.key --tlscacert=/certs/root_ca.crt

make-mongo:
	sleep 1
	mkdir -p $(shell pwd)/test/mongo
	docker exec -it step-ca-test /bin/bash -c "mkdir -p certs/mongo && step ca certificate localhost certs/mongo/mongo.crt certs/mongo/mongo.key --provisioner acme && cat certs/mongo/mongo.crt >> certs/mongo/mongo.key"
	docker run \
	    -d \
		--network=host \
		--name=mongo \
		-v $(shell pwd)/test/mongo:/data/db \
		-v $(shell pwd)/test/step-ca/data/certs:/certs --user $(shell id -u):$(shell id -g) \
		mongo --tlsMode requireTLS --tlsCAFile /certs/root_ca.crt --tlsCertificateKeyFile certs/mongo/mongo.key

env: clean make-ca make-nats make-mongo
	if [ "${TRAVIS_OS_NAME}" == "linux" ]; then \
		sudo sh -c 'echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6'; \
	fi
	docker build ./device-simulator --network=host -t device-simulator --target service
	docker build ./device-simulator -f ./device-simulator/Dockerfile.insecure --network=host -t device-simulator-insecure --target service
	docker run -d --name devsimsec --network=host device-simulator devsimsec-$(SIMULATOR_NAME_SUFFIX)
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
	docker rm -f step-ca-test || true
	docker rm -f mongo || true
	docker rm -f nats || true
	rm -rf ./test/step-ca || true
	rm -rf ./test/mongo || true

.PHONY: build-testcontainer build test clean env make-ca make-mongo make-nats

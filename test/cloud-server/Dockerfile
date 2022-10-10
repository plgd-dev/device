FROM golang:1.19.2-alpine AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/device
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test -c -coverpkg=./... -covermode=atomic ./client -o /go/bin/device.client.test
RUN go test -c -coverpkg=./... -covermode=atomic ./client/core -o /go/bin/device.client.core.test

FROM alpine:3.15 AS service
RUN apk add bash
COPY --from=build /go/bin/device.client.test /usr/local/bin/device.client.test
COPY --from=build /go/bin/device.client.core.test /usr/local/bin/device.client.core.test
COPY ./test/cloud-server/run.sh /usr/local/bin/run.sh
ENV ROOT_CA_CRT="/pki_certs/cloudca.pem"
ENV ROOT_CA_KEY="/pki_certs/cloudcakey.pem"
ENV INTERMEDIATE_CA_CRT="/pki_certs/intermediatecacrt.pem"
ENV INTERMEDIATE_CA_KEY="/pki_certs/intermediatecakey.pem"
ENV MFG_CRT="/pki_certs/mfgcrt.pem"
ENV MFG_KEY="/pki_certs/mfgkey.pem"
ENV IDENTITY_CRT="/pki_certs/identitycrt.pem"
ENV IDENTITY_KEY="/pki_certs/identitykey.pem"
ENTRYPOINT ["/usr/local/bin/run.sh"]
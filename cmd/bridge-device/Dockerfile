# syntax=docker/dockerfile:1
FROM golang:1.23.9-alpine AS build
RUN apk add --no-cache build-base curl git
WORKDIR $GOPATH/src/github.com/plgd-dev/device
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /usr/local/go
RUN patch -p1 < "${GOPATH}/src/github.com/plgd-dev/device/tools/docker/patches/shrink_tls_conn.patch"
WORKDIR $GOPATH/src/github.com/plgd-dev/device
RUN CGO_ENABLED=0 go build -o /go/bin/bridge-device ./cmd/bridge-device

FROM alpine:3.20 AS security-provider
RUN apk add -U --no-cache ca-certificates \
    && addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

FROM scratch AS service
COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=security-provider /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/bin/bridge-device /usr/local/bin/bridge-device
USER nonroot
COPY ./cmd/bridge-device/config.yaml /config.yaml
COPY ./cmd/bridge-device/bridge-device.jsonld /bridge-device.jsonld
ENTRYPOINT [ "/usr/local/bin/bridge-device" ]

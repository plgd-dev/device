# syntax=docker/dockerfile:1
FROM golang:1.20.14-alpine AS build
RUN apk add --no-cache curl git build-base
WORKDIR $GOPATH/src/github.com/plgd-dev/device
COPY go.mod go.sum ./
RUN go mod download
COPY . .
WORKDIR /usr/local/go
RUN patch -p1 < "${GOPATH}/src/github.com/plgd-dev/device/tools/docker/patches/shrink_tls_conn.patch"
WORKDIR $GOPATH/src/github.com/plgd-dev/device
RUN CGO_ENABLED=0 go build -o /go/bin/bridge-device ./cmd/bridge-device

FROM alpine:3.19 AS security-provider
RUN apk add -U --no-cache ca-certificates
RUN addgroup -S nonroot \
    && adduser -S nonroot -G nonroot

FROM scratch AS service
COPY --from=security-provider /etc/passwd /etc/passwd
COPY --from=security-provider /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/bin/bridge-device /usr/local/bin/bridge-device
USER nonroot
COPY ./cmd/bridge-device/config.yaml /config.yaml
ENTRYPOINT [ "/usr/local/bin/bridge-device" ]

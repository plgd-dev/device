FROM golang:1.11-alpine3.8

RUN apk add --no-cache curl git build-base zeromq-dev && \
	curl -SL -o /usr/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && \
	chmod +x /usr/bin/dep

ENV PROJDIR $GOPATH/src/github.com/go-ocf/sdk
WORKDIR $PROJDIR
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure -v --vendor-only
COPY . .

RUN OCF_INSECURE=true go generate ./vendor/github.com/go-ocf/kit/security/
RUN go build ./...

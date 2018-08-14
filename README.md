[![Build Status](https://travis-ci.com/go-ocf/go-sdk.svg?branch=master)](https://travis-ci.com/go-ocf/go-sdk)
[![codecov](https://codecov.io/gh/go-ocf/go-sdk/branch/master/graph/badge.svg)](https://codecov.io/gh/go-ocf/go-sdk)
[![Go Report](https://goreportcard.com/badge/github.com/go-ocf/go-sdk)](https://goreportcard.com/badge/github.com/go-ocf/go-sdk)

# go-sdk

Is an open source software framework enabling seamless device-to-device connectivity to address the emerging needs of the Internet of Things. It will be implemented according to specification [ocf-spec]

TODO:
* full implementation of resource discovery (update resourceDiscoveryInterface::Retrieve)
* full implementation of interfaces of "oic.if.baseline", "oic.if.link", "oic.if.r", "oic.if.rw"
* observation feature
* other resources
* design abstraction layer for protocols(coap, coaps, coap-tcp, coaps-tcp, http, https, quic) and encoding payload(cbor,json)
* whole security topic [ocf-spec-sec]
* ...

[ocf-spec]: https://openconnectivity.org/specs/OCF_Core_Specification.pdf
[ocf-spec-sec]: https://openconnectivity.org/specs/OCF_Security_Specification.pdf
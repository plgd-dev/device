[![CI](https://github.com/plgd-dev/device/workflows/CI/badge.svg)](https://github.com/plgd-dev/device/actions?query=workflow%3ACI)
[![Coverage Status](https://codecov.io/gh/plgd-dev/device/branch/main/graph/badge.svg)](https://codecov.io/gh/plgd-dev/device)
[![Go Report Card](https://goreportcard.com/badge/plgd-dev/device)](https://goreportcard.com/report/plgd-dev/device)
[![Gitter](https://badges.gitter.im/ocfcloud/Lobby.svg)](https://gitter.im/ocfcloud/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=plgd-dev_sdk&metric=alert_status)](https://sonarcloud.io/dashboard?id=plgd-dev_sdk)

# Device

The **client** enables interaction with devices in a local network:

- Listing devices
- Retrieving and updating resources
- Secure ownership transfer via coaps+tcp and coaps
- Onboard and offboard device
- Provisioning the cloud resource and credentials

## Requirements

- Go 1.18 or higher

## Installation OCF Client

```bash
go install github.com/plgd-dev/device/v2/cmd/ocfclient@latest
```

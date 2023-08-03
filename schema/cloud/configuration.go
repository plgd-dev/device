// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

// Package cloud implements Cloud Configuration Resource.
// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.coapcloudconf.swagger.json
package cloud

const (
	// ResourceType is the resource type of the Cloud Configuration Resource.
	ResourceType = "oic.r.coapcloudconf"
	// ResourceURI is the URI of the Cloud Configuration Resource.
	ResourceURI = "/CoapCloudConfResURI"
)

// ProvisioningStatus indicates the Cloud Provisioning status of the Device.
type ProvisioningStatus string

const (
	ProvisioningStatus_UNINITIALIZED     ProvisioningStatus = "uninitialized"
	ProvisioningStatus_READY_TO_REGISTER ProvisioningStatus = "readytoregister"
	ProvisioningStatus_REGISTERING       ProvisioningStatus = "registering"
	ProvisioningStatus_REGISTERED        ProvisioningStatus = "registered"
	ProvisioningStatus_FAILED            ProvisioningStatus = "failed"
)

// Configuration contains the supported fields of the Cloud Configuration Resource.
type Configuration struct {
	ResourceTypes         []string           `json:"rt"`
	Interfaces            []string           `json:"if"`
	Name                  string             `json:"n"`
	AuthorizationProvider string             `json:"apn"`
	CloudID               string             `json:"sid"`
	URL                   string             `json:"cis"`
	LastErrorCode         int                `json:"clec"`
	ProvisioningStatus    ProvisioningStatus `json:"cps"`
	RedirectURI           string             `json:"redirecturi"`
}

// ConfigurationUpdateRequest is used to update the Cloud Configuration Resource.
type ConfigurationUpdateRequest struct {
	AuthorizationProvider string `json:"apn"`
	URL                   string `json:"cis"`
	AuthorizationCode     string `json:"at"`
	CloudID               string `json:"sid"`
	RedirectURI           string `json:"redirecturi,omitempty"`
}

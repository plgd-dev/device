/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package device

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/credential"
)

type CloudConfig struct {
	Enabled bool
	cloud.Config
}

type CredentialConfig struct {
	Enabled bool
	credential.Config
}

type Config struct {
	ID                    uuid.UUID
	Name                  string
	ProtocolIndependentID uuid.UUID
	ResourceTypes         []string
	MaxMessageSize        uint32
	Cloud                 CloudConfig
	Credential            CredentialConfig
}

func (cfg *Config) Validate() error {
	if cfg.ProtocolIndependentID == uuid.Nil {
		return fmt.Errorf("protocolIndependentID is required")
	}
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}

	if cfg.Name == "" {
		cfg.Name = "Unnamed"
	}

	return nil
}

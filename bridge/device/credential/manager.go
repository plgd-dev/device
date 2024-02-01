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

package credential

import (
	"crypto/x509"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
)

type Manager struct {
	credentials *sync.Map[int, credential.Credential]
	save        func()
}

func New(cfg Config, save func()) *Manager {
	if save == nil {
		save = func() {
			// do nothing
		}
	}
	m := Manager{
		save:        save,
		credentials: sync.NewMap[int, credential.Credential](),
	}
	m.importConfig(cfg)
	return &m
}

func (m *Manager) importConfig(cfg Config) {
	m.AddOrReplaceCredentials(cfg.Credentials...)
}

func (m *Manager) GetCAPool() []*x509.Certificate {
	certs := make([]*x509.Certificate, 0, m.credentials.Length())
	m.credentials.Range(func(key int, value credential.Credential) bool {
		if value.Type != credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE {
			return true
		}
		if value.Usage != credential.CredentialUsage_TRUST_CA && value.Usage != credential.CredentialUsage_MFG_TRUST_CA {
			return true
		}
		cas, err := pkgX509.ParsePemCertificates(value.PublicData.Data())
		if err != nil {
			return true
		}
		certs = append(certs, cas...)
		return true
	})
	return certs
}

func (m *Manager) getNextID() int {
	var id int
	m.credentials.Range(func(key int, value credential.Credential) bool {
		if key > id {
			id = key
		}
		return true
	})
	return id + 1
}

func (m *Manager) add(c credential.Credential) {
	for {
		_, loaded := m.credentials.LoadOrStore(c.ID, c)
		if !loaded {
			return
		}
		c.ID = m.getNextID()
	}
}

func (m *Manager) AddOrReplaceCredential(c credential.Credential) {
	if c.Type == credential.CredentialType_EMPTY {
		return
	}
	if c.ID != 0 {
		// replace
		m.credentials.Store(c.ID, c)
		return
	}
	// add
	m.add(c)
}

func (m *Manager) AddOrReplaceCredentials(ca ...credential.Credential) {
	for _, c := range ca {
		m.AddOrReplaceCredential(c)
	}
}

func (m *Manager) RemoveCredentials(ids ...int) {
	for _, id := range ids {
		m.credentials.Delete(id)
	}
}

func (m *Manager) RemoveCredentialsBySubjects(subjects ...string) {
	m.credentials.Range(func(key int, value credential.Credential) bool {
		for _, subject := range subjects {
			if value.Subject == subject {
				m.credentials.Delete(key)
			}
		}
		return true
	})
}

func (m *Manager) ClearCredentials() {
	_ = m.credentials.LoadAndDeleteAll()
}

func (m *Manager) getRep(privateData bool) credential.CredentialResponse {
	cas := m.credentials.CopyData()
	creds := credential.CredentialResponse{
		Interfaces:    []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW},
		ResourceTypes: []string{credential.ResourceType},
		Credentials:   make([]credential.Credential, 0, len(cas)),
	}
	for _, cred := range cas {
		if !privateData {
			// remove private data
			cred.PrivateData = nil
		}
		creds.Credentials = append(creds.Credentials, cred)
	}
	return creds
}

func (m *Manager) Get(request *net.Request) (*pool.Message, error) {
	creds := m.getRep(false)
	return resources.CreateResponseContent(request.Context(), creds, codes.Content)
}

func (m *Manager) Post(request *net.Request) (*pool.Message, error) {
	var cfg credential.CredentialUpdateRequest
	err := cbor.ReadFrom(request.Body(), &cfg)
	if err != nil {
		return resources.CreateResponseBadRequest(request.Context(), err)
	}
	m.AddOrReplaceCredentials(cfg.Credentials...)
	m.save()
	creds := m.getRep(false)
	return resources.CreateResponseContent(request.Context(), creds, codes.Changed)
}

func (m *Manager) ExportConfig() Config {
	return m.getRep(true)
}

func (m *Manager) Close() {
	m.ClearCredentials()
}

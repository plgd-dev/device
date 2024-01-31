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
	"bytes"
	"context"
	"testing"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/test"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/stretchr/testify/require"
)

func TestGetCAPool(t *testing.T) {
	m := New(func() {})

	// Add some credentials to the manager
	cred1 := credential.Credential{
		Type:  credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
		Usage: credential.CredentialUsage_TRUST_CA,
		PublicData: &credential.CredentialPublicData{
			DataInternal: test.GetRootCApem(t),
		},
	}
	cred2 := credential.Credential{
		Type:  credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
		Usage: credential.CredentialUsage_MFG_TRUST_CA,
		PublicData: &credential.CredentialPublicData{
			DataInternal: test.GetRootCApem(t),
		},
	}
	cred3 := credential.Credential{
		Type:  credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
		Usage: credential.CredentialUsage_CERT,
		PublicData: &credential.CredentialPublicData{
			DataInternal: test.GetRootCApem(t),
		},
	}
	m.credentials.Store(1, cred1)
	m.credentials.Store(2, cred2)
	m.credentials.Store(3, cred3)

	// Call the GetCAPool function
	certs := m.GetCAPool()

	// Verify the result
	require.Len(t, certs, 2)
}

func TestAddOrReplaceCredential(t *testing.T) {
	m := New(func() {})

	// Test adding a new credential
	cred1 := credential.Credential{
		ID:      1,
		Subject: "subject",
		Type:    credential.CredentialType_PIN_OR_PASSWORD,
	}
	m.AddOrReplaceCredential(cred1)
	_, ok := m.credentials.Load(cred1.ID)
	require.True(t, ok)

	// Test replacing an existing credential
	cred2 := credential.Credential{
		ID:      1,
		Subject: "subject1",
		Type:    credential.CredentialType_ASYMMETRIC_SIGNING,
	}
	m.AddOrReplaceCredential(cred2)
	c, ok := m.credentials.Load(cred2.ID)
	require.True(t, ok)
	require.Equal(t, cred2, c)

	// Test adding an empty credential
	cred3 := credential.Credential{
		ID:   0,
		Type: credential.CredentialType_EMPTY,
	}
	m.AddOrReplaceCredential(cred3)
	_, ok = m.credentials.Load(cred3.ID)
	require.False(t, ok)
}

func TestAddOrReplaceCredentials(t *testing.T) {
	m := New(func() {}) // Create an instance of the Manager struct

	// Create some test credentials
	cred1 := credential.Credential{
		Subject: "subject1",
		Type:    credential.CredentialType_PIN_OR_PASSWORD,
	}
	cred2 := credential.Credential{
		Subject: "subject2",
		Type:    credential.CredentialType_PIN_OR_PASSWORD,
	}
	cred3 := credential.Credential{
		Subject: "subject3",
		Type:    credential.CredentialType_EMPTY,
	}

	// Add the credentials to the manager
	m.AddOrReplaceCredentials(cred1, cred2, cred3)

	// Verify that the credentials were added correctly
	require.Equal(t, 2, m.credentials.Length())
}

func TestRemoveCredentials(t *testing.T) {
	m := New(func() {})

	// Add some credentials to the Manager
	m.credentials.Store(1, credential.Credential{
		Subject: "subject1",
		Type:    credential.CredentialType_PIN_OR_PASSWORD,
	})
	m.credentials.Store(2, credential.Credential{
		Subject: "subject2",
		Type:    credential.CredentialType_PIN_OR_PASSWORD,
	})
	m.credentials.Store(3, credential.Credential{
		Subject: "subject3",
		Type:    credential.CredentialType_PIN_OR_PASSWORD,
	})

	// Remove credentials with IDs 1 and 3
	m.RemoveCredentials(1, 3)

	// Check if the credentials were removed
	_, ok := m.credentials.Load(1)
	require.False(t, ok, "Credential with ID 1 should be removed")
	_, ok = m.credentials.Load(3)
	require.False(t, ok, "Credential with ID 3 should be removed")

	// Check if the credential with ID 2 still exists
	_, ok = m.credentials.Load(2)
	require.True(t, ok, "Credential with ID 2 should still exist")
}

func TestRemoveCredentialsBySubjects(t *testing.T) {
	// Create a new instance of the Manager
	m := New(func() {})

	// Add some credentials to the Manager
	cred1 := credential.Credential{Subject: "subject1", Type: credential.CredentialType_PIN_OR_PASSWORD}
	cred2 := credential.Credential{Subject: "subject2", Type: credential.CredentialType_PIN_OR_PASSWORD}
	cred3 := credential.Credential{Subject: "subject3", Type: credential.CredentialType_PIN_OR_PASSWORD}
	m.credentials.Store(1, cred1)
	m.credentials.Store(2, cred2)
	m.credentials.Store(3, cred3)

	// Call the RemoveCredentialsBySubjects method with some subjects
	m.RemoveCredentialsBySubjects("subject1", "subject3")

	// Check that the credentials with the specified subjects have been removed
	_, ok1 := m.credentials.Load(1)
	_, ok2 := m.credentials.Load(2)
	_, ok3 := m.credentials.Load(3)
	require.False(t, ok1, "Credential with subject 'subject1' should have been removed")
	require.True(t, ok2, "Credential with subject 'subject2' should not have been removed")
	require.False(t, ok3, "Credential with subject 'subject3' should have been removed")
}

func TestClearCredentials(t *testing.T) {
	m := New(func() {}) // Create an instance of the Manager struct

	// Add some credentials to the Manager
	cred1 := credential.Credential{Subject: "subject1", Type: credential.CredentialType_PIN_OR_PASSWORD}
	cred2 := credential.Credential{Subject: "subject2", Type: credential.CredentialType_PIN_OR_PASSWORD}
	cred3 := credential.Credential{Subject: "subject3", Type: credential.CredentialType_PIN_OR_PASSWORD}
	m.credentials.Store(1, cred1)
	m.credentials.Store(2, cred2)
	m.credentials.Store(3, cred3)

	// Call the ClearCredentials method
	m.ClearCredentials()

	require.Equal(t, 0, m.credentials.Length())
}

func TestGetCredential(t *testing.T) {
	m := New(func() {}) // Create an instance of the Manager struct

	// Add some credentials to the Manager
	cred1 := credential.Credential{Subject: "subject1", Type: credential.CredentialType_PIN_OR_PASSWORD, PrivateData: &credential.CredentialPrivateData{DataInternal: []byte("private data")}}
	cred2 := credential.Credential{Subject: "subject2", Type: credential.CredentialType_PIN_OR_PASSWORD}
	cred3 := credential.Credential{Subject: "subject3", Type: credential.CredentialType_PIN_OR_PASSWORD}
	m.credentials.Store(1, cred1)
	m.credentials.Store(2, cred2)
	m.credentials.Store(3, cred3)

	msg := pool.NewMessage(context.Background())
	msg.SetCode(codes.GET)
	err := msg.SetPath(credential.ResourceURI)
	require.NoError(t, err)
	msg.SetToken([]byte{0x01})

	// Create a mock request
	request := &net.Request{
		Message: msg,
	}

	// Call the Get method
	response, err := m.Get(request)

	// Check if the response and error are as expected
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Equal(t, codes.Content, response.Code())
	require.NotNil(t, response.Body())

	var cred credential.CredentialResponse
	err = cbor.ReadFrom(response.Body(), &cred)
	require.NoError(t, err)
	require.Len(t, cred.Credentials, 3)
	for _, c := range cred.Credentials {
		require.Nil(t, c.PrivateData)
	}
}

func TestPostCredential(t *testing.T) {
	m := New(func() {}) // Create an instance of the Manager struct

	msg := pool.NewMessage(context.Background())
	msg.SetCode(codes.POST)
	err := msg.SetPath(credential.ResourceURI)
	require.NoError(t, err)
	msg.SetToken([]byte{0x01})

	updBody := credential.CredentialUpdateRequest{
		Credentials: []credential.Credential{
			{
				Subject: "subject1",
				Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   credential.CredentialUsage_TRUST_CA,
				PublicData: &credential.CredentialPublicData{
					DataInternal: test.GetRootCApem(t),
					Encoding:     credential.CredentialPublicDataEncoding_PEM,
				},
			},
		},
	}
	body, err := cbor.Encode(updBody)
	require.NoError(t, err)
	msg.SetBody(bytes.NewReader(body))

	// Create a mock request
	req := &net.Request{
		Message: msg,
	}

	// Call the Post method
	resp, err := m.Post(req)
	require.NoError(t, err)

	// Verify the response
	require.NotNil(t, resp)
	require.Equal(t, codes.Changed, resp.Code())

	// Verify the credentials were added or replaced
	creds := m.getRep(false)
	require.Len(t, creds.Credentials, 1)
	require.Equal(t, "subject1", creds.Credentials[0].Subject)
}

func TestExportConfig(t *testing.T) {
	m := New(func() {}) // Create an instance of the Manager struct

	// Add some credentials to the Manager
	cred1 := credential.Credential{Subject: "subject1", Type: credential.CredentialType_PIN_OR_PASSWORD, PrivateData: &credential.CredentialPrivateData{DataInternal: []byte("private data")}}
	m.credentials.Store(1, cred1)

	// Call the ExportConfig function
	config := m.ExportConfig()

	// Assert that the returned config is not nil
	require.Len(t, config.Credentials, 1)
	require.Equal(t, "subject1", config.Credentials[0].Subject)
	require.NotNil(t, config.Credentials[0].PrivateData)
	require.NotNil(t, config.Credentials[0].PrivateData.DataInternal)
}

func TestClose(t *testing.T) {
	m := New(func() {}) // Create an instance of the Manager struct

	// Add some credentials to the Manager
	cred1 := credential.Credential{Subject: "subject1", Type: credential.CredentialType_PIN_OR_PASSWORD, PrivateData: &credential.CredentialPrivateData{DataInternal: []byte("private data")}}
	m.credentials.Store(1, cred1)

	// Call the Close function
	m.Close()
	require.Equal(t, 0, m.credentials.Length())
}

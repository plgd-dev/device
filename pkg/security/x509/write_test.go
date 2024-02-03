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

package x509_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func TestCreatePemChain(t *testing.T) {
	cfg := generateCertificate.Configuration{}
	caKey, err := cfg.GenerateKey()
	require.NoError(t, err)
	caPem, err := generateCertificate.GenerateRootCA(cfg, caKey)
	require.NoError(t, err)
	ca, err := pkgX509.ParsePemCertificates(caPem)
	require.NoError(t, err)
	key, err := cfg.GenerateKey()
	require.NoError(t, err)
	intermediatePem, err := generateCertificate.GenerateIntermediateCA(cfg, key, ca, caKey)
	require.NoError(t, err)
	intermediate, err := pkgX509.ParsePemCertificates(intermediatePem)
	require.NoError(t, err)

	_, err = pkgX509.CreatePemChain(intermediate, caPem)
	require.NoError(t, err)
}

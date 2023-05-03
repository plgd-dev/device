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

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/util/metautils"
	"github.com/plgd-dev/device/v2/client/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SignFunc = func(ctx context.Context, csr []byte) (signedCsr []byte, err error)

type deviceOwnershipBackend struct {
	identityCertificate tls.Certificate
	identityCACert      []*x509.Certificate
	jwtClaimOwnerID     string
	app                 ApplicationCallback
	dialTLS             core.DialTLS
	dialDTLS            core.DialDTLS
	sign                SignFunc
}

type DeviceOwnershipBackendConfig struct {
	JWTClaimOwnerID string
	Sign            SignFunc
}

func newDeviceOwnershipBackendFromConfig(app ApplicationCallback, dialTLS core.DialTLS, dialDTLS core.DialDTLS,
	cfg *DeviceOwnershipBackendConfig,
) (*deviceOwnershipBackend, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing device ownership backend config")
	}

	if cfg.JWTClaimOwnerID == "" {
		cfg.JWTClaimOwnerID = "sub"
	}

	return &deviceOwnershipBackend{
		sign:            cfg.Sign,
		app:             app,
		jwtClaimOwnerID: cfg.JWTClaimOwnerID,
		dialTLS:         dialTLS,
		dialDTLS:        dialDTLS,
	}, nil
}

func (o *deviceOwnershipBackend) OwnDevice(ctx context.Context, deviceID string, otmTypes []OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error) {
	otmClients, err := getOtmClients(o.app, o.dialTLS, o.dialDTLS, otmTypes)
	if err != nil {
		return "", err
	}
	opts = append([]core.OwnOption{core.WithSetupCertificates(o.sign)}, opts...)
	return own(ctx, deviceID, otmClients, discoveryConfiguration, opts...)
}

func (o *deviceOwnershipBackend) setIdentityCertificate(ctx context.Context, accessToken string) error {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	var c jwt.MapClaims
	_, _, err := parser.ParseUnverified(accessToken, &c)
	if err != nil {
		return fmt.Errorf("cannot parse jwt token: %w", err)
	}
	if c[o.jwtClaimOwnerID] == nil {
		return fmt.Errorf("cannot get '%v' from jwt token: is not set", o.jwtClaimOwnerID)
	}
	ownerStr := fmt.Sprintf("%v", c[o.jwtClaimOwnerID])
	ownerID, err := uuid.Parse(ownerStr)
	if err != nil || ownerStr == uuid.Nil.String() {
		ownerID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(ownerStr))
	}
	cert, caCert, err := GenerateSDKIdentityCertificate(ctx, o.sign, ownerID.String())
	if err != nil {
		return err
	}

	o.identityCertificate = cert
	o.identityCACert = caCert

	return nil
}

var headerAuthorize = "authorization"

// TokenFromOutgoingMD extracts token stored by CtxWithToken.
func TokenFromOutgoingMD(ctx context.Context) (string, error) {
	expectedScheme := "bearer"
	val := metautils.ExtractOutgoing(ctx).Get(headerAuthorize)
	if val == "" {
		return "", status.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	splits := strings.SplitN(val, " ", 2)
	if len(splits) < 2 {
		return "", status.Errorf(codes.Unauthenticated, "Bad authorization string")
	}
	if !strings.EqualFold(splits[0], expectedScheme) {
		return "", status.Errorf(codes.Unauthenticated, "Request unauthenticated with %v, expected %v", splits[0], expectedScheme)
	}
	return splits[1], nil
}

func (o *deviceOwnershipBackend) Initialization(ctx context.Context) error {
	token, err := TokenFromOutgoingMD(ctx)
	if err != nil {
		return err
	}
	return o.setIdentityCertificate(ctx, token)
}

func (o *deviceOwnershipBackend) GetIdentityCertificate() (tls.Certificate, error) {
	if o.identityCertificate.PrivateKey == nil {
		return tls.Certificate{}, fmt.Errorf("client is not initialized")
	}
	return o.identityCertificate, nil
}

func (o *deviceOwnershipBackend) GetIdentityCACerts() ([]*x509.Certificate, error) {
	if o.identityCACert == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	return o.identityCACert, nil
}

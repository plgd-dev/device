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

package test

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/metadata"
	"github.com/stretchr/testify/require"
)

const (
	authorizationKey = "authorization"
	jwtSecret        = "secret"
)

func CreateJWTToken(t *testing.T, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return tokenString
}

// CtxWithToken stores token to ctx of request.
func CtxWithToken(ctx context.Context, token string) context.Context {
	niceMD := metadata.ExtractOutgoing(ctx)
	niceMD.Set(authorizationKey, fmt.Sprintf("%s %s", "bearer", token))
	return niceMD.ToOutgoing(ctx)
}

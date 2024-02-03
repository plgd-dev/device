package client_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/stretchr/testify/require"
)

func TestBackendOwnershipClient(t *testing.T) {
	const ownerClaimKey = "sub"
	jwtWithSubUserID := test.CreateJWTToken(t, jwt.MapClaims{
		ownerClaimKey: "userId",
	})

	s, err := test.NewTestSigner()
	require.NoError(t, err)

	cfg := client.Config{
		DeviceOwnershipBackend: &client.DeviceOwnershipBackendConfig{
			JWTClaimOwnerID: ownerClaimKey,
			Sign:            s.Sign,
		},
	}

	mfgTrustedCABlock, _ := pem.Decode(test.RootCACrt)
	require.NotNil(t, mfgTrustedCABlock)

	mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
	require.NoError(t, err)

	mfgCert, err := tls.X509KeyPair(test.MfgCert, test.MfgKey)
	require.NoError(t, err)

	client, err := client.NewClientFromConfig(&cfg,
		testClient.NewTestSetupSecureClient(nil, mfgCA, mfgCert),
		log.NewStdLogger(log.LevelDebug))
	require.NoError(t, err)

	ctxWithToken := test.CtxWithToken(context.Background(), jwtWithSubUserID)
	err = client.Initialization(ctxWithToken)
	require.NoError(t, err)

	_, err = client.GetIdentityCertificate()
	require.NoError(t, err)

	_, err = client.GetIdentityCACerts()
	require.NoError(t, err)

	deviceID := test.MustFindDeviceByName(test.DevsimName)
	ctx, cancel := context.WithTimeout(ctxWithToken, time.Second*30)
	defer cancel()

	_, err = client.OwnDevice(ctx, deviceID)
	require.NoError(t, err)

	err = client.DisownDevice(ctx, deviceID)
	require.NoError(t, err)
}

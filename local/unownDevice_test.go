package local_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/stretchr/testify/require"
)

func TestClient_UnownDevice(t *testing.T) {
	ownDevice := "00000000-cafe-baba-0000-000000000000"

	type args struct {
		deviceID         string
		discoveryTimeout time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				deviceID: ownDevice,
			},
		},
	}

	cert, err := tls.X509KeyPair(CertPEMBlock, KeyPEMBlock)
	require.NoError(t, err)
	derBlock, _ := pem.Decode(CARootPemBlock)
	require.NotEmpty(t, derBlock)
	ca, err := x509.ParseCertificate(derBlock.Bytes)
	require.NoError(t, err)
	derBlockKey, _ := pem.Decode(CARootKeyPemBlock)
	require.NotEmpty(t, derBlockKey)
	caKey, err := x509.ParseECPrivateKey(derBlockKey.Bytes)
	require.NoError(t, err)

	testOwnCfg := testCfg
	testOwnCfg.Resource.DiscoveryTimeout = time.Second * 10
	testOwnCfg.TLSConfig.GetCertificate = func() (tls.Certificate, error) {
		return cert, nil
	}
	testOwnCfg.TLSConfig.GetCertificateAuthorities = func() ([]*x509.Certificate, error) {
		return []*x509.Certificate{ca}, nil
	}

	signer := TestCertificateSigner{
		ca:       ca,
		caKey:    caKey,
		validFor: time.Hour * 86400,
	}

	otm := ocf.NewManufacturerOTMClient(cert, ca, signer)

	c, err := ocf.NewClientFromConfig(testOwnCfg, nil)
	require := require.New(t)
	require.NoError(err)

	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = c.OwnDevice(timeout, ownDevice, otm, 10*time.Second)
	require.NoError(err)

	/*
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		h := testOnboardDeviceHandler{}
		err = c.GetDevices(timeout, []string{"oic.d.cloudDevice"}, &h)
		require.NoError(err)
		deviceIds := h.PopDeviceIds()
		require.NotEmpty(deviceIds)
	*/

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout, reqCancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := c.UnownDevice(timeout, tt.args.deviceID, 10*time.Second)
			reqCancel()
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

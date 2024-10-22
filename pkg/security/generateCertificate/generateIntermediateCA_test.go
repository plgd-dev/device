package generateCertificate_test

import (
	"crypto/ecdsa"
	"crypto/x509"
	"testing"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func generateRootCA(t *testing.T, cfg generateCertificate.Configuration) ([]*x509.Certificate, *ecdsa.PrivateKey) {
	privateKey, err := cfg.GenerateKey()
	require.NoError(t, err)
	cert, err := generateCertificate.GenerateRootCA(cfg, privateKey)
	require.NoError(t, err)
	crt, err := pkgX509.ParsePemCertificates(cert)
	require.NoError(t, err)
	return crt, privateKey
}

func TestGenerateIntermediateCA(t *testing.T) {
	type args struct {
		cfg generateCertificate.Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid - default",
			args: args{
				cfg: generateCertificate.Configuration{},
			},
		},
		{
			name: "valid - sha384",
			args: args{
				cfg: generateCertificate.Configuration{
					SignatureAlgorithm: generateCertificate.SignatureAlgorithmECDSAWithSHA384,
				},
			},
		},
		{
			name: "valid - sha512",
			args: args{
				cfg: generateCertificate.Configuration{
					SignatureAlgorithm: generateCertificate.SignatureAlgorithmECDSAWithSHA512,
				},
			},
		},
		{
			name: "valid - p384",
			args: args{
				cfg: generateCertificate.Configuration{
					EllipticCurve: generateCertificate.EllipticCurveP384,
				},
			},
		},
		{
			name: "valid - p521",
			args: args{
				cfg: generateCertificate.Configuration{
					EllipticCurve: generateCertificate.EllipticCurveP521,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caCrt, caKey := generateRootCA(t, tt.args.cfg)
			privateKey, err := tt.args.cfg.GenerateKey()
			require.NoError(t, err)
			got, err := generateCertificate.GenerateIntermediateCA(tt.args.cfg, privateKey, caCrt, caKey)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

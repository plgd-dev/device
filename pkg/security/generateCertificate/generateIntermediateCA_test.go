package generateCertificate

import (
	"crypto/ecdsa"
	"crypto/x509"
	"testing"

	"github.com/plgd-dev/kit/v2/security"
	"github.com/stretchr/testify/require"
)

func generateRootCA(t *testing.T, cfg Configuration) ([]*x509.Certificate, *ecdsa.PrivateKey) {
	privateKey, err := cfg.GenerateKey()
	require.NoError(t, err)
	cert, err := GenerateRootCA(cfg, privateKey)
	require.NoError(t, err)
	crt, err := security.ParseX509FromPEM(cert)
	require.NoError(t, err)
	return crt, privateKey
}

func TestGenerateIntermediateCA(t *testing.T) {
	type args struct {
		cfg Configuration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid - default",
			args: args{
				cfg: Configuration{},
			},
		},
		{
			name: "valid - sha384",
			args: args{
				cfg: Configuration{
					SignatureAlgorithm: SignatureAlgorithmECDSAWithSHA384,
				},
			},
		},
		{
			name: "valid - sha512",
			args: args{
				cfg: Configuration{
					SignatureAlgorithm: SignatureAlgorithmECDSAWithSHA512,
				},
			},
		},
		{
			name: "valid - p384",
			args: args{
				cfg: Configuration{
					EllipticCurve: EllipticCurveP384,
				},
			},
		},
		{
			name: "valid - p521",
			args: args{
				cfg: Configuration{
					EllipticCurve: EllipticCurveP521,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caCrt, caKey := generateRootCA(t, tt.args.cfg)
			privateKey, err := tt.args.cfg.GenerateKey()
			require.NoError(t, err)
			got, err := GenerateIntermediateCA(tt.args.cfg, privateKey, caCrt, caKey)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

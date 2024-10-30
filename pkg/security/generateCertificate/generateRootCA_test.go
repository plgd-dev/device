package generateCertificate_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/stretchr/testify/require"
)

func TestGenerateRootCA(t *testing.T) {
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
			privateKey, err := tt.args.cfg.GenerateKey()
			require.NoError(t, err)
			got, err := generateCertificate.GenerateRootCA(tt.args.cfg, privateKey)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

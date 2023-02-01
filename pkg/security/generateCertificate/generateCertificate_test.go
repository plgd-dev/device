package generateCertificate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateCertificate(t *testing.T) {
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
				cfg: Configuration{
					ValidFor: time.Minute,
				},
			},
		},
		{
			name: "valid - sha384",
			args: args{
				cfg: Configuration{
					ValidFor:           time.Minute,
					SignatureAlgorithm: SignatureAlgorithmECDSAWithSHA384,
				},
			},
		},
		{
			name: "valid - sha512",
			args: args{
				cfg: Configuration{
					ValidFor:           time.Minute,
					SignatureAlgorithm: SignatureAlgorithmECDSAWithSHA512,
				},
			},
		},
		{
			name: "valid - p384",
			args: args{
				cfg: Configuration{
					ValidFor:      time.Minute,
					EllipticCurve: EllipticCurveP384,
				},
			},
		},
		{
			name: "valid - p521",
			args: args{
				cfg: Configuration{
					ValidFor:      time.Minute,
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
			got, err := GenerateCert(tt.args.cfg, privateKey, caCrt, caKey)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

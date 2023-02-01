package generateCertificate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateRootCA(t *testing.T) {
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
			privateKey, err := tt.args.cfg.GenerateKey()
			require.NoError(t, err)
			got, err := GenerateRootCA(tt.args.cfg, privateKey)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

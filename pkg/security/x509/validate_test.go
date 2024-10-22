package x509_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func TestValidateCRLDistributionPoints(t *testing.T) {
	tests := []struct {
		name      string
		crlPoints []string
		wantErr   bool
	}{
		{
			name:      "Valid CRL distribution points",
			crlPoints: []string{"http://example.com/crl1", "https://example.com/crl2"},
		},
		{
			name:      "Invalid CRL distribution point",
			crlPoints: []string{"http://valid-crl.com", "invalid-url"},
			wantErr:   true,
		},
		{
			name:      "Empty CRL distribution point",
			crlPoints: []string{"http://example.com/crl1", ""},
			wantErr:   true,
		},
		{
			name:      "Valid - No CRL distribution points",
			crlPoints: []string{},
		},
		{
			name:      "Valid - Nil CRL distribution points",
			crlPoints: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := x509.ValidateCRLDistributionPoints(tt.crlPoints)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

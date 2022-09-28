package credential_test

import (
	"testing"

	"github.com/plgd-dev/device/schema/credential"
	"github.com/stretchr/testify/require"
)

func TestCredentialTypeString(t *testing.T) {
	tests := []struct {
		name string
		s    credential.CredentialType
		want string
	}{
		{
			name: "Empty",
			s:    0,
			want: "EMPTY",
		},
		{
			name: "Unknown",
			s:    credential.CredentialType_ASYMMETRIC_ENCRYPTION_KEY << 1, // double of the last credential.CredentialType value
			want: "unknown(64)",
		},
		{
			name: "Single",
			s:    credential.CredentialType_SYMMETRIC_PAIR_WISE,
			want: "SYMMETRIC_PAIR_WISE",
		},
		{
			name: "All",
			s: credential.CredentialType_SYMMETRIC_PAIR_WISE | credential.CredentialType_SYMMETRIC_GROUP |
				credential.CredentialType_ASYMMETRIC_SIGNING | credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE |
				credential.CredentialType_PIN_OR_PASSWORD | credential.CredentialType_ASYMMETRIC_ENCRYPTION_KEY,
			want: "SYMMETRIC_PAIR_WISE|SYMMETRIC_GROUP|ASYMMETRIC_SIGNING|ASYMMETRIC_SIGNING_WITH_CERTIFICATE|PIN_OR_PASSWORD|ASYMMETRIC_ENCRYPTION_KEY",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.String()
			require.Equal(t, tt.want, got)
		})
	}
}

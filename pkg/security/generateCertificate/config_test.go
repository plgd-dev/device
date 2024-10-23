package generateCertificate_test

import (
	"crypto/x509"
	"encoding/asn1"
	"net"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/stretchr/testify/require"
)

func TestConfigToValidFrom(t *testing.T) {
	cfg := generateCertificate.Configuration{}
	_, err := cfg.ToValidFrom()
	require.NoError(t, err)

	cfg.ValidFrom = "now"
	_, err = cfg.ToValidFrom()
	require.NoError(t, err)

	cfg.ValidFrom = "2021-01-01T00:00:00Z"
	validFrom, err := cfg.ToValidFrom()
	require.NoError(t, err)
	vf, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
	require.NoError(t, err)
	require.Equal(t, vf, validFrom)

	cfg.ValidFrom = "invalid"
	_, err = cfg.ToValidFrom()
	require.Error(t, err)
}

func TestX509KeyUsages(t *testing.T) {
	cfg := generateCertificate.Configuration{}
	cfg.KeyUsages = []string{""}
	_, err := cfg.X509KeyUsages()
	require.NoError(t, err)

	cfg.KeyUsages = []string{"invalid"}
	_, err = cfg.X509KeyUsages()
	require.Error(t, err)

	cfg.KeyUsages = []string{"digitalSignature", "contentCommitment", "keyEncipherment", "dataEncipherment", "keyAgreement", "certSign", "crlSign", "encipherOnly", "decipherOnly"}
	ku, err := cfg.X509KeyUsages()
	require.NoError(t, err)
	require.NotEmpty(t, ku&x509.KeyUsageDigitalSignature)
	require.NotEmpty(t, ku&x509.KeyUsageContentCommitment)
	require.NotEmpty(t, ku&x509.KeyUsageKeyEncipherment)
	require.NotEmpty(t, ku&x509.KeyUsageDataEncipherment)
	require.NotEmpty(t, ku&x509.KeyUsageKeyAgreement)
	require.NotEmpty(t, ku&x509.KeyUsageCertSign)
	require.NotEmpty(t, ku&x509.KeyUsageCRLSign)
	require.NotEmpty(t, ku&x509.KeyUsageEncipherOnly)
	require.NotEmpty(t, ku&x509.KeyUsageDecipherOnly)
}

func TestX509ExtKeyUsages(t *testing.T) {
	cfg := generateCertificate.Configuration{
		ExtensionKeyUsages: []string{"invalid"},
	}
	_, _, err := cfg.X509ExtKeyUsages()
	require.Error(t, err)

	cfg = generateCertificate.Configuration{
		ExtensionKeyUsages: []string{""},
	}
	ekus, unknownEkus, err := cfg.X509ExtKeyUsages()
	require.NoError(t, err)
	require.Nil(t, ekus)
	require.Empty(t, unknownEkus)

	cfg = generateCertificate.Configuration{
		ExtensionKeyUsages: []string{"server", "client", "1.2.3.4.5"},
	}

	ekus, unknownEkus, err = cfg.X509ExtKeyUsages()
	require.NoError(t, err)
	require.Equal(t, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, ekus)
	expectedUnknownEkus := []asn1.ObjectIdentifier{{1, 2, 3, 4, 5}}
	require.Equal(t, expectedUnknownEkus, unknownEkus)
}

func TestAsnExtensionKeyUsages(t *testing.T) {
	cfg := generateCertificate.Configuration{
		ExtensionKeyUsages: []string{"invalid"},
	}
	_, err := cfg.AsnExtensionKeyUsages()
	require.Error(t, err)

	cfg = generateCertificate.Configuration{
		ExtensionKeyUsages: []string{"server", "client", "1.2.3.4.5"},
	}

	ekus, err := cfg.AsnExtensionKeyUsages()
	require.NoError(t, err)
	expected := []asn1.ObjectIdentifier{{1, 3, 6, 1, 5, 5, 7, 3, 1}, {1, 3, 6, 1, 5, 5, 7, 3, 2}, {1, 2, 3, 4, 5}}
	require.Equal(t, expected, ekus)
}

func TestToIPAddresses(t *testing.T) {
	cfg := generateCertificate.Configuration{}
	cfg.SubjectAlternativeName.IPAddresses = []string{"not an IP address"}
	_, err := cfg.ToIPAddresses()
	require.Error(t, err)

	cfg = generateCertificate.Configuration{}
	cfg.SubjectAlternativeName.IPAddresses = []string{"192.168.0.1", "2001:0db8:85a3:0000:0000:8a2e:0370:7334"}
	ips, err := cfg.ToIPAddresses()
	require.NoError(t, err)
	expected := []net.IP{net.ParseIP("192.168.0.1"), net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")}
	require.Equal(t, expected, ips)
}

func TestToCRLDistributionPoints(t *testing.T) {
	tests := []struct {
		name    string
		cfg     generateCertificate.Configuration
		want    []string
		wantErr bool
	}{
		{
			name: "Valid CRL URLs",
			cfg: generateCertificate.Configuration{
				CRLDistributionPoints: []string{
					"http://example.com/crl1",
					"http://example.com/crl2",
				},
			},
			want: []string{"http://example.com/crl1", "http://example.com/crl2"},
		},
		{
			name: "Duplicate CRL URLs",
			cfg: generateCertificate.Configuration{
				CRLDistributionPoints: []string{
					"http://example.com/crl1",
					"http://example.com/crl1", // duplicate
				},
			},
			want: []string{"http://example.com/crl1"},
		},
		{
			name: "Invalid CRL URL",
			cfg: generateCertificate.Configuration{
				CRLDistributionPoints: []string{
					"invalid-url",
				},
			},
			wantErr: true,
		},
		{
			name: "Empty CRL list",
			cfg: generateCertificate.Configuration{
				CRLDistributionPoints: []string{},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crls, err := tt.cfg.ToCRLDistributionPoints()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.ElementsMatch(t, tt.want, crls)
		})
	}
}

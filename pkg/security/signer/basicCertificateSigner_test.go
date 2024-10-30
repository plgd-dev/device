package signer_test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/device/v2/pkg/security/signer"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func TestNewBasicCertificateSigner(t *testing.T) {
	caCert, err := pkgX509.ReadPemCertificates(os.Getenv("INTERMEDIATE_CA_CRT"))
	require.NoError(t, err)
	caKey, err := pkgX509.ReadPemEcdsaPrivateKey(os.Getenv("INTERMEDIATE_CA_KEY"))
	require.NoError(t, err)
	type args struct {
		caCert         []*x509.Certificate
		caKey          crypto.PrivateKey
		validNotBefore time.Time
		validNotAfter  time.Time
		crlPoints      []string
	}
	tests := []struct {
		name    string
		args    args
		want    []*x509.Certificate
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now(),
				validNotAfter:  time.Now().Add(time.Hour * 86400),
			},
			wantErr: false,
		},
		{
			name: "invalid CRL address",
			args: args{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now().Add(-time.Second),
				validNotAfter:  time.Now().Add(time.Hour),
				crlPoints:      []string{"invalid-crl-address"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := signer.NewBasicCertificateSigner(tt.args.caCert, tt.args.caKey, tt.args.validNotBefore, tt.args.validNotAfter, tt.args.crlPoints)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBasicCertificateSignerSign(t *testing.T) {
	caCert, err := pkgX509.ReadPemCertificates(os.Getenv("INTERMEDIATE_CA_CRT"))
	require.NoError(t, err)
	caKey, err := pkgX509.ReadPemEcdsaPrivateKey(os.Getenv("INTERMEDIATE_CA_KEY"))
	require.NoError(t, err)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csrCfg := generateCertificate.Configuration{}
	csrCfg.Subject.CommonName = "uuid:00000000-0000-0000-0000-000000000001"
	csrCfg.ExtensionKeyUsages = []string{"server", "client", coap.ExtendedKeyUsage_IDENTITY_CERTIFICATE.String()}
	csr, err := generateCertificate.GenerateCSR(csrCfg, priv)
	require.NoError(t, err)

	type fields struct {
		caCert         []*x509.Certificate
		caKey          crypto.PrivateKey
		validNotBefore time.Time
		validNotAfter  time.Time
		crlPoints      []string
	}
	type args struct {
		csr []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*x509.Certificate
		wantErr bool
	}{
		{
			name: "invalid",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now(),
				validNotAfter:  time.Now().Add(time.Hour * 86400),
			},
			wantErr: true,
		},
		{
			name: "valid",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now(),
				validNotAfter:  time.Now().Add(time.Hour * 86400),
			},
			args: args{
				csr: csr,
			},
			wantErr: false,
			want: []*x509.Certificate{
				{
					Subject: pkix.Name{
						CommonName: "uuid:00000000-0000-0000-0000-000000000001",
					},
					ExtKeyUsage: []x509.ExtKeyUsage{
						x509.ExtKeyUsageServerAuth,
						x509.ExtKeyUsageClientAuth,
					},
					UnknownExtKeyUsage: []asn1.ObjectIdentifier{coap.ExtendedKeyUsage_IDENTITY_CERTIFICATE},
				},
				{
					Subject: pkix.Name{
						CommonName: "intermediateCA",
					},
				},
			},
		},
		{
			name: "valid with CRL points",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now().Add(-time.Second),
				validNotAfter:  time.Now().Add(time.Hour),
				crlPoints:      []string{"http://example.com/crl"},
			},
			args: args{
				csr: csr,
			},
			want: []*x509.Certificate{
				{
					Subject: pkix.Name{
						CommonName: "uuid:00000000-0000-0000-0000-000000000001",
					},
					ExtKeyUsage: []x509.ExtKeyUsage{
						x509.ExtKeyUsageServerAuth,
						x509.ExtKeyUsageClientAuth,
					},
					UnknownExtKeyUsage:    []asn1.ObjectIdentifier{coap.ExtendedKeyUsage_IDENTITY_CERTIFICATE},
					CRLDistributionPoints: []string{"http://example.com/crl"},
				},
				{
					Subject: pkix.Name{
						CommonName: "intermediateCA",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := signer.NewBasicCertificateSigner(tt.fields.caCert, tt.fields.caKey, tt.fields.validNotBefore, tt.fields.validNotAfter, tt.fields.crlPoints)
			require.NoError(t, err)
			got, err := s.Sign(context.Background(), tt.args.csr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			for i := 0; i < len(tt.want); i++ {
				block, rest := pem.Decode(got)
				require.NotEmpty(t, block.Bytes)
				cert, err := x509.ParseCertificate(block.Bytes)
				require.NoError(t, err)
				require.Equal(t, tt.want[i].Subject.CommonName, cert.Subject.CommonName)
				require.Equal(t, tt.want[i].ExtKeyUsage, cert.ExtKeyUsage)
				require.Equal(t, tt.want[i].UnknownExtKeyUsage, cert.UnknownExtKeyUsage)
				require.Equal(t, tt.want[i].CRLDistributionPoints, cert.CRLDistributionPoints)
				got = rest
			}
		})
	}
}

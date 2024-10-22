package signer_test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/device/v2/pkg/security/signer"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func TestNewOCFIdentityCertificate(t *testing.T) {
	type args struct {
		caCert         []*x509.Certificate
		caKey          crypto.PrivateKey
		validNotBefore time.Time
		validNotAfter  time.Time
		crlPoints      []string
	}
	caCert, err := pkgX509.ReadPemCertificates(os.Getenv("ROOT_CA_CRT"))
	require.NoError(t, err)
	caKey, err := pkgX509.ReadPemEcdsaPrivateKey(os.Getenv("ROOT_CA_KEY"))
	require.NoError(t, err)

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now().Add(-time.Second),
				validNotAfter:  time.Now().Add(time.Hour),
			},
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
			_, err := signer.NewOCFIdentityCertificate(tt.args.caCert, tt.args.caKey, tt.args.validNotBefore, tt.args.validNotAfter, tt.args.crlPoints)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestOCFIdentityCertificateSign(t *testing.T) {
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
	caCert, err := pkgX509.ReadPemCertificates(os.Getenv("ROOT_CA_CRT"))
	require.NoError(t, err)
	caKey, err := pkgX509.ReadPemEcdsaPrivateKey(os.Getenv("ROOT_CA_KEY"))
	require.NoError(t, err)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	id := uuid.NewString()
	csr, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, id, priv)
	require.NoError(t, err)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now().Add(-time.Second),
				validNotAfter:  time.Now().Add(time.Hour),
			},
			args: args{
				csr: csr,
			},
		},
		{
			name: "valid - bigger time range than ca signer chain supported",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now().Add(-time.Hour * 8760 * 10),
				validNotAfter:  time.Now().Add(time.Hour * 8760 * 10),
			},
			args: args{
				csr: csr,
			},
		},
		{
			name: "invalid time range",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now().Add(time.Hour),
				validNotAfter:  time.Now().Add(-time.Second),
			},
			args: args{
				csr: csr,
			},
			wantErr: true,
		},
		{
			name: "invalid time range - now before not before",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now().Add(time.Hour),
				validNotAfter:  time.Now().Add(time.Hour * 2),
			},
			args: args{
				csr: csr,
			},
			wantErr: true,
		},
		{
			name: "invalid time range - now after not after",
			fields: fields{
				caCert:         caCert,
				caKey:          caKey,
				validNotBefore: time.Now(),
				validNotAfter:  time.Now().Add(time.Nanosecond),
			},
			args: args{
				csr: csr,
			},
			wantErr: true,
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := signer.NewOCFIdentityCertificate(tt.fields.caCert, tt.fields.caKey, tt.fields.validNotBefore, tt.fields.validNotAfter, tt.fields.crlPoints)
			require.NoError(t, err)
			gotSignedCsr, err := s.Sign(context.Background(), tt.args.csr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, gotSignedCsr)
		})
	}
}

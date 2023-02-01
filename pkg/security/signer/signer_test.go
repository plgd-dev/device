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
	"github.com/plgd-dev/kit/v2/security"
	"github.com/stretchr/testify/require"
)

func TestOCFIdentityCertificateSign(t *testing.T) {
	type fields struct {
		caCert         []*x509.Certificate
		caKey          crypto.PrivateKey
		validNotBefore time.Time
		validNotAfter  time.Time
	}
	type args struct {
		ctx context.Context
		csr []byte
	}
	caCert, err := security.LoadX509(os.Getenv("ROOT_CA_CRT"))
	require.NoError(t, err)
	caKey, err := security.LoadX509PrivateKey(os.Getenv("ROOT_CA_KEY"))
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
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
				ctx: context.Background(),
				csr: csr,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := signer.NewOCFIdentityCertificate(tt.fields.caCert, tt.fields.caKey, tt.fields.validNotBefore, tt.fields.validNotAfter)
			gotSignedCsr, err := s.Sign(tt.args.ctx, tt.args.csr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, gotSignedCsr)
		})
	}
}

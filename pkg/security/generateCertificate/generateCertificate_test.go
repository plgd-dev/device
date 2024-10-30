package generateCertificate_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"slices"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/stretchr/testify/require"
)

func TestGenerateCSR(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	validCfg := func() generateCertificate.Configuration {
		cfg := generateCertificate.Configuration{}
		cfg.Subject.Country = []string{"US"}
		cfg.Subject.Organization = []string{"TestOrg"}
		cfg.Subject.CommonName = "test.example.com"
		cfg.SubjectAlternativeName.DNSNames = []string{"example.com"}
		cfg.BasicConstraints.Ignore = true
		cfg.BasicConstraints.MaxPathLen = -1
		cfg.ValidFor = time.Hour * 24
		return cfg
	}()

	type args struct {
		cfg generateCertificate.Configuration
	}
	tests := []struct {
		name    string
		args    args
		verify  func(t *testing.T, cfg generateCertificate.Configuration, parsedCSR *x509.CertificateRequest)
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				cfg: validCfg,
			},
			verify: func(t *testing.T, cfg generateCertificate.Configuration, parsedCSR *x509.CertificateRequest) {
				require.Equal(t, cfg.Subject.CommonName, parsedCSR.Subject.CommonName)
				require.ElementsMatch(t, cfg.Subject.Country, parsedCSR.Subject.Country)
				require.ElementsMatch(t, cfg.Subject.Organization, parsedCSR.Subject.Organization)
				require.ElementsMatch(t, cfg.SubjectAlternativeName.DNSNames, parsedCSR.DNSNames)
			},
		},
		{
			name: "valid with basic constraint",
			args: args{
				cfg: func() generateCertificate.Configuration {
					cfg := validCfg
					cfg.BasicConstraints.Ignore = false
					cfg.BasicConstraints.MaxPathLen = 42
					return cfg
				}(),
			},
			verify: func(t *testing.T, cfg generateCertificate.Configuration, parsedCSR *x509.CertificateRequest) {
				require.Equal(t, cfg.Subject.CommonName, parsedCSR.Subject.CommonName)
				require.ElementsMatch(t, cfg.Subject.Country, parsedCSR.Subject.Country)
				require.ElementsMatch(t, cfg.Subject.Organization, parsedCSR.Subject.Organization)
				require.ElementsMatch(t, cfg.SubjectAlternativeName.DNSNames, parsedCSR.DNSNames)
				var bcExt *pkix.Extension
				for _, ext := range parsedCSR.Extensions {
					if slices.Equal(ext.Id, generateCertificate.ASN1BasicConstraints) {
						bcExt = &ext
					}
				}
				require.NotNil(t, bcExt)
				require.False(t, bcExt.Critical)
				var bc generateCertificate.BasicConstraints
				_, err := asn1.Unmarshal(bcExt.Value, &bc)
				require.NoError(t, err)
				require.False(t, bc.CA)
			},
		},
		{
			name: "valid with key usages",
			args: args{
				cfg: func() generateCertificate.Configuration {
					cfg := validCfg
					cfg.KeyUsages = append(cfg.KeyUsages, "digitalSignature", "certSign")
					return cfg
				}(),
			},
			verify: func(t *testing.T, cfg generateCertificate.Configuration, parsedCSR *x509.CertificateRequest) {
				require.Equal(t, cfg.Subject.CommonName, parsedCSR.Subject.CommonName)
				require.ElementsMatch(t, cfg.Subject.Country, parsedCSR.Subject.Country)
				require.ElementsMatch(t, cfg.Subject.Organization, parsedCSR.Subject.Organization)
				require.ElementsMatch(t, cfg.SubjectAlternativeName.DNSNames, parsedCSR.DNSNames)
				var kuExt *pkix.Extension
				for _, ext := range parsedCSR.Extensions {
					if slices.Equal(ext.Id, generateCertificate.ASN1KeyUsage) {
						kuExt = &ext
					}
				}
				require.NotNil(t, kuExt)
				require.False(t, kuExt.Critical)
				var ku asn1.BitString
				_, err := asn1.Unmarshal(kuExt.Value, &ku)
				require.NoError(t, err)
				keyUsages, err := cfg.AsnKeyUsages()
				require.NoError(t, err)
				require.Equal(t, keyUsages, ku)
			},
		},
		{
			name: "valid with extended key usages",
			args: args{
				cfg: func() generateCertificate.Configuration {
					cfg := validCfg
					cfg.ExtensionKeyUsages = append(cfg.ExtensionKeyUsages,
						asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 3}.String(),
						asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 3, 9}.String())
					return cfg
				}(),
			},
			verify: func(t *testing.T, cfg generateCertificate.Configuration, parsedCSR *x509.CertificateRequest) {
				require.Equal(t, cfg.Subject.CommonName, parsedCSR.Subject.CommonName)
				require.ElementsMatch(t, cfg.Subject.Country, parsedCSR.Subject.Country)
				require.ElementsMatch(t, cfg.Subject.Organization, parsedCSR.Subject.Organization)
				require.ElementsMatch(t, cfg.SubjectAlternativeName.DNSNames, parsedCSR.DNSNames)
				var ekuExt *pkix.Extension
				for _, ext := range parsedCSR.Extensions {
					if slices.Equal(ext.Id, generateCertificate.ASN1ExtKeyUsage) {
						ekuExt = &ext
					}
				}
				require.NotNil(t, ekuExt)
				require.False(t, ekuExt.Critical)
				var ekus []asn1.ObjectIdentifier
				_, err := asn1.Unmarshal(ekuExt.Value, &ekus)
				require.NoError(t, err)
				x509ExtKeyUsages, unknownKeyUsages, err := cfg.X509ExtKeyUsages()
				require.NoError(t, err)
				var extKeyUsages []asn1.ObjectIdentifier
				for _, e := range x509ExtKeyUsages {
					eku, ok := generateCertificate.OidFromExtKeyUsage(e)
					require.True(t, ok)
					extKeyUsages = append(extKeyUsages, eku)
				}
				extKeyUsages = append(extKeyUsages, unknownKeyUsages...)
				require.ElementsMatch(t, extKeyUsages, ekus)
			},
		},
		{
			name: "invalid IP address",
			args: args{
				cfg: func() generateCertificate.Configuration {
					cfg := validCfg
					cfg.SubjectAlternativeName.IPAddresses = []string{"invalid-ip"}
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid signature algorithm",
			args: args{
				cfg: func() generateCertificate.Configuration {
					cfg := validCfg
					cfg.SignatureAlgorithm = "invalid-algorithm"
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid key usages",
			args: args{
				cfg: func() generateCertificate.Configuration {
					cfg := validCfg
					cfg.KeyUsages = []string{"invalid-usage"}
					return cfg
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid extended key usages",
			args: args{
				cfg: func() generateCertificate.Configuration {
					cfg := validCfg
					cfg.ExtensionKeyUsages = []string{"invalid-eku"}
					return cfg
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			csr, err := generateCertificate.GenerateCSR(tt.args.cfg, privateKey)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, csr)

			// Parse the PEM-encoded CSR and verify it's well-formed
			block, _ := pem.Decode(csr)
			require.NotNil(t, block)
			require.Equal(t, "CERTIFICATE REQUEST", block.Type)

			parsedCSR, err := x509.ParseCertificateRequest(block.Bytes)
			require.NoError(t, err)
			tt.verify(t, tt.args.cfg, parsedCSR)
		})
	}
}

func TestGenerateCertificate(t *testing.T) {
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
				cfg: generateCertificate.Configuration{
					ValidFor: time.Minute,
				},
			},
		},
		{
			name: "valid - sha384",
			args: args{
				cfg: generateCertificate.Configuration{
					ValidFor:           time.Minute,
					SignatureAlgorithm: generateCertificate.SignatureAlgorithmECDSAWithSHA384,
				},
			},
		},
		{
			name: "valid - sha512",
			args: args{
				cfg: generateCertificate.Configuration{
					ValidFor:           time.Minute,
					SignatureAlgorithm: generateCertificate.SignatureAlgorithmECDSAWithSHA512,
				},
			},
		},
		{
			name: "valid - p384",
			args: args{
				cfg: generateCertificate.Configuration{
					ValidFor:      time.Minute,
					EllipticCurve: generateCertificate.EllipticCurveP384,
				},
			},
		},
		{
			name: "valid - p521",
			args: args{
				cfg: generateCertificate.Configuration{
					ValidFor:      time.Minute,
					EllipticCurve: generateCertificate.EllipticCurveP521,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caCrt, caKey := generateRootCA(t, tt.args.cfg)
			privateKey, err := tt.args.cfg.GenerateKey()
			require.NoError(t, err)
			got, err := generateCertificate.GenerateCert(tt.args.cfg, privateKey, caCrt, caKey)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, got)
		})
	}
}

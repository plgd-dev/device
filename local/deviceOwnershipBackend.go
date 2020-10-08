package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/plgd-dev/kit/security"
	"github.com/plgd-dev/sdk/local/core"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"github.com/plgd-dev/cloud/certificate-authority/pb"
	caSigner "github.com/plgd-dev/cloud/certificate-authority/signer"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

type deviceOwnershipBackend struct {
	caClient            pb.CertificateAuthorityClient
	caConn              *grpc.ClientConn
	identityCertificate tls.Certificate
	identityCACert      *x509.Certificate
	authCodeURL         string
	accessTokenURL      string
	jwtClaimOwnerID     string
	app                 ApplicationCallback
	disableDTLS         bool
}

type DeviceOwnershipBackendConfig struct {
	SigningServerAddress string
	AuthCodeURL          string
	AccessTokenURL       string
	JWTClaimOwnerID      string
}

func validateURL(URL string) error {
	if URL == "" {
		return fmt.Errorf("empty url")
	}
	_, err := url.Parse(URL)
	if err != nil {
		return err
	}
	return nil
}

func NewDeviceOwnershipBackendFromConfig(app ApplicationCallback, cfg *DeviceOwnershipBackendConfig, disableDTLS bool, errorsFunc func(err error)) (*deviceOwnershipBackend, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing device ownership backend config")
	}

	if cfg.JWTClaimOwnerID == "" {
		cfg.JWTClaimOwnerID = "sub"
	}

	err := validateURL(cfg.AuthCodeURL)
	if err != nil {
		return nil, fmt.Errorf("invalid AuthCodeURL: %w", err)
	}

	err = validateURL(cfg.AccessTokenURL)
	if err != nil {
		return nil, fmt.Errorf("invalid AccessTokenURL: %w", err)
	}

	rootCA, err := app.GetRootCertificateAuthorities()
	if err != nil {
		return nil, fmt.Errorf("cannot get root CAs: %w", err)
	}

	conn, err := grpc.Dial(cfg.SigningServerAddress, grpc.WithTransportCredentials(credentials.NewTLS(security.NewDefaultTLSConfig(rootCA))))
	if err != nil {
		return nil, fmt.Errorf("cannot create certificate authority client: %w", err)
	}
	caClient := pb.NewCertificateAuthorityClient(conn)

	return &deviceOwnershipBackend{
		caClient:        caClient,
		caConn:          conn,
		accessTokenURL:  cfg.AccessTokenURL,
		authCodeURL:     cfg.AuthCodeURL,
		app:             app,
		jwtClaimOwnerID: cfg.JWTClaimOwnerID,
		disableDTLS:     disableDTLS,
	}, nil
}

func (o *deviceOwnershipBackend) OwnDevice(ctx context.Context, deviceID string, own ownFunc, opts ...core.OwnOption) (string, error) {
	identCert := caSigner.NewIdentityCertificateSigner(o.caClient)
	otm, err := getOTMManufacturer(o.app, o.disableDTLS, identCert)
	if err != nil {
		return "", err
	}
	return own(ctx, deviceID, otm, opts...)
}

type claims map[string]interface{}

func (c *claims) Valid() error {
	return nil
}

func (o *deviceOwnershipBackend) setIdentityCertificate(ctx context.Context, accessToken string) error {
	parser := &jwt.Parser{
		SkipClaimsValidation: true,
	}
	var claims claims
	_, _, err := parser.ParseUnverified(accessToken, &claims)
	if err != nil {
		return fmt.Errorf("cannot parse jwt token: %w", err)
	}
	if claims[o.jwtClaimOwnerID] == nil {
		return fmt.Errorf("cannot get '%v' from jwt token: is not set", o.jwtClaimOwnerID)
	}
	ownerStr := fmt.Sprintf("%v", claims[o.jwtClaimOwnerID])
	deviceID := uuid.NewV5(uuid.NamespaceURL, ownerStr)

	signer := caSigner.NewIdentityCertificateSigner(o.caClient)
	cert, caCert, err := GenerateSDKIdentityCertificate(ctx, signer, deviceID.String())
	if err != nil {
		return err
	}

	o.identityCertificate = cert
	o.identityCACert = caCert

	return nil
}

func (o *deviceOwnershipBackend) Initialization(ctx context.Context) error {
	token, err := kitNetGrpc.TokenFromOutgoingMD(ctx)
	if err != nil {
		return err
	}
	return o.setIdentityCertificate(ctx, token)
}

func (o *deviceOwnershipBackend) GetIdentityCertificate() (tls.Certificate, error) {
	if o.identityCertificate.PrivateKey == nil {
		return tls.Certificate{}, fmt.Errorf("client is not initialized")
	}
	return o.identityCertificate, nil
}

func (o *deviceOwnershipBackend) GetIdentityCACerts() ([]*x509.Certificate, error) {
	if o.identityCACert == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	return []*x509.Certificate{o.identityCACert}, nil
}

func (o *deviceOwnershipBackend) GetAccessTokenURL(ctx context.Context) (string, error) {
	return o.accessTokenURL, nil
}

func (o *deviceOwnershipBackend) GetOnboardAuthorizationCodeURL(ctx context.Context, deviceID string) (string, error) {
	if deviceID == "" {
		return "", fmt.Errorf("invalid deviceID: empty")
	}
	_, err := uuid.FromString(deviceID)
	if err != nil {
		return "", fmt.Errorf("invalid deviceID: %w", err)
	}

	u, err := url.Parse(o.authCodeURL)
	if err != nil {
		return "", err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", err
	}
	q.Add("deviceId", deviceID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (o *deviceOwnershipBackend) Close(ctx context.Context) error {
	return o.caConn.Close()
}

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/util/metautils"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/client/core/otm"
	justworks "github.com/plgd-dev/device/client/core/otm/just-works"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SignFunc = func(ctx context.Context, csr []byte) (signedCsr []byte, err error)

type deviceOwnershipBackend struct {
	identityCertificate tls.Certificate
	identityCACert      []*x509.Certificate
	jwtClaimOwnerID     string
	app                 ApplicationCallback
	dialTLS             core.DialTLS
	dialDTLS            core.DialDTLS
	sign                SignFunc
}

type DeviceOwnershipBackendConfig struct {
	JWTClaimOwnerID string
	Sign            SignFunc
}

func NewDeviceOwnershipBackendFromConfig(app ApplicationCallback, dialTLS core.DialTLS, dialDTLS core.DialDTLS,
	cfg *DeviceOwnershipBackendConfig, errorsFunc func(err error)) (*deviceOwnershipBackend, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing device ownership backend config")
	}

	if cfg.JWTClaimOwnerID == "" {
		cfg.JWTClaimOwnerID = "sub"
	}

	return &deviceOwnershipBackend{
		sign:            cfg.Sign,
		app:             app,
		jwtClaimOwnerID: cfg.JWTClaimOwnerID,
		dialTLS:         dialTLS,
		dialDTLS:        dialDTLS,
	}, nil
}

func (o *deviceOwnershipBackend) OwnDevice(ctx context.Context, deviceID string, otmType OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error) {
	var otmClient otm.Client
	opts = append([]core.OwnOption{core.WithSetupCertificates(o.sign)}, opts...)
	switch otmType {
	case OTMType_Manufacturer:
		otm, err := getOTMManufacturer(o.app, o.dialTLS, o.dialDTLS)
		if err != nil {
			return "", err
		}
		otmClient = otm
	case OTMType_JustWorks:
		otmClient = justworks.NewClient(justworks.WithDialDTLS(o.dialDTLS))
	default:
		return "", fmt.Errorf("unsupported ownership transfer method: %v", otmType)
	}
	return own(ctx, deviceID, otmClient, discoveryConfiguration, opts...)
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
	ownerID, err := uuid.Parse(ownerStr)
	if err != nil || ownerStr == uuid.Nil.String() {
		ownerID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(ownerStr))
	}
	cert, caCert, err := GenerateSDKIdentityCertificate(ctx, o.sign, ownerID.String())
	if err != nil {
		return err
	}

	o.identityCertificate = cert
	o.identityCACert = caCert

	return nil
}

var (
	headerAuthorize = "authorization"
)

// TokenFromOutgoingMD extracts token stored by CtxWithToken.
func TokenFromOutgoingMD(ctx context.Context) (string, error) {
	expectedScheme := "bearer"
	val := metautils.ExtractOutgoing(ctx).Get(headerAuthorize)
	if val == "" {
		return "", status.Errorf(codes.Unauthenticated, "Request unauthenticated with "+expectedScheme)
	}
	splits := strings.SplitN(val, " ", 2)
	if len(splits) < 2 {
		return "", status.Errorf(codes.Unauthenticated, "Bad authorization string")
	}
	if !strings.EqualFold(splits[0], expectedScheme) {
		return "", status.Errorf(codes.Unauthenticated, "Request unauthenticated with %v, expected %v", splits[0], expectedScheme)
	}
	return splits[1], nil
}

func (o *deviceOwnershipBackend) Initialization(ctx context.Context) error {
	token, err := TokenFromOutgoingMD(ctx)
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
	return o.identityCACert, nil
}

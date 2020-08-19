package app

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/plgd-dev/kit/security"
)

type AppConfig struct {
	RootCA       string
	Manufacturer *ManufacturerCerts
}

type ManufacturerCerts struct {
	CA, Cert, CertKey string
}

type App struct {
	rootCA, manufacturerCA []*x509.Certificate
	manufacturerCert       *tls.Certificate
}

func NewApp(cfg *AppConfig) (*App, error) {
	if cfg == nil {
		return &App{}, nil
	}
	var rootCA, manufacturerCA []*x509.Certificate
	var manufacturerCert tls.Certificate
	var err error
	if len(cfg.RootCA) != 0 {
		rootCA, err = security.ParseX509FromPEM([]byte(cfg.RootCA))
		if err != nil {
			return nil, fmt.Errorf("invalid Root CA: %w", err)
		}
	}
	if cfg.Manufacturer != nil && len(cfg.Manufacturer.CA) != 0 {
		manufacturerCA, err = security.ParseX509FromPEM([]byte(cfg.Manufacturer.CA))
		if err != nil {
			return nil, fmt.Errorf("invalid Manufacturer's CA: %w", err)
		}
	}
	if cfg.Manufacturer != nil && len(cfg.Manufacturer.Cert) != 0 && len(cfg.Manufacturer.CertKey) != 0 {
		manufacturerCert, err = tls.X509KeyPair([]byte(cfg.Manufacturer.Cert), []byte(cfg.Manufacturer.CertKey))
		if err != nil {
			return nil, fmt.Errorf("invalid Manufacturer's cert: %w", err)
		}
	}
	a := App{
		rootCA:           rootCA,
		manufacturerCA:   manufacturerCA,
		manufacturerCert: &manufacturerCert,
	}
	return &a, nil
}

func (a *App) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(a.rootCA) == 0 {
		return nil, fmt.Errorf("missing Root CA")
	}
	return a.rootCA, nil
}

func (a *App) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(a.manufacturerCA) == 0 {
		return nil, fmt.Errorf("missing Manufacturer's CA")
	}
	return a.manufacturerCA, nil
}

func (a *App) GetManufacturerCertificate() (r tls.Certificate, _ error) {
	if a.manufacturerCert == nil {
		return r, fmt.Errorf("missing Manufacturer's certificate")
	}
	return *a.manufacturerCert, nil
}

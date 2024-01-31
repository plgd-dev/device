package main

import (
	"fmt"

	"github.com/plgd-dev/device/v2/bridge/service"
)

type TLSConfig struct {
	CAPoolPath      string `yaml:"caPoolPath" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyPath         string `yaml:"keyPath" json:"keyFile" description:"file path to the private key in PEM format"`
	CertPath        string `yaml:"certPath" json:"certFile" description:"file path to the certificate in PEM format"`
	UseSystemCAPool bool   `yaml:"useSystemCAPool" json:"useSystemCaPool" description:"use system certification pool"`
}

func (c *TLSConfig) Validate() error {
	if c.CAPoolPath == "" && !c.UseSystemCAPool {
		return fmt.Errorf("caPool is required")
	}
	if (c.KeyPath == "" && c.CertPath != "") || (c.KeyPath != "" && c.CertPath == "") {
		return fmt.Errorf("keyFile and certFile must be set together")
	}
	return nil
}

type CloudConfig struct {
	Enabled bool      `yaml:"enabled" json:"enabled" description:"enable cloud connection"`
	TLS     TLSConfig `yaml:"tls" json:"tls"`
}

func (c *CloudConfig) Validate() error {
	if c.Enabled {
		return c.TLS.Validate()
	}
	return nil
}

type Config struct {
	service.Config             `yaml:",inline"`
	Cloud                      CloudConfig `yaml:"cloud" json:"cloud"`
	NumGeneratedBridgedDevices int         `yaml:"numGeneratedBridgedDevices"`
	NumResourcesPerDevice      int         `yaml:"numResourcesPerDevice"`
}

func (c *Config) Validate() error {
	if err := c.Config.Validate(); err != nil {
		return err
	}
	if c.NumGeneratedBridgedDevices <= 0 {
		return fmt.Errorf("numGeneratedBridgedDevices - must be > 0")
	}
	return nil
}

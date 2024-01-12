package service

import (
	"fmt"

	"github.com/plgd-dev/device/v2/bridge/net"
)

type CoAPConfig struct {
	ID         string `yaml:"id"`
	net.Config `yaml:",inline"`
}

func (c *CoAPConfig) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("id is required")
	}
	if err := c.Config.Validate(); err != nil {
		return err
	}
	return nil
}

type APIConfig struct {
	CoAP CoAPConfig `yaml:"coap"`
}

func (c *APIConfig) Validate() error {
	if err := c.CoAP.Validate(); err != nil {
		return fmt.Errorf("coap.%w", err)
	}
	return nil
}

type Config struct {
	API APIConfig `yaml:"apis"`
}

func (c *Config) Validate() error {
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("api.%w", err)
	}
	return nil
}

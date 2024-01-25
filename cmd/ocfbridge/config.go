package main

import (
	"fmt"

	"github.com/plgd-dev/device/v2/bridge/service"
)

type Config struct {
	service.Config             `yaml:",inline"`
	NumGeneratedBridgedDevices int `yaml:"numGeneratedBridgedDevices"`
	NumResourcesPerDevice      int `yaml:"numResourcesPerDevice"`
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

package device

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/plgd-dev/device/v2/pkg/log"
	"gopkg.in/yaml.v3"
)

type TLSConfig struct {
	CAPoolPath      string `yaml:"caPoolPath" json:"caPool" description:"file path to the root certificates in PEM format"`
	KeyPath         string `yaml:"keyPath" json:"keyFile" description:"file path to the private key in PEM format"`
	CertPath        string `yaml:"certPath" json:"certFile" description:"file path to the certificate in PEM format"`
	UseSystemCAPool bool   `yaml:"useSystemCAPool" json:"useSystemCaPool" description:"use system certification pool"`
}

func (c *TLSConfig) Validate() error {
	if c.CAPoolPath == "" && !c.UseSystemCAPool {
		return errors.New("caPool is required")
	}
	if (c.KeyPath == "" && c.CertPath != "") || (c.KeyPath != "" && c.CertPath == "") {
		return errors.New("keyFile and certFile must be set together")
	}
	return nil
}

type LogConfig struct {
	Level log.Level `yaml:"level" json:"level" description:"log level"`
}

type CloudConfig struct {
	Enabled bool      `yaml:"enabled" json:"enabled" description:"enable cloud connection"`
	CloudID string    `yaml:"cloudID" json:"cloudID" description:"cloud id"`
	TLS     TLSConfig `yaml:"tls" json:"tls"`
}

type CredentialConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled" description:"enable credential manager"`
}

type ThingDescriptionConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled" description:"enable thing description"`
	File    string `yaml:"file" json:"file" description:"file path to the thing description"`
}

func (c *CloudConfig) Validate() error {
	if c.Enabled {
		return c.TLS.Validate()
	}
	return nil
}

type Config struct {
	service.Config             `yaml:",inline"`
	Log                        LogConfig              `yaml:"log" json:"log"`
	Cloud                      CloudConfig            `yaml:"cloud" json:"cloud"`
	Credential                 CredentialConfig       `yaml:"credential" json:"credential"`
	ThingDescription           ThingDescriptionConfig `yaml:"thingDescription" json:"thingDescription"`
	NumGeneratedBridgedDevices int                    `yaml:"numGeneratedBridgedDevices"`
	NumResourcesPerDevice      int                    `yaml:"numResourcesPerDevice"`
}

func (c *Config) Validate() error {
	if err := c.Config.Validate(); err != nil {
		return err
	}
	if c.NumGeneratedBridgedDevices <= 0 {
		return errors.New("numGeneratedBridgedDevices - must be > 0")
	}
	return nil
}

func LoadConfig(configFile string) (Config, error) {
	// Sanitize the configFile variable to ensure it only contains a valid file path
	configFile = filepath.Clean(configFile)
	f, err := os.Open(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}
	defer func() {
		_ = f.Close()
	}()
	var cfg Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return Config{}, fmt.Errorf("failed to decode config: %w", err)
	}
	if err = cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

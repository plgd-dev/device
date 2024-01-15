package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/service"
	"gopkg.in/yaml.v3"
)

func loadConfig(configFile string) (Config, error) {
	// Sanitize the configFile variable to ensure it only contains a valid file path
	configFile = filepath.Clean(configFile)
	f, err := os.Open(configFile)
	if err != nil {
		return Config{}, err
	}
	defer func() {
		_ = f.Close()
	}()
	var cfg Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func main() {
	configFile := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()
	cfg, err := loadConfig(*configFile)
	if err != nil {
		panic(err)
	}
	s, err := service.New(cfg.Config)
	if err != nil {
		panic(err)
	}
	for i := 0; i < cfg.NumGeneratedBridgedDevices; i++ {
		d, ok := s.CreateDevice(uuid.New(), fmt.Sprintf("bridged-device-%d", i), service.WithDeviceTypes([]string{"oic.d.virtual"}))
		if ok {
			d.Init()
		}
	}
	err = s.Serve()
	if err != nil {
		panic(err)
	}
}

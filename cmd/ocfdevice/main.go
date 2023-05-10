package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/service"
)

func createDevice(idx int, name, externalAddress, listenerAddress, protocolIndependentID string) {
	cfg := service.DeviceConfig{
		ID:                    uuid.NewString(),
		ProtocolIndependentID: protocolIndependentID,
		Name:                  fmt.Sprintf("%v-%v", name, idx),
		ResourceTypes:         []string{"oic.d.goDevice"},
		ExternalAddress:       externalAddress,
		ListenAddress:         listenerAddress,
	}
	d, err := service.NewDevice(cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		err = d.Close()
		if err != nil {
			panic(err)
		}
	}()
	err = d.Listen()
	if err != nil {
		panic(err)
	}
}

func main() {
	externalAddress := flag.String("externalAddress", "", "external DNS/IP address, used for discovery")
	listenerAddress := flag.String("listenAddress", "", "address to bind listener for incoming connections")
	protocolIndependentID := flag.String("protocolIndependentID", "", "protocol independent ID")
	name := flag.String("name", "GO-OCF-Device", "device name")
	numDevices := flag.Int("numDevices", 1, "number of devices to simulate")
	help := flag.Bool("help", false, "print help")
	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}
	if *externalAddress == "" {
		panic("externalIP is required")
	}
	if *protocolIndependentID == "" {
		*protocolIndependentID = uuid.NewString()
	}

	var wg sync.WaitGroup
	wg.Add(*numDevices)
	for i := 0; i < *numDevices; i++ {
		go func(idx int) {
			defer wg.Done()
			createDevice(idx, *name, *externalAddress, *listenerAddress, *protocolIndependentID)
		}(i)
	}
	wg.Wait()
}

package service

import (
	"fmt"
	"log"
	gonet "net"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/plgd-dev/go-coap/v3/udp/server"
)

type Device struct {
	cfg           DeviceConfig
	listener      *net.UDPConn
	server        *server.Server
	mcastListener *net.UDPConn
	mcastServer   *server.Server

	mux       *mux.Router
	resources *sync.Map[string, *Resource]
}

type DeviceConfig struct {
	ID                    string
	Name                  string
	ProtocolIndependentID string
	ResourceTypes         []string
	ExternalAddress       string
	ListenAddress         string
}

func (cfg *DeviceConfig) Validate() error {
	if cfg.ExternalAddress == "" {
		return fmt.Errorf("externalAddress is required")
	}
	if cfg.ProtocolIndependentID == "" {
		return fmt.Errorf("protocolIndependentID is required")
	}
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}

	if cfg.Name == "" {
		cfg.Name = "Unnamed"
	}
	return nil
}

func initConnectivity(listenAddress string) (*net.UDPConn, *net.UDPConn, error) {
	multicastAddr := "224.0.1.187:5683"

	mcastListener, err := net.NewListenUDP("udp4", multicastAddr)
	if err != nil {
		return nil, nil, err
	}

	ifaces, err := gonet.Interfaces()
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
	}

	a, err := gonet.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
	}

	var anySet bool
	for i := range ifaces {
		iface := ifaces[i]
		err = mcastListener.JoinGroup(&iface, a)
		if err == nil {
			anySet = true
		}
		if err != nil {
			log.Printf("cannot JoinGroup(%v, %v): %v", iface, a, err)
		}
	}
	if !anySet {
		_ = mcastListener.Close()
		return nil, nil, fmt.Errorf("cannot JoinGroup(%v): %v", a, err)
	}

	err = mcastListener.SetMulticastLoopback(true)
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
	}

	l, err := net.NewListenUDP("udp4", listenAddress)
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
	}
	return mcastListener, l, nil
}

func loggingMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		next.ServeCOAP(w, r)
		log.Printf("%v, req=%v resp =%v\n", w.Conn().RemoteAddr(), r.String(), w.Message().String())
	})
}

func NewDevice(cfg DeviceConfig) (*Device, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	m := mux.NewRouter()
	m.Use(loggingMiddleware)
	mcastListener, listener, err := initConnectivity(cfg.ListenAddress)
	if err != nil {
		return nil, err
	}
	cfg.ResourceTypes = append(cfg.ResourceTypes, device.ResourceType)
	d := &Device{
		cfg:           cfg,
		listener:      listener,
		mcastListener: mcastListener,
		server:        udp.NewServer(options.WithMux(m)),
		mcastServer:   udp.NewServer(options.WithMux(m)),
		mux:           m,
		resources:     sync.NewMap[string, *Resource](),
	}

	devRes := NewDeviceResource(d)
	err = d.AddResource(devRes.Resource)
	if err != nil {
		return nil, err
	}
	disRes := NewDiscoveryResource(d)
	err = d.AddResource(disRes.Resource)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Device) getEndpoints() schema.Endpoints {
	return schema.Endpoints{
		{
			URI: fmt.Sprintf("coap://%v:%v", d.cfg.ExternalAddress, d.listener.LocalAddr().(*gonet.UDPAddr).Port),
		},
	}
}

func (d *Device) Listen() error {
	go func() {
		err := d.mcastServer.Serve(d.mcastListener)
		if err != nil {
			log.Printf("mcastServer.Serve: %v", err)
		}
	}()
	return d.server.Serve(d.listener)
}

func (d *Device) Close() error {
	d.server.Stop()
	d.mcastServer.Stop()
	_ = d.mcastListener.Close()
	return d.listener.Close()
}

func (d *Device) AddResource(resource *Resource) error {
	err := d.mux.Handle(resource.Href, resource)
	if err != nil {
		return err
	}
	d.resources.Store(resource.Href, resource)
	return err
}

func (d *Device) RemoveResource(resource *Resource) error {
	d.resources.Delete(resource.Href)
	return d.mux.HandleRemove(resource.Href)
}

/****************************************************************************
 *
 * Copyright (c) 2023 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package cloud

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	goSync "sync"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/bridge/resources/discovery"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/device"
	plgdResources "github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
)

type GetLinksFilteredBy func(endpoints schema.Endpoints, deviceIDfilter uuid.UUID, resourceTypesFitler []string, policyBitMaskFitler schema.BitMask) (links schema.ResourceLinks)

type Config struct {
	AccessToken           string
	UserID                string
	RefreshToken          string
	ValidUntil            time.Time
	AuthorizationProvider string
	CloudID               string
	CloudURL              string
}

type Manager struct {
	handler        net.RequestHandler
	getLinks       GetLinksFilteredBy
	maxMessageSize uint32
	deviceID       uuid.UUID
	save           func()

	private struct {
		mutex goSync.Mutex
		cfg   Configuration
	}

	creds              CoapSignUpResponse
	client             *client.Conn
	signedIn           bool
	resourcesPublished bool
	done               chan struct{}
	trigger            chan bool
}

func New(deviceID uuid.UUID, save func(), handler net.RequestHandler, getLinks GetLinksFilteredBy, maxMessageSize uint32) *Manager {
	c := &Manager{
		done:           make(chan struct{}),
		trigger:        make(chan bool, 10),
		handler:        handler,
		getLinks:       getLinks,
		maxMessageSize: maxMessageSize,
		deviceID:       deviceID,
		save:           save,
	}
	c.private.cfg.ProvisioningStatus = cloud.ProvisioningStatus_UNINITIALIZED
	return c
}

func (c *Manager) Get(request *net.Request) (*pool.Message, error) {
	cfg := c.getCloudConfiguration()
	return resources.CreateResponseContent(request.Context(), cfg, codes.Content)
}

func (c *Manager) ExportConfig() Config {
	configuration := c.getCloudConfiguration()
	creds := c.getCreds()
	return Config{
		CloudID:               configuration.CloudID,
		AuthorizationProvider: configuration.AuthorizationProvider,
		CloudURL:              configuration.URL,
		AccessToken:           creds.AccessToken,
		UserID:                creds.UserID,
		RefreshToken:          creds.RefreshToken,
		ValidUntil:            creds.ValidUntil,
	}
}

func (c *Manager) ImportConfig(cfg Config) {
	c.setCloudConfiguration(cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: cfg.AuthorizationProvider,
		URL:                   cfg.CloudURL,
		CloudID:               cfg.CloudID,
	})
	c.setCreds(CoapSignUpResponse{
		AccessToken:  cfg.AccessToken,
		UserID:       cfg.UserID,
		RefreshToken: cfg.RefreshToken,
		ValidUntil:   cfg.ValidUntil,
	})
}

func (c *Manager) Init() {
	if c.private.cfg.URL != "" {
		c.triggerRunner(false)
	}
	go c.run()
}

func (c *Manager) resetCredentials(ctx context.Context, signOff bool) {
	if signOff {
		resetCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		if err := c.signOff(resetCtx); err != nil {
			log.Printf("%v\n", err)
		}
	}
	c.creds = CoapSignUpResponse{}
	c.signedIn = false
	c.resourcesPublished = false
	if err := c.close(); err != nil {
		log.Printf("cannot close connection: %v\n", err)
	}
	c.save()
}

func (c *Manager) cleanup() {
	c.setCloudConfiguration(cloud.ConfigurationUpdateRequest{})
	c.resetCredentials(context.Background(), false)
	c.triggerRunner(false)
}

func (c *Manager) triggerRunner(reset bool) {
	select {
	case c.trigger <- reset:
	default:
	}
}

func validateConfigurationUpdate(cfg cloud.ConfigurationUpdateRequest) error {
	if cfg.CloudID == "" {
		return fmt.Errorf("cloud ID cannot be empty")
	}
	if cfg.AuthorizationProvider == "" {
		return fmt.Errorf("authorization provider cannot be empty")
	}
	if cfg.URL == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	return nil
}

func (c *Manager) Post(request *net.Request) (*pool.Message, error) {
	var cfg cloud.ConfigurationUpdateRequest
	err := cbor.ReadFrom(request.Body(), &cfg)
	if err != nil {
		return resources.CreateResponseBadRequest(request.Context(), err)
	}
	if cfg.URL == "" {
		c.setCloudConfiguration(cloud.ConfigurationUpdateRequest{})
	} else {
		err = validateConfigurationUpdate(cfg)
		if err != nil {
			return resources.CreateResponseBadRequest(request.Context(), err)
		}
		c.setCloudConfiguration(cfg)
	}
	c.triggerRunner(true)
	currentCfg := c.getCloudConfiguration()
	return resources.CreateResponseContent(request.Context(), currentCfg, codes.Changed)
}

func (c *Manager) setCloudConfiguration(cfg cloud.ConfigurationUpdateRequest) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.cfg.AuthorizationProvider = cfg.AuthorizationProvider
	c.private.cfg.CloudID = cfg.CloudID
	c.private.cfg.URL = cfg.URL
	c.private.cfg.AuthorizationCode = cfg.AuthorizationCode
	if cfg.URL == "" {
		c.private.cfg.ProvisioningStatus = cloud.ProvisioningStatus_UNINITIALIZED
	} else {
		c.private.cfg.ProvisioningStatus = cloud.ProvisioningStatus_READY_TO_REGISTER
	}
}

func (c *Manager) setProvisioningStatus(status cloud.ProvisioningStatus) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.cfg.ProvisioningStatus = status
}

func (c *Manager) getCloudConfiguration() Configuration {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	return c.private.cfg
}

func validUntil(expiresIn int64) time.Time {
	if expiresIn == -1 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(expiresIn) * time.Second)
}

func (c *Manager) setCreds(creds CoapSignUpResponse) {
	c.creds = creds
	c.signedIn = false
}

func (c *Manager) getCreds() CoapSignUpResponse {
	return c.creds
}

func (c *Manager) isCredsExpiring() bool {
	if !c.signedIn || c.creds.ValidUntil.IsZero() {
		return false
	}
	return !time.Now().Before(c.creds.ValidUntil.Add(-time.Second * 10))
}

func (c *Manager) serveCOAP(w mux.ResponseWriter, request *mux.Message) {
	request.Message.AddQuery("di=" + c.deviceID.String())
	r := net.Request{
		Message:   request.Message,
		Endpoints: nil,
		Conn:      w.Conn(),
	}
	var resp *pool.Message
	p, err := r.Path()
	if err == nil {
		switch p {
		case device.ResourceURI:
			links := c.getLinks(schema.Endpoints{}, c.deviceID, nil, resources.PublishToCloud)
			for _, link := range links {
				if link.HasType(device.ResourceType) {
					_ = r.SetPath(link.Href)
					break
				}
			}
			resp, err = c.handler(&r)
		case plgdResources.ResourceURI:
			links := c.getLinks(schema.Endpoints{}, c.deviceID, nil, resources.PublishToCloud)
			links = patchDeviceLink(links)
			links = discovery.PatchLinks(links, c.deviceID.String())
			resp, err = resources.CreateResponseContent(request.Context(), links, codes.Content)
		default:
			resp, err = c.handler(&r)
		}
	} else {
		resp, err = c.handler(&r)
	}
	if err != nil {
		resp = net.CreateResponseError(request.Context(), err, request.Token())
	}
	if resp != nil {
		resp.SetToken(w.Message().Token())
		resp.SetMessageID(w.Message().MessageID())
		resp.SetType(w.Message().Type())
		w.SetMessage(resp)
	}
}

func (c *Manager) close() error {
	c.signedIn = false
	if c.client == nil {
		return nil
	}
	client := c.client
	c.client = nil
	return client.Close()
}

func (c *Manager) dial(ctx context.Context) error {
	if c.client != nil && c.client.Context().Err() == nil {
		return nil
	}
	_ = c.close()
	cfg := c.getCloudConfiguration()
	tlsConfig := &tls.Config{
		// TODO: set RootCAs from configuration
		InsecureSkipVerify: true, //nolint:gosec
	}
	ep := schema.Endpoint{
		URI: cfg.URL,
	}
	addr, err := ep.GetAddr()
	if err != nil {
		return fmt.Errorf("cannot get address from %v: %w", ep, err)
	}
	m := mux.NewRouter()
	m.Use(net.LoggingMiddleware)
	m.DefaultHandle(mux.HandlerFunc(c.serveCOAP))
	conn, err := tcp.Dial(addr.String(),
		options.WithTLS(tlsConfig),
		options.WithMux(m),
		options.WithContext(ctx),
		options.WithMaxMessageSize(c.maxMessageSize),
		options.WithBlockwise(false, blockwise.SZX1024, time.Second*4),
		options.WithErrors(func(err error) {
			log.Printf("error: %v\n", err)
		}),
		options.WithKeepAlive(2, time.Second*10, func(c *client.Conn) {
			log.Printf("keepalive timeout\n")
			if errC := c.Close(); errC != nil {
				log.Printf("cannot close connection: %v\n", errC)
			}
		}))
	if err != nil {
		return fmt.Errorf("cannot dial to %v: %w", addr.String(), err)
	}
	c.client = conn
	return nil
}

func patchDeviceLink(links schema.ResourceLinks) schema.ResourceLinks {
	for idx, link := range links {
		if !link.HasType(device.ResourceType) {
			continue
		}
		newLink := link
		newLink.Href = device.ResourceURI
		links[idx] = newLink
		break
	}
	return links
}

func (c *Manager) run() {
	ctx := context.Background()
	t := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-c.done:
			return
		case wantToReset := <-c.trigger:
			if wantToReset {
				c.resetCredentials(ctx, true)
			}
		case <-t.C:
		}
		if c.getCloudConfiguration().URL != "" {
			if err := c.connect(ctx); err != nil {
				log.Printf("cannot connect to cloud: %v\n", err)
			} else {
				c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERED)
			}
		}
	}
}

func (c *Manager) connect(ctx context.Context) error {
	var funcs []func(ctx context.Context) error
	if c.isCredsExpiring() {
		funcs = append(funcs, c.refreshToken)
	}
	funcs = append(funcs, []func(ctx context.Context) error{
		c.signUp,
		c.signIn,
		c.publishResources,
	}...)
	err := c.dial(ctx)
	if err != nil {
		return err
	}
	for _, f := range funcs {
		r := func(ctx context.Context) error {
			fctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()
			return f(fctx)
		}
		err := r(ctx)
		if err != nil {
			_ = c.close()
			return err
		}
	}
	return nil
}

func (c *Manager) Close() {
	c.done <- struct{}{}
}

func (c *Manager) Unregister() {
	c.setCloudConfiguration(cloud.ConfigurationUpdateRequest{})
	c.triggerRunner(true)
}

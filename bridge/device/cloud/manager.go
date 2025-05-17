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
	"crypto/x509"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/bridge/resources/discovery"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/pkg/eventloop"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/device"
	plgdResources "github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
)

type (
	GetLinksFilteredBy func(endpoints schema.Endpoints, deviceIDfilter uuid.UUID, resourceTypesFitler []string, policyBitMaskFitler schema.BitMask) (links schema.ResourceLinks)
	GetCertificates    func(deviceID string) []tls.Certificate
	RemoveCloudCAs     func(cloudID ...string)
)

type Config struct {
	AccessToken           string
	UserID                string
	RefreshToken          string
	ValidUntil            time.Time
	AuthorizationProvider string
	CloudID               string
	CloudURL              string
}

type CAPoolGetter = interface {
	IsValid() bool
	GetPool() (*x509.CertPool, error)
}

type Manager struct {
	handler         net.RequestHandler
	getLinks        GetLinksFilteredBy
	maxMessageSize  uint32
	deviceID        uuid.UUID
	save            func()
	caPool          CAPoolGetter
	getCertificates GetCertificates
	removeCloudCAs  RemoveCloudCAs
	tickInterval    time.Duration

	private struct {
		mutex                     sync.Mutex
		cfg                       Configuration
		previousCloudIDs          []string
		readyToPublishResources   map[string]struct{}
		readyToUnpublishResources map[string]struct{}
		creds                     ocfCloud.CoapSignUpResponse
		client                    *client.Conn
		signedIn                  bool
	}

	logger             log.Logger
	resourcesPublished bool
	forceRefreshToken  bool
	done               chan struct{}
	stopped            atomic.Bool
	reconnect          atomic.Bool
	trigger            chan bool
	loop               *eventloop.Loop
}

func New(cfg Config, deviceID uuid.UUID, save func(), handler net.RequestHandler, getLinks GetLinksFilteredBy, caPool CAPoolGetter, loop *eventloop.Loop, opts ...Option) (*Manager, error) {
	if !caPool.IsValid() {
		return nil, errors.New("invalid ca pool")
	}
	o := OptionsCfg{
		maxMessageSize: net.DefaultMaxMessageSize,
		getCertificates: func(string) []tls.Certificate {
			return nil
		},
		removeCloudCAs: func(...string) {
			// do nothing
		},
		logger:       log.NewNilLogger(),
		tickInterval: time.Second * 10,
	}
	for _, opt := range opts {
		opt(&o)
	}

	c := &Manager{
		done:            make(chan struct{}),
		trigger:         make(chan bool, 10),
		handler:         handler,
		getLinks:        getLinks,
		deviceID:        deviceID,
		maxMessageSize:  o.maxMessageSize,
		save:            save,
		caPool:          caPool,
		getCertificates: o.getCertificates,
		removeCloudCAs:  o.removeCloudCAs,
		logger:          o.logger,
		loop:            loop,
		tickInterval:    o.tickInterval,
	}
	c.private.cfg.ProvisioningStatus = cloud.ProvisioningStatus_UNINITIALIZED
	c.importConfig(cfg)
	return c, nil
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

func (c *Manager) importConfig(cfg Config) {
	c.setCloudConfiguration(cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: cfg.AuthorizationProvider,
		URL:                   cfg.CloudURL,
		CloudID:               cfg.CloudID,
	})
	c.setCreds(ocfCloud.CoapSignUpResponse{
		AccessToken:  cfg.AccessToken,
		UserID:       cfg.UserID,
		RefreshToken: cfg.RefreshToken,
		ValidUntil:   cfg.ValidUntil,
	})
}

func (c *Manager) isInitialized() bool {
	cfg := c.getCloudConfiguration()
	return cfg.URL != ""
}

func (c *Manager) isSignedIn() bool {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	return c.private.signedIn
}

func (c *Manager) handleTrigger(value reflect.Value, closed bool) {
	if closed {
		return
	}
	ctx := context.Background()
	wantToReset := value.Bool()
	if wantToReset {
		c.resetCredentials(ctx, true)
	}
	if c.reconnect.CompareAndSwap(true, false) {
		err := c.close()
		if err != nil && !errors.Is(err, context.Canceled) {
			c.logger.Errorf("cannot close connection for reconnect: %w", err)
		}
		return
	}
	if !c.isInitialized() {
		// resources will be published after sign in
		c.resetPublishing()
		return
	}
	if !c.isSignedIn() {
		// resources will be published after sign in
		c.resetPublishing()
	}
	if err := c.connect(ctx); err != nil {
		c.logger.Errorf("cannot connect to cloud: %w", err)
	} else {
		c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERED)
	}
}

func (c *Manager) handleTimer(_ reflect.Value, closed bool) {
	if closed {
		return
	}
	if c.getCloudConfiguration().URL == "" {
		return
	}
	if err := c.connect(context.Background()); err != nil {
		c.logger.Errorf("cannot connect to cloud: %w", err)
	} else {
		c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERED)
	}
}

func (c *Manager) Init() {
	if c.private.cfg.URL != "" {
		c.triggerRunner(false)
	}
	t := time.NewTicker(c.tickInterval)
	handlers := []eventloop.Handler{
		eventloop.NewReadHandler(reflect.ValueOf(c.trigger), c.handleTrigger),
		eventloop.NewReadHandler(reflect.ValueOf(t.C), c.handleTimer),
		eventloop.NewReadHandler(reflect.ValueOf(c.done), func(_ reflect.Value, _ bool) {
			_ = c.close()
			// cleanup resources
			c.loop.RemoveByChannels(reflect.ValueOf(c.done), reflect.ValueOf(t.C), reflect.ValueOf(c.trigger))
			t.Stop()
		}),
	}
	c.loop.Add(handlers...)
}

func (c *Manager) resetCredentials(ctx context.Context, signOff bool) {
	if signOff {
		resetCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		if err := c.signOff(resetCtx); err != nil {
			c.logger.Debugf("%w", err)
		}
	}
	c.setCreds(ocfCloud.CoapSignUpResponse{})
	c.resourcesPublished = false
	c.forceRefreshToken = false
	c.reconnect.Store(false)
	if err := c.close(); err != nil {
		c.logger.Warnf("cannot close connection: %w", err)
	}
	c.save()
	c.removePreviousCloudIDs()
	c.logger.Infof("reset credentials")
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
		return errors.New("cloud ID cannot be empty")
	}
	if cfg.AuthorizationProvider == "" {
		return errors.New("authorization provider cannot be empty")
	}
	if cfg.URL == "" {
		return errors.New("URL cannot be empty")
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

func (c *Manager) popPreviousCloudIDs() []string {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	previousCloudIDs := c.private.previousCloudIDs
	c.private.previousCloudIDs = nil
	return previousCloudIDs
}

func (c *Manager) removePreviousCloudIDs() {
	c.removeCloudCAs(c.popPreviousCloudIDs()...)
}

func (c *Manager) setCloudConfiguration(cfg cloud.ConfigurationUpdateRequest) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.previousCloudIDs = append(c.private.previousCloudIDs, c.private.cfg.CloudID)
	c.private.cfg.AuthorizationProvider = cfg.AuthorizationProvider
	c.private.cfg.CloudID = cfg.CloudID
	c.private.cfg.URL = cfg.URL
	c.private.cfg.AuthorizationCode = cfg.AuthorizationCode
	if cfg.URL == "" {
		c.private.cfg.ProvisioningStatus = cloud.ProvisioningStatus_UNINITIALIZED
		c.private.readyToPublishResources = nil
		c.private.readyToUnpublishResources = nil
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

func (c *Manager) setCreds(creds ocfCloud.CoapSignUpResponse) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.creds = creds
	c.private.signedIn = false
}

func (c *Manager) getCreds() ocfCloud.CoapSignUpResponse {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	return c.private.creds
}

func (c *Manager) isCredsExpiring() bool {
	creds := c.getCreds()
	if creds.ValidUntil.IsZero() {
		return false
	}
	diff := time.Until(creds.ValidUntil)
	if diff < c.tickInterval*2 {
		// refresh token before it expires
		return true
	}
	// refresh token when it is 1/3 before it expires
	return time.Now().After(creds.ValidUntil.Add(-diff / 3))
}

func getResourceTypesFilter(messageOptions message.Options) []string {
	queries, _ := messageOptions.Queries()
	resourceTypesFitler := []string{}
	for _, q := range queries {
		if len(q) > 3 && q[:3] == "rt=" {
			resourceTypesFitler = append(resourceTypesFitler, q[3:])
		}
	}
	return resourceTypesFitler
}

func inFilterSupportedCodes(request *mux.Message) bool {
	switch request.Code() {
	case codes.POST, codes.PUT, codes.DELETE, codes.GET:
		return true
	default:
		return false
	}
}

func (c *Manager) handleDeviceResource(r *net.Request) (*pool.Message, error) {
	links := c.getLinks(schema.Endpoints{}, c.deviceID, nil, resources.PublishToCloud)
	for _, link := range links {
		if link.HasType(device.ResourceType) {
			_ = r.SetPath(link.Href)
			break
		}
	}
	return c.handler(r)
}

func (c *Manager) handleDiscoveryResource(r *net.Request) (*pool.Message, error) {
	links := c.getLinks(schema.Endpoints{}, c.deviceID, getResourceTypesFilter(r.Options()), resources.PublishToCloud)
	links = patchDeviceLink(links)
	links = discovery.PatchLinks(links, c.deviceID.String())
	return resources.CreateResponseContent(r.Context(), links, codes.Content)
}

func (c *Manager) getHandler(r *net.Request) func(r *net.Request) (*pool.Message, error) {
	h := c.handler
	p, err := r.Path()
	if err == nil {
		switch p {
		case device.ResourceURI:
			h = c.handleDeviceResource
		case plgdResources.ResourceURI:
			h = c.handleDiscoveryResource
		}
	}
	return h
}

func (c *Manager) serveCOAP(w mux.ResponseWriter, request *mux.Message) {
	if !inFilterSupportedCodes(request) {
		// ignore unsupported request
		if w.Conn().Context().Err() == nil {
			// log only if connection is still open
			c.logger.Debugf("unsupported request: %v\n", request)
		}
		return
	}
	request.AddQuery("di=" + c.deviceID.String())
	r := net.Request{
		Message:   request.Message,
		Endpoints: nil,
		Conn:      w.Conn(),
	}
	h := c.getHandler(&r)
	resp, err := h(&r)
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

func (c *Manager) replaceClient(client *client.Conn) *client.Conn {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.signedIn = false
	oldClient := c.private.client
	c.private.client = client
	return oldClient
}

func (c *Manager) close() error {
	oldClient := c.replaceClient(nil)
	if oldClient == nil {
		return nil
	}
	return oldClient.Close()
}

func (c *Manager) getClient() *client.Conn {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	return c.private.client
}

func (c *Manager) dial(ctx context.Context) error {
	cc := c.getClient()
	if cc != nil && cc.Context().Err() == nil {
		return nil
	}
	_ = c.close()
	cfg := c.getCloudConfiguration()

	caPool, err := c.caPool.GetPool()
	if err != nil {
		return fmt.Errorf("cannot get ca pool: %w", err)
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec
		MinVersion:         tls.VersionTLS12,
		Certificates:       c.getCertificates(c.deviceID.String()),
		VerifyPeerCertificate: coap.NewVerifyPeerCertificate(caPool, func(cert *x509.Certificate) error {
			cloudID, errP := uuid.Parse(c.getCloudConfiguration().CloudID)
			if errP != nil {
				return fmt.Errorf("cannot parse cloudID: %w", errP)
			}
			return coap.VerifyCloudCertificate(cert, cloudID)
		}),
	}

	ep := schema.Endpoint{
		URI: cfg.URL,
	}
	addr, err := ep.GetAddr()
	if err != nil {
		return fmt.Errorf("cannot get address from %v: %w", ep, err)
	}
	m := mux.NewRouter()
	m.Use(net.CreateLoggingMiddleware(c.logger))
	m.DefaultHandle(mux.HandlerFunc(c.serveCOAP))
	conn, err := tcp.Dial(addr.String(),
		options.WithTLS(tlsConfig),
		options.WithMux(m),
		options.WithContext(ctx),
		options.WithMaxMessageSize(c.maxMessageSize),
		options.WithBlockwise(false, blockwise.SZX1024, time.Second*4),
		options.WithErrors(func(err error) {
			c.logger.Errorf("cloud connection error: %w", err)
		}),
		options.WithKeepAlive(2, time.Second*10, func(conn *client.Conn) {
			c.logger.Infof("cloud connection: keepalive timeout")
			if errC := conn.Close(); errC != nil {
				c.logger.Warnf("cannot close cloud connection: %w", errC)
			}
		}))
	if err != nil {
		return fmt.Errorf("cannot dial to %v: %w", addr.String(), err)
	}
	conn.AddOnClose(func() {
		c.private.mutex.Lock()
		defer c.private.mutex.Unlock()
		if c.private.client == conn {
			c.logger.Infof("cloud connection: closed")
			c.private.client = nil
			c.private.signedIn = false
		}
	})
	c.replaceClient(conn)
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

func (c *Manager) connect(ctx context.Context) error {
	funcs := make([]func(ctx context.Context) error, 0, 5)
	if c.isCredsExpiring() || c.forceRefreshToken {
		funcs = append(funcs, c.refreshToken)
		c.forceRefreshToken = false
	}
	funcs = append(funcs, []func(ctx context.Context) error{
		c.signUp,
		c.signIn,
		c.publishResources,
		c.unpublishResources,
	}...)
	err := c.dial(ctx)
	if err != nil {
		return err
	}
	for _, f := range funcs {
		r := func(ctx context.Context) error {
			fctx, cancel := context.WithTimeout(ctx, c.tickInterval)
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
	if !c.stopped.CompareAndSwap(false, true) {
		return
	}
	close(c.done)
}

func (c *Manager) Unregister() {
	c.setCloudConfiguration(cloud.ConfigurationUpdateRequest{})
	c.triggerRunner(true)
}

func (c *Manager) resetPublishing() {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.readyToPublishResources = nil
	c.private.readyToUnpublishResources = nil
}

func (c *Manager) PublishResources(hrefs ...string) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()

	if c.private.readyToPublishResources == nil {
		c.private.readyToPublishResources = make(map[string]struct{})
	}
	for _, href := range hrefs {
		c.private.readyToPublishResources[href] = struct{}{}
		delete(c.private.readyToUnpublishResources, href)
	}
	c.triggerRunner(false)
}

func (c *Manager) UnpublishResources(hrefs ...string) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	if c.private.readyToUnpublishResources == nil {
		c.private.readyToUnpublishResources = make(map[string]struct{})
	}
	for _, href := range hrefs {
		c.private.readyToUnpublishResources[href] = struct{}{}
		delete(c.private.readyToPublishResources, href)
	}
	c.triggerRunner(false)
}

func (c *Manager) popReadyToPublishResources() map[string]struct{} {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	res := c.private.readyToPublishResources
	c.private.readyToPublishResources = nil
	return res
}

func (c *Manager) Reconnect() {
	c.reconnect.Store(true)
	c.triggerRunner(false)
}

func (c *Manager) popReadyToUnpublishResources(count int) []string {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	toUnpublish := make([]string, 0, count)
	for href := range c.private.readyToUnpublishResources {
		if count == 0 {
			break
		}
		count--
		toUnpublish = append(toUnpublish, href)
		delete(c.private.readyToUnpublishResources, href)
	}
	if len(c.private.readyToUnpublishResources) == 0 {
		c.private.readyToUnpublishResources = nil
	}
	return toUnpublish
}

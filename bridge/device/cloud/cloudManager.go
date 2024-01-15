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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	goSync "sync"
	"time"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
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
	"gopkg.in/yaml.v3"
)

type GetLinksFilteredBy func(endpoints schema.Endpoints, deviceIDfilter string, resourceTypesFitler []string, policyBitMaskFitler schema.BitMask) (links schema.ResourceLinks)

type Manager struct {
	handler        net.RequestHandler
	getLinks       GetLinksFilteredBy
	maxMessageSize uint32
	deviceID       string
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

func New(deviceID string, save func(), handler net.RequestHandler, getLinks GetLinksFilteredBy, maxMessageSize uint32) *Manager {
	c := &Manager{
		done:           make(chan struct{}),
		trigger:        make(chan bool, 10),
		handler:        handler,
		getLinks:       getLinks,
		maxMessageSize: maxMessageSize,
		deviceID:       resources.ToUUID(deviceID).String(),
		save:           save,
	}
	c.private.cfg.ProvisioningStatus = cloud.ProvisioningStatus_UNINITIALIZED
	return c
}

func (c *Manager) Get(request *net.Request) (*pool.Message, error) {
	cfg := c.getCloudConfiguration()
	return resources.CreateResponseContent(request.Context(), cfg, codes.Content)
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
			log.Printf("cannot sign off: %v\n", err)
		}
	}
	c.creds = CoapSignUpResponse{}
	c.signedIn = false
	c.resourcesPublished = false
	c.close()
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
	if cfg.CloudID == "" {
		return fmt.Errorf("cloud ID cannot be empty")
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

type ConfigurationYaml struct {
	Configuration      `yaml:",inline"`
	CoapSignUpResponse `yaml:"credentials"`
}

func (c *Manager) MarshalYAML() (interface{}, error) {
	var cfg ConfigurationYaml
	cfg.Configuration = c.getCloudConfiguration()
	cfg.CoapSignUpResponse = c.getCreds()
	return cfg, nil
}

func (c *Manager) UnmarshalYAML(value *yaml.Node) error {
	var cfg ConfigurationYaml
	err := value.Decode(&cfg)
	if err != nil {
		return err
	}
	c.setCloudConfiguration(cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: cfg.AuthorizationProvider,
		URL:                   cfg.URL,
		CloudID:               cfg.CloudID,
	})
	c.setCreds(cfg.CoapSignUpResponse)
	return nil
}

func (c *Manager) setCreds(creds CoapSignUpResponse) {
	c.creds = creds
	if creds.ExpiresIn != 0 {
		c.creds.ValidUntil = ValidUntil{
			Time: time.Now().Add(time.Duration(creds.ExpiresIn) * time.Second),
		}
	}
	c.signedIn = false
}

func (c *Manager) updateCredsBySignInResponse(resp CoapSignInResponse) {
	c.creds.ExpiresIn = resp.ExpiresIn
	if resp.ExpiresIn != 0 {
		c.creds.ValidUntil = ValidUntil{time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)}
	}
	c.signedIn = true
}

func (c *Manager) updateCredsByRefreshTokenResponse(resp CoapRefreshTokenResponse) {
	c.creds.AccessToken = resp.AccessToken
	c.creds.RefreshToken = resp.RefreshToken
	c.creds.ExpiresIn = resp.ExpiresIn
	if resp.ExpiresIn != 0 {
		c.creds.ValidUntil = ValidUntil{time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)}
	}
	c.signedIn = false
}

func (c *Manager) getCreds() CoapSignUpResponse {
	return c.creds
}

func (c *Manager) serveCOAP(w mux.ResponseWriter, request *mux.Message) {
	request.Message.AddQuery("di=" + c.deviceID)
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
	err := client.Close()
	return err
}

func (c *Manager) dial(ctx context.Context) error {
	if c.client != nil && c.client.Context().Err() == nil {
		return nil
	}
	c.close()
	cfg := c.getCloudConfiguration()
	tlsConfig := &tls.Config{
		// TODO: set RootCAs from configuration
		InsecureSkipVerify: true,
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
			c.Close()
		}))
	if err != nil {
		return fmt.Errorf("cannot dial to %v: %w", addr.String(), err)
	}
	c.client = conn
	return nil
}

func (c *Manager) newSignUpReq(ctx context.Context) (*pool.Message, error) {
	cfg := c.getCloudConfiguration()
	if cfg.AuthorizationCode == "" {
		return nil, fmt.Errorf("cannot sign up: no authorization code")
	}
	if cfg.AuthorizationProvider == "" {
		return nil, fmt.Errorf("cannot sign up: no authorization provider")
	}

	signUpRequest := CoapSignUpRequest{
		DeviceID:              c.deviceID,
		AuthorizationCode:     cfg.AuthorizationCode,
		AuthorizationProvider: cfg.AuthorizationProvider,
	}
	inputCbor, err := cbor.Encode(signUpRequest)
	if err != nil {
		return nil, err
	}
	req := c.client.AcquireMessage(ctx)
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(SignUp)
	if err != nil {
		return nil, err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))
	return req, nil
}

func (c *Manager) signUp(ctx context.Context) error {
	creds := c.getCreds()
	if creds.AccessToken != "" {
		return nil
	}
	req, err := c.newSignUpReq(ctx)
	if err != nil {
		return fmt.Errorf("cannot sign up: %w", err)
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot sign up: %w", err)
	}
	if resp.Code() != codes.Changed {
		return fmt.Errorf("cannot sign up: unexpected status code %v", resp.Code())
	}
	var signUpResp CoapSignUpResponse
	err = cbor.ReadFrom(resp.Body(), &signUpResp)
	if err != nil {
		return fmt.Errorf("cannot sign up: %w", err)
	}
	c.setCreds(signUpResp)
	log.Printf("signed up\n")
	c.save()
	return nil
}

func (c *Manager) newSignOffReq(ctx context.Context) (*pool.Message, error) {
	req := c.client.AcquireMessage(ctx)
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req.SetCode(codes.DELETE)
	req.SetToken(token)
	req.AddQuery("di=" + c.deviceID)
	req.AddQuery("uid=" + c.getCreds().UserID)
	err = req.SetPath(SignUp)
	if err != nil {
		return nil, err
	}
	return req, nil
}

const ProvisioningStatusDEREGISTERING cloud.ProvisioningStatus = "deregistering"

func (c *Manager) signOff(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	// signIn / refresh token fails
	if ctx.Err() != nil {
		return nil
	}
	req, err := c.newSignOffReq(ctx)
	if err != nil {
		return err
	}
	c.setProvisioningStatus(ProvisioningStatusDEREGISTERING)
	resp, err := c.client.Do(req)
	defer c.setProvisioningStatus(cloud.ProvisioningStatus_UNINITIALIZED)
	if err != nil {
		return err
	}
	if resp.Code() != codes.Deleted {
		return fmt.Errorf("unexpected status code %v", resp.Code())
	}
	log.Printf("signed off\n")
	return nil
}

func (c *Manager) newRefreshTokenReq(ctx context.Context, creds CoapSignUpResponse) (*pool.Message, error) {
	signInReq := CoapRefreshTokenRequest{
		DeviceID:     c.deviceID,
		UserID:       creds.UserID,
		RefreshToken: creds.RefreshToken,
	}
	inputCbor, err := cbor.Encode(signInReq)
	if err != nil {
		return nil, err
	}
	req := c.client.AcquireMessage(ctx)
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(RefreshToken)
	if err != nil {
		return nil, err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))
	return req, nil
}

func (c *Manager) refreshToken(ctx context.Context) error {
	creds := c.getCreds()
	if creds.RefreshToken == "" {
		return nil
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	if time.Now().Before(creds.ValidUntil.Add(-time.Minute * 5)) {
		return nil
	}

	req, err := c.newRefreshTokenReq(ctx, creds)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.Code() != codes.Changed {
		if resp.Code() == codes.Unauthorized {
			c.cleanup()
		}
		return fmt.Errorf("unexpected status code %v", resp.Code())
	}
	var refreshResp CoapRefreshTokenResponse
	err = cbor.ReadFrom(resp.Body(), &refreshResp)
	if err != nil {
		return err
	}
	c.updateCredsByRefreshTokenResponse(refreshResp)
	log.Printf("refresh token\n")
	c.save()
	return nil
}

func (c *Manager) newSignInReq(ctx context.Context) (*pool.Message, error) {
	creds := c.getCreds()
	if creds.AccessToken == "" {
		return nil, fmt.Errorf("cannot sign in: no access token")
	}
	signInReq := CoapSignInRequest{
		DeviceID:    c.deviceID,
		UserID:      creds.UserID,
		AccessToken: creds.AccessToken,
		Login:       true,
	}
	inputCbor, err := cbor.Encode(signInReq)
	if err != nil {
		return nil, err
	}
	req := c.client.AcquireMessage(ctx)
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(SignIn)
	if err != nil {
		return nil, err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))
	return req, nil
}

func (c *Manager) signIn(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("cannot sign in: no connection")
	}
	if c.signedIn {
		return nil
	}
	req, err := c.newSignInReq(ctx)
	if err != nil {
		return err
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.Code() != codes.Changed {
		if resp.Code() == codes.Unauthorized {
			c.cleanup()
		}
		return fmt.Errorf("unexpected status code %v", resp.Code())
	}
	var signInResp CoapSignInResponse
	err = cbor.ReadFrom(resp.Body(), &signInResp)
	if err != nil {
		return err
	}
	c.updateCredsBySignInResponse(signInResp)
	log.Printf("signed in\n")
	c.save()
	return nil
}

func patchDeviceLink(links schema.ResourceLinks) schema.ResourceLinks {
	for idx, link := range links {
		if link.HasType(device.ResourceType) {
			newLink := link
			newLink.Href = device.ResourceURI
			newLink.Anchor = "ocf://" + device.ResourceURI
			links[idx] = newLink
			break
		}
	}
	return links
}

func (c *Manager) newPublishResourcesReq(ctx context.Context) (*pool.Message, error) {
	links := c.getLinks(schema.Endpoints{}, c.deviceID, nil, resources.PublishToCloud)
	links = patchDeviceLink(links)
	wkRd := PublishResourcesRequest{
		DeviceID:   c.deviceID,
		Links:      links,
		TimeToLive: 0,
	}
	inputCbor, err := cbor.Encode(wkRd)
	if err != nil {
		return nil, err
	}
	req := c.client.AcquireMessage(ctx)
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req.SetCode(codes.POST)
	req.SetToken(token)
	err = req.SetPath(ResourceDirectory)
	if err != nil {
		return nil, err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))
	return req, nil
}

func (c *Manager) publishResources(ctx context.Context) error {
	if c.resourcesPublished {
		return nil
	}
	req, err := c.newPublishResourcesReq(ctx)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.Code() != codes.Changed {
		return fmt.Errorf("unexpected status code %v", resp.Code())
	}
	c.resourcesPublished = true
	log.Printf("resourcesPublished\n")
	return nil
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
			err := c.connect(ctx)
			if err != nil {
				log.Printf("cannot connect to cloud: %v\n", err)
			} else {
				c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERED)
			}
		}
	}
}

func (c *Manager) connect(ctx context.Context) error {
	funcs := []func(ctx context.Context) error{
		c.signUp,
		c.refreshToken,
		c.signIn,
		c.publishResources,
	}
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
			c.close()
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

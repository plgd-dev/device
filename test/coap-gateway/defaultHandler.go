/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
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

package test

import (
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/device/v2/pkg/ocf/cloud"
)

// DefaultHandler is the default handler for tests
//
// It implements ServiceHandler interface by just logging the called method and
// returning default response and no error (if required).
type DefaultHandler struct {
	deviceID            string
	accessToken         string
	refreshToken        string
	accessTokenLifetime int64 // lifetime in seconds or <0 value for a token without expiration
}

func MakeDefaultHandler(accessTokenLifetime int64) DefaultHandler {
	return DefaultHandler{
		accessToken:         "access-token",
		refreshToken:        "refresh-token",
		accessTokenLifetime: accessTokenLifetime,
	}
}

func (h *DefaultHandler) GetDeviceID() string {
	return h.deviceID
}

func (h *DefaultHandler) SetDeviceID(deviceID string) {
	h.deviceID = deviceID
}

func (h *DefaultHandler) SetAccessToken(accessToken string) {
	h.accessToken = accessToken
}

func (h *DefaultHandler) SetRefreshToken(refreshToken string) {
	h.refreshToken = refreshToken
}

func (h *DefaultHandler) SignUp(req cloud.CoapSignUpRequest) (cloud.CoapSignUpResponse, error) {
	fmt.Printf("SignUp: %v\n", req)
	h.SetDeviceID(req.DeviceID)
	return cloud.CoapSignUpResponse{
		AccessToken:  h.accessToken,
		UserID:       "1",
		RefreshToken: h.refreshToken,
		ExpiresIn:    h.accessTokenLifetime,
		RedirectURI:  "",
	}, nil
}

func (h *DefaultHandler) CloseOnError() bool {
	return true
}

func (h *DefaultHandler) SignOff() error {
	fmt.Printf("SignOff deviceID:%v\n", h.deviceID)
	return nil
}

func (h *DefaultHandler) SignIn(req cloud.CoapSignInRequest) (cloud.CoapSignInResponse, error) {
	fmt.Printf("SignIn: %v\n", req)
	return cloud.CoapSignInResponse{
		ExpiresIn: h.accessTokenLifetime,
	}, nil
}

func (h *DefaultHandler) SignOut(req cloud.CoapSignInRequest) error {
	fmt.Printf("SignOut: %v\n", req)
	return nil
}

func (h *DefaultHandler) PublishResources(req cloud.PublishResourcesRequest) error {
	fmt.Printf("PublishResources: %v\n", req)
	return nil
}

func (h *DefaultHandler) UnpublishResources(req cloud.UnpublishResourcesRequest) error {
	fmt.Printf("UnpublishResources: %v\n", req)
	return nil
}

func (h *DefaultHandler) RefreshToken(req cloud.CoapRefreshTokenRequest) (cloud.CoapRefreshTokenResponse, error) {
	fmt.Printf("RefreshToken: %v\n", req)
	return cloud.CoapRefreshTokenResponse{
		RefreshToken: h.refreshToken,
		AccessToken:  h.accessToken,
		ExpiresIn:    h.accessTokenLifetime,
	}, nil
}

const (
	SignUpKey       = "SignUp"  // register
	SignOffKey      = "SignOff" // deregister
	SignInKey       = "SignIn"  // log in
	SignOutKey      = "SignOut" // log out
	PublishKey      = "Publish"
	UnpublishKey    = "Unpublish"
	RefreshTokenKey = "RefreshToken"
)

type DefaultHandlerWithCounter struct {
	*DefaultHandler

	CallCounter struct {
		Data map[string]int
		Lock sync.Mutex
	}

	signedInChan  chan int
	signedOffChan chan int
	publishChan   chan int
	unpublishChan chan int
}

func NewCoapHandlerWithCounter(atLifetime int64) *DefaultHandlerWithCounter {
	dh := MakeDefaultHandler(atLifetime)
	return &DefaultHandlerWithCounter{
		DefaultHandler: &dh,
		CallCounter: struct {
			Data map[string]int
			Lock sync.Mutex
		}{
			Data: make(map[string]int),
		},

		signedInChan:  make(chan int, 16),
		signedOffChan: make(chan int, 16),
		publishChan:   make(chan int, 16),
		unpublishChan: make(chan int, 16),
	}
}

func sendToChan(c chan int, v int) {
	select {
	case c <- v:
	default:
	}
}

func waitForAction(c chan int, timeout time.Duration) int {
	select {
	case v := <-c:
		return v
	case <-time.After(timeout):
		return -1
	}
}

func (ch *DefaultHandlerWithCounter) SignUp(req cloud.CoapSignUpRequest) (cloud.CoapSignUpResponse, error) {
	resp, err := ch.DefaultHandler.SignUp(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignUpKey]++
	ch.CallCounter.Lock.Unlock()
	return resp, err
}

func (ch *DefaultHandlerWithCounter) SignOff() error {
	err := ch.DefaultHandler.SignOff()
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignOffKey]++
	signOffCount, ok := ch.CallCounter.Data[SignOffKey]
	if ok {
		sendToChan(ch.signedOffChan, signOffCount)
	}
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *DefaultHandlerWithCounter) WaitForSignOff(timeout time.Duration) int {
	return waitForAction(ch.signedOffChan, timeout)
}

func (ch *DefaultHandlerWithCounter) SignIn(req cloud.CoapSignInRequest) (cloud.CoapSignInResponse, error) {
	resp, err := ch.DefaultHandler.SignIn(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignInKey]++
	signInCount, ok := ch.CallCounter.Data[SignInKey]
	if ok {
		sendToChan(ch.signedInChan, signInCount)
	}
	ch.CallCounter.Lock.Unlock()
	return resp, err
}

func (ch *DefaultHandlerWithCounter) WaitForSignIn(timeout time.Duration) int {
	return waitForAction(ch.signedInChan, timeout)
}

func (ch *DefaultHandlerWithCounter) SignOut(req cloud.CoapSignInRequest) error {
	err := ch.DefaultHandler.SignOut(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[SignOutKey]++
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *DefaultHandlerWithCounter) WaitForPublish(timeout time.Duration) int {
	return waitForAction(ch.publishChan, timeout)
}

func (ch *DefaultHandlerWithCounter) PublishResources(req cloud.PublishResourcesRequest) error {
	err := ch.DefaultHandler.PublishResources(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[PublishKey]++
	count, ok := ch.CallCounter.Data[PublishKey]
	if ok {
		sendToChan(ch.publishChan, count)
	}
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *DefaultHandlerWithCounter) WaitForUnpublish(timeout time.Duration) int {
	return waitForAction(ch.unpublishChan, timeout)
}

func (ch *DefaultHandlerWithCounter) UnpublishResources(req cloud.UnpublishResourcesRequest) error {
	err := ch.DefaultHandler.UnpublishResources(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[UnpublishKey]++
	count, ok := ch.CallCounter.Data[UnpublishKey]
	if ok {
		sendToChan(ch.unpublishChan, count)
	}
	ch.CallCounter.Lock.Unlock()
	return err
}

func (ch *DefaultHandlerWithCounter) RefreshToken(req cloud.CoapRefreshTokenRequest) (cloud.CoapRefreshTokenResponse, error) {
	resp, err := ch.DefaultHandler.RefreshToken(req)
	ch.CallCounter.Lock.Lock()
	ch.CallCounter.Data[RefreshTokenKey]++
	ch.CallCounter.Lock.Unlock()
	return resp, err
}

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

package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
)

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpPostHandler(r *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.sendErrorResponse(fmt.Errorf("cannot handle sign up: %w", err), code, r.Token())
		if client.handler == nil || client.handler.CloseOnError() {
			// to send the error response
			time.Sleep(time.Millisecond * 100)
			if err := client.Close(); err != nil {
				fmt.Printf("sign up error: %v\n", err)
			}
		}
	}

	var signUp cloud.CoapSignUpRequest
	if err := cbor.ReadFrom(r.Body(), &signUp); err != nil {
		logErrorAndCloseClient(err, coapCodes.BadRequest)
		return
	}

	client.SetDeviceID(signUp.DeviceID)

	resp, err := client.handler.SignUp(signUp)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	out, err := cbor.Encode(resp)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Changed, r.Token(), out)
}

// Sign-off
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signOffHandler(req *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.sendErrorResponse(fmt.Errorf("cannot handle sign off: %w", err), code, req.Token())
		if client.handler == nil || client.handler.CloseOnError() {
			// to send the error response
			time.Sleep(time.Millisecond * 100)
			if err := client.Close(); err != nil {
				fmt.Printf("sign off error: %v\n", err)
			}
		}
	}

	err := client.handler.SignOff()
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Deleted, req.Token(), nil)
}

// Sign-up
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.account.swagger.json
func signUpHandler(r *mux.Message, client *Client) {
	switch r.Code() {
	case coapCodes.POST:
		signUpPostHandler(r, client)
	case coapCodes.DELETE:
		signOffHandler(r, client)
	default:
		client.sendErrorResponse(fmt.Errorf("forbidden request from %v", client.RemoteAddrString()), coapCodes.Forbidden, r.Token())
	}
}

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

	"github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInPostHandler(req *mux.Message, client *Client, signIn cloud.CoapSignInRequest) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.sendErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), code, req.Token())
		if client.handler == nil || client.handler.CloseOnError() {
			if err := client.Close(); err != nil {
				fmt.Printf("sign in error: %v\n", err)
			}
		}
	}

	resp, err := client.handler.SignIn(signIn)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	out, err := cbor.Encode(resp)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token(), out)
}

// Sign-Out
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signOutPostHandler(req *mux.Message, client *Client, signOut cloud.CoapSignInRequest) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.sendErrorResponse(fmt.Errorf("cannot handle sign out: %w", err), code, req.Token())
		if client.handler == nil || client.handler.CloseOnError() {
			if err := client.Close(); err != nil {
				fmt.Printf("sign out error: %v\n", err)
			}
		}
	}

	err := client.handler.SignOut(signOut)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.InternalServerError)
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token(), []byte{0xA0}) // empty object
}

// Sign-in
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.session.swagger.json
func signInHandler(req *mux.Message, client *Client) {
	if req.Code() == coapCodes.POST {
		var r cloud.CoapSignInRequest
		err := cbor.ReadFrom(req.Body(), &r)
		if err != nil {
			client.sendErrorResponse(fmt.Errorf("cannot handle sign in: %w", err), coapCodes.BadRequest, req.Token())
			return
		}
		if r.Login {
			signInPostHandler(req, client, r)
			return
		}
		signOutPostHandler(req, client, r)
		return
	}
	client.sendErrorResponse(fmt.Errorf("forbidden request from %v", client.RemoteAddrString()), coapCodes.Forbidden, req.Token())
}

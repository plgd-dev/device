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
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	grpcCodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client a setup of connection
type Client struct {
	server   *Service
	coapConn *coapTcpClient.Conn
	handler  ServiceHandler

	deviceID string
}

// newClient creates and initializes client
func newClient(server *Service, client *coapTcpClient.Conn, handler ServiceHandler) *Client {
	return &Client{
		server:   server,
		coapConn: client,
		handler:  handler,
	}
}

func (c *Client) GetCoapConnection() *coapTcpClient.Conn {
	return c.coapConn
}

func (c *Client) GetServiceHandler() ServiceHandler {
	return c.handler
}

func (c *Client) GetDeviceID() string {
	return c.deviceID
}

func (c *Client) SetDeviceID(deviceID string) {
	c.deviceID = deviceID
}

func (c *Client) RemoteAddrString() string {
	return c.coapConn.RemoteAddr().String()
}

func (c *Client) Context() context.Context {
	return c.coapConn.Context()
}

// Close closes coap connection
func (c *Client) Close() error {
	if err := c.coapConn.Close(); err != nil {
		return fmt.Errorf("cannot close client: %w", err)
	}
	return nil
}

// OnClose is invoked when the coap connection was closed.
func (c *Client) OnClose() {
	fmt.Printf("close client %v\n", c.coapConn.RemoteAddr())
}

type grpcErr interface {
	GRPCStatus() *status.Status
}

func isContextCanceled(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	var gErr grpcErr
	if ok := errors.As(err, &gErr); ok {
		return gErr.GRPCStatus().Code() == grpcCodes.Canceled
	}
	return false
}

func (c *Client) sendResponse(code codes.Code, token message.Token, payload []byte) {
	msg := pool.NewMessage(c.Context())
	msg.SetCode(code)
	msg.SetToken(token)
	if len(payload) > 0 {
		msg.SetContentFormat(message.AppOcfCbor)
		msg.SetBody(bytes.NewReader(payload))
	}
	if err := c.coapConn.WriteMessage(msg); err != nil {
		if !isContextCanceled(err) {
			fmt.Printf("cannot send reply to %v: %v\n", c.GetDeviceID(), err)
		}
	}
}

func (c *Client) sendErrorResponse(err error, code codes.Code, token message.Token) {
	msg := pool.NewMessage(c.Context())
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	if err = c.coapConn.WriteMessage(msg); err != nil {
		fmt.Printf("cannot send error to %v: %v\n", c.GetDeviceID(), err)
	}
}

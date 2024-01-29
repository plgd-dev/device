package service

import (
	"fmt"

	"github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

func refreshTokenPostHandler(req *mux.Message, client *Client) {
	logErrorAndCloseClient := func(err error, code coapCodes.Code) {
		client.sendErrorResponse(fmt.Errorf("cannot handle refresh token: %w", err), code, req.Token())
		if client.handler == nil || client.handler.CloseOnError() {
			if err := client.Close(); err != nil {
				fmt.Printf("refresh token error: %v\n", err)
			}
		}
	}

	var r cloud.CoapRefreshTokenRequest
	err := cbor.ReadFrom(req.Body(), &r)
	if err != nil {
		logErrorAndCloseClient(err, coapCodes.BadRequest)
		return
	}

	resp, err := client.handler.RefreshToken(r)
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

// RefreshToken
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.tokenrefresh.swagger.json
func refreshTokenHandler(req *mux.Message, client *Client) {
	if req.Code() == coapCodes.POST {
		refreshTokenPostHandler(req, client)
		return
	}
	client.sendErrorResponse(fmt.Errorf("forbidden request from %v", client.RemoteAddrString()), coapCodes.Forbidden, req.Token())
}

package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/mux"
)

type GetHandlerFunc func(req *Request) (*pool.Message, error)

type PostHandlerFunc func(req *Request) (*pool.Message, error)

type Resource struct {
	Href               string
	ResourceTypes      []string
	ResourceInterfaces []string
	getHandler         GetHandlerFunc
	postHandler        PostHandlerFunc
}

func (r *Resource) HasResourceTypes(resourceTypes []string) bool {
	for _, rt := range resourceTypes {
		for _, rrt := range r.ResourceTypes {
			if rt == rrt {
				return true
			}
		}
	}
	return false
}

type Request struct {
	*pool.Message
}

func (r *Request) Interface() string {
	q, err := r.Queries()
	if err != nil {
		return ""
	}
	for _, query := range q {
		if strings.HasPrefix(query, "if=") {
			return strings.TrimPrefix(query, "if=")
		}
	}
	return ""
}

func (r *Request) ResourceTypes() []string {
	q, err := r.Queries()
	if err != nil {
		return nil
	}
	resourceTypes := make([]string, 0, len(q))
	for _, query := range q {
		if strings.HasPrefix(query, "rt=") {
			resourceTypes = append(resourceTypes, strings.TrimPrefix(query, "rt="))
		}
	}
	return resourceTypes
}

func NewResource(href string, getHandler GetHandlerFunc, postHandler PostHandlerFunc, resourceTypes, resourceInterfaces []string) *Resource {
	return &Resource{
		Href:               href,
		ResourceTypes:      resourceTypes,
		ResourceInterfaces: resourceInterfaces,
		getHandler:         getHandler,
		postHandler:        postHandler,
	}
}

/*
func (c *session) createErrorResponse(err error, token message.Token) *pool.Message {
	if err == nil {
		return nil
	}
	s, ok := status.FromError(err)
	code := codes.BadRequest
	if ok {
		code = s.Code()
	}
	msg := c.server.messagePool.AcquireMessage(c.Context())
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	return msg
}
*/

func createResponseError(ctx context.Context, err error, token message.Token) *pool.Message {
	if err == nil {
		return nil
	}
	s, ok := status.FromError(err)
	code := codes.BadRequest
	if ok {
		code = s.Code()
	}
	msg := pool.NewMessage(ctx)
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	return msg
}

func createResponseMethodNotAllowed(ctx context.Context, code codes.Code, token message.Token) *pool.Message {
	msg := pool.NewMessage(ctx)
	msg.SetCode(codes.MethodNotAllowed)
	msg.SetToken(token)
	msg.SetBody(bytes.NewReader([]byte(fmt.Sprintf("unsupported method %v", code))))
	return msg
}

func (r *Resource) ServeCOAP(w mux.ResponseWriter, request *mux.Message) {
	var resp *pool.Message
	var err error
	switch request.Code() {
	case codes.GET:
		if r.getHandler == nil {
			resp = createResponseMethodNotAllowed(request.Context(), request.Code(), request.Token())
		} else {
			resp, err = r.getHandler(&Request{request.Message})
			if err != nil {
				resp = createResponseError(request.Context(), err, request.Token())
			}
		}
	case codes.POST:
		if r.postHandler == nil {
			resp = createResponseMethodNotAllowed(request.Context(), request.Code(), request.Token())
		} else {
			resp, err = r.postHandler(&Request{request.Message})
			if err != nil {
				resp = createResponseError(request.Context(), err, request.Token())
			}
		}
	default:
	}
	if resp != nil {
		resp.SetToken(w.Message().Token())
		resp.SetMessageID(w.Message().MessageID())
		resp.SetType(w.Message().Type())
		w.SetMessage(resp)
	}
}

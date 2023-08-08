// ************************************************************************
// Copyright (C) 2023 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package coap

import (
	"errors"
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type DetailedResponseSetter interface {
	SetCode(code codes.Code)
	SetETag(etag []byte)
	GetBody() interface{}
}

// DetailedResponse is a response with metadata from the response options.
type DetailedResponse[T any] struct {
	Code codes.Code // content format from the response options
	ETag []byte     // ETag from the response
	Body T          // parsed response body, pointer to the variable will be send to codec.Decode
}

func (dr *DetailedResponse[T]) SetCode(code codes.Code) {
	dr.Code = code
}

func (dr *DetailedResponse[T]) SetETag(etag []byte) {
	dr.ETag = etag
}

func (dr *DetailedResponse[T]) GetBody() interface{} {
	return &dr.Body
}

// TrySetDetailedReponse checks if the output value implements DetailedResponseSetter interface
// and sets the response options (Code and ETag) to the value.
func TrySetDetailedReponse(response *pool.Message, v interface{}) (interface{}, error) {
	if responseDetail, ok := v.(DetailedResponseSetter); ok {
		etag, err := response.ETag()
		if err == nil {
			responseDetail.SetETag(etag)
		} else if !errors.Is(err, message.ErrOptionNotFound) {
			return nil, fmt.Errorf("cannot get ETag: %w", err)
		}
		responseDetail.SetCode(response.Code())
		v = responseDetail.GetBody()
	}
	return v, nil
}

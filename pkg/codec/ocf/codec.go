// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
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

package ocf

import (
	"errors"
	"fmt"
	"io"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

var (
	ErrUnknownContentFormat = errors.New("unknown content format")
	ErrEmptyBody            = errors.New("unexpected empty body")
	ErrReadBody             = errors.New("cannot read body")
)

// VNDOCFCBORCodec encodes/decodes according to the CoAP content format/media type.
type VNDOCFCBORCodec struct{}

// ContentFormat used for encoding.
func (VNDOCFCBORCodec) ContentFormat() message.MediaType { return message.AppOcfCbor }

// Encode encodes v and returns bytes.
func (VNDOCFCBORCodec) Encode(v interface{}) ([]byte, error) {
	return cbor.Encode(v)
}

func errUnknownContentFormat(err error) error {
	return fmt.Errorf("%w: %w", ErrUnknownContentFormat, err)
}

// Decode the CBOR payload of a COAP message.
func (VNDOCFCBORCodec) Decode(m *pool.Message, v interface{}) error {
	if v == nil {
		return nil
	}
	mt, err := m.Options().ContentFormat()
	if err != nil {
		return errUnknownContentFormat(err)
	}
	if mt != message.AppCBOR && mt != message.AppOcfCbor {
		return fmt.Errorf("not a CBOR content format: %v", mt)
	}
	if m.Body() == nil {
		return ErrEmptyBody
	}
	if err := cbor.ReadFrom(m.Body(), v); err != nil {
		p, _ := m.Options().Path()
		return fmt.Errorf("decoding failed for the message %v on %v with error: %w", m.Token(), p, err)
	}
	return nil
}

// MakeRawVNDOCFCBORCodec creates a RawVNDOCFCBORCodec codec, which performs no encoding/decoding,
// but propagates/validates the CoAP content format/media type.
func MakeRawVNDOCFCBORCodec() RawCodec {
	return RawCodec{
		EncodeMediaType: message.AppOcfCbor,
		DecodeMediaTypes: []message.MediaType{
			message.AppOcfCbor,
			message.AppCBOR,
		},
	}
}

// RawCodec performs no encoding/decoding but it propagates/validates the CoAP content format/media type.
type RawCodec struct {
	EncodeMediaType  message.MediaType
	DecodeMediaTypes []message.MediaType
}

// ContentFormat propagates the CoAP media type
func (c RawCodec) ContentFormat() message.MediaType { return c.EncodeMediaType }

// DecodeContentFormat propagates the CoAP media type
func (c RawCodec) DecodeContentFormat() []message.MediaType {
	return c.DecodeMediaTypes
}

// Encode propagates the payload without any conversions.
func (c RawCodec) Encode(v interface{}) ([]byte, error) {
	p, ok := v.([]byte)
	if !ok {
		return nil, errors.New("expected []byte")
	}
	return p, nil
}

func contains(a []message.MediaType, x message.MediaType) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func errReadBody(err error) error {
	return fmt.Errorf("%w: %w", ErrReadBody, err)
}

// Decode validates the content format and
// propagates the payload to v as *[]byte without any conversions.
func (c RawCodec) Decode(m *pool.Message, v interface{}) error {
	if v == nil {
		return nil
	}
	mt, err := m.Options().ContentFormat()
	if err != nil {
		if m.Body() == nil {
			return nil
		}
		return errUnknownContentFormat(err)
	}
	if !contains(c.DecodeMediaTypes, mt) {
		return fmt.Errorf("unexpected content format: %v, supported content formats: %v", mt, c.DecodeMediaTypes)
	}
	if m.Body() == nil {
		return ErrEmptyBody
	}

	p, ok := v.(*[]byte)
	if !ok {
		return errors.New("expected *[]byte")
	}
	b, err := io.ReadAll(m.Body())
	if err != nil {
		return errReadBody(err)
	}
	*p = b
	return nil
}

// DumpPayload dumps the COAP message payload to a string.
func DumpPayload(m *pool.Message) (string, error) {
	if m.Body() == nil {
		return "nil", nil
	}
	mt, err := m.Options().ContentFormat()
	if err != nil {
		return "", errUnknownContentFormat(err)
	}
	switch mt {
	case message.TextPlain, message.AppJSON:
		b, err := io.ReadAll(m.Body())
		if err != nil {
			return "", errReadBody(err)
		}
		return string(b), nil
	case message.AppCBOR, message.AppOcfCbor:
		b, err := io.ReadAll(m.Body())
		if err != nil {
			return "", errReadBody(err)
		}
		return cbor.ToJSON(b)
	default:
		return "", fmt.Errorf("unknown content format %v", mt)
	}
}

// DumpHeader dumps the basic COAP message details to a string.
func DumpHeader(m *pool.Message) string {
	buf := ""
	path, err := m.Options().Path()
	if err != nil {
		buf = fmt.Sprintf("%sPath: %v\n", buf, path)
	}
	cf, err := m.Options().ContentFormat()
	if err == nil {
		buf = fmt.Sprintf("%sFormat: %v\n", buf, cf)
	}
	queries, err := m.Options().Queries()
	if err == nil {
		buf = fmt.Sprintf("%sQueries: %+v\n", buf, queries)
	}

	return buf
}

// Dump a COAP message to a string. If parsing fails, the error is appended.
func Dump(message *pool.Message) string {
	header := DumpHeader(message)
	payload, err := DumpPayload(message)
	if err != nil {
		payload = err.Error()
	}
	return fmt.Sprintf("%s\nContent: %s\n", header, payload)
}

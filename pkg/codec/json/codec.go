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

package json

import (
	"bytes"
	"fmt"
	"io"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/ugorji/go/codec"
)

// WriteTo writes v to writer.
func WriteTo(w io.Writer, v interface{}) error {
	var h codec.JsonHandle
	h.Canonical = true
	err := codec.NewEncoder(w, &h).Encode(v)
	if err != nil {
		return fmt.Errorf("JSON encoder failed: %w", err)
	}
	return nil
}

// Encode encodes v and returns bytes.
func Encode(v interface{}) ([]byte, error) {
	b := bytes.NewBuffer(make([]byte, 0, 128))
	err := WriteTo(b, v)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ReadFrom reads and stores the result in v.
func ReadFrom(w io.Reader, v interface{}) error {
	var h codec.JsonHandle
	err := codec.NewDecoder(w, &h).Decode(v)
	if err != nil {
		return fmt.Errorf("JSON decoder failed: %w", err)
	}
	return nil
}

// Decode decodes bytes and stores the result in v.
func Decode(b []byte, v interface{}) error {
	buf := bytes.NewBuffer(b)
	err := ReadFrom(buf, v)
	if err != nil {
		return err
	}
	return nil
}

// ToCBOR converts JSON to CBOR.
func ToCBOR(json string) ([]byte, error) {
	var m interface{}
	if err := Decode([]byte(json), &m); err != nil {
		return nil, err
	}
	return cbor.Encode(m)
}

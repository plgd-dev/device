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

package cbor

import (
	"bytes"
	"io"

	"github.com/fxamacker/cbor/v2"
	"github.com/ugorji/go/codec"
)

// Encode encodes v and returns bytes.
func Encode(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	err := WriteTo(buf, v)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteTo writes v to writer.
func WriteTo(w io.Writer, v interface{}) error {
	encOpts := cbor.EncOptions{
		Sort: cbor.SortBytewiseLexical,
	}
	encMode, err := encOpts.EncMode()
	if err != nil {
		return err
	}
	return encMode.NewEncoder(w).Encode(v)
}

// Decode decodes bytes and stores the result in v.
func Decode(b []byte, v interface{}) error {
	return cbor.Unmarshal(b, v)
}

// ReadFrom reads and stores the result in v.
func ReadFrom(w io.Reader, v interface{}) error {
	return cbor.NewDecoder(w).Decode(v)
}

// ToJSON converts CBOR to JSON.
func ToJSON(cbor []byte) (string, error) {
	var m interface{}
	if err := Decode(cbor, &m); err != nil {
		return "", err
	}
	b := bytes.NewBuffer(make([]byte, 0, 1024))
	h := codec.JsonHandle{}
	h.BasicHandle.Canonical = true
	enc := codec.NewEncoder(b, &h)
	if err := enc.Encode(m); err != nil {
		return "", err
	}
	return b.String(), nil
}

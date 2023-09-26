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
	"encoding/base64"
	"strings"
)

const (
	// maxURIQueryLen is the maximum length of a URI query. See https://datatracker.ietf.org/doc/html/rfc7252#section-5.10
	maxURIQueryLen = 255
	// maxETagLen is the maximum length of an ETag. See https://datatracker.ietf.org/doc/html/rfc7252#section-5.10
	maxETagLen = 8
	// prefixQueryIncChanged is the prefix of the URI query for the "incremental changed" option. See https://docs.plgd.dev/docs/features/control-plane/entity-tag/#etag-batch-interface-for-oicres
	prefixQueryIncChanges = "incChanges="
)

func EncodeETagsForIncrementalChanges(etags [][]byte) []string {
	if len(etags) < 1 {
		return nil
	}
	etagsStr := make([]string, 0, (len(etags)/15)+1)
	var b strings.Builder
	for _, etag := range etags {
		if len(etag) > maxETagLen {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(prefixQueryIncChanges)
		} else {
			b.WriteString(",")
		}
		b.WriteString(base64.RawURLEncoding.EncodeToString(etag))
		if b.Len() >= maxURIQueryLen-(maxETagLen*2) {
			etagsStr = append(etagsStr, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		etagsStr = append(etagsStr, b.String())
	}
	return etagsStr
}

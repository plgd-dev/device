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

package resources

import (
	"crypto/rand"
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/device/v2/internal/math"
)

var globalETag atomic.Uint64

func generateNextETag(currentETag uint64) uint64 {
	buf := make([]byte, 4)
	_, err := rand.Read(buf)
	if err != nil {
		return currentETag + math.CastTo[uint64](time.Now().UnixNano()%1000)
	}
	return currentETag + uint64(binary.BigEndian.Uint32(buf)%1000)
}

func GetETag() uint64 {
	for {
		now := math.CastTo[uint64](time.Now().UnixNano())
		oldEtag := globalETag.Load()
		etag := oldEtag
		if now > etag {
			etag = now
		}
		newEtag := generateNextETag(etag)
		if globalETag.CompareAndSwap(oldEtag, newEtag) {
			return newEtag
		}
	}
}

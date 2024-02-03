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

package resources_test

import (
	"sync"
	"testing"

	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/stretchr/testify/require"
)

func TestGetETag(t *testing.T) {
	var last uint64
	for i := 0; i < 1000; i++ {
		etag := resources.GetETag()
		require.Greater(t, etag, last)
		last = etag
	}
}

func TestGetETagParallel(t *testing.T) {
	const numRoutines = 1000

	etagMap := make(map[uint64]struct{})
	var mutex sync.Mutex

	var wg sync.WaitGroup
	wg.Add(numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func() {
			defer wg.Done()
			etag := resources.GetETag()
			mutex.Lock()
			etagMap[etag] = struct{}{}
			mutex.Unlock()
		}()
	}

	wg.Wait()

	require.Len(t, etagMap, numRoutines)
}

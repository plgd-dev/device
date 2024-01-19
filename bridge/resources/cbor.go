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

import "github.com/plgd-dev/device/v2/pkg/codec/cbor"

func MergeCBORStructs(a ...interface{}) interface{} {
	var merged map[interface{}]interface{}
	for _, v := range a {
		if v == nil {
			continue
		}
		data, err := cbor.Encode(v)
		if err != nil {
			continue
		}
		var m map[interface{}]interface{}
		err = cbor.Decode(data, &m)
		if err != nil {
			continue
		}
		if merged == nil {
			merged = m
		} else {
			for k, v := range m {
				merged[k] = v
			}
		}
	}
	return merged
}

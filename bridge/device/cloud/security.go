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

package cloud

import (
	"crypto/x509"
	"fmt"
)

type GetCAPool func() []*x509.Certificate

type CAPool struct {
	getCAPool       GetCAPool
	useSystemCAPool bool
}

func MakeCAPool(getCAPool GetCAPool, useSystemCAPool bool) CAPool {
	return CAPool{
		getCAPool:       getCAPool,
		useSystemCAPool: useSystemCAPool,
	}
}

func (c *CAPool) IsValid() bool {
	return c.useSystemCAPool || (c.getCAPool != nil && c.getCAPool() != nil)
}

func (c *CAPool) GetPool() (*x509.CertPool, error) {
	var pool *x509.CertPool
	if c.useSystemCAPool {
		systemPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("cannot get system pool: %w", err)
		}
		pool = systemPool
	} else {
		pool = x509.NewCertPool()
	}
	for _, ca := range c.getCAPool() {
		pool.AddCert(ca)
	}
	return pool, nil
}

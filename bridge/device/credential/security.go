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

package credential

import (
	"crypto/x509"
)

type GetCAPool func() []*x509.Certificate

type CAPoolGetter = interface {
	IsValid() bool
	GetPool() (*x509.CertPool, error)
}

type CAPool struct {
	origCAPool CAPoolGetter
	getCAPool  GetCAPool
}

func MakeCAPool(caPool CAPoolGetter, getCAPool GetCAPool) CAPool {
	return CAPool{
		getCAPool:  getCAPool,
		origCAPool: caPool,
	}
}

func (c CAPool) IsValid() bool {
	if c.getCAPool != nil {
		return true
	}
	if c.origCAPool == nil {
		return false
	}
	return c.origCAPool.IsValid()
}

func (c CAPool) GetPool() (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if c.origCAPool != nil && c.origCAPool.IsValid() {
		p, err := c.origCAPool.GetPool()
		if err != nil {
			return nil, err
		}
		pool = p
	}
	if c.getCAPool == nil {
		return pool, nil
	}
	for _, ca := range c.getCAPool() {
		pool.AddCert(ca)
	}
	return pool, nil
}

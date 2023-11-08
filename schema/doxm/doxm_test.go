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

package doxm_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/stretchr/testify/require"
)

func TestOwnerTransferMethodToString(t *testing.T) {
	methodToStr := map[doxm.OwnerTransferMethod]string{
		doxm.JustWorks:               "JustWorks",
		doxm.SharedPin:               "SharedPin",
		doxm.ManufacturerCertificate: "ManufacturerCertificate",
		doxm.Self:                    "Self",
	}

	for method, str := range methodToStr {
		require.Equal(t, str, method.String())
	}

	invalid := doxm.OwnerTransferMethod(42)
	require.Equal(t, "unknown 42", invalid.String())
}

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

package device_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/stretchr/testify/require"
)

func TestDeviceGetManufacturerName(t *testing.T) {
	type fields struct {
		ManufacturerName []device.LocalizedString
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Empty",
			fields: fields{
				ManufacturerName: nil,
			},
			want: "",
		},
		{
			name: "Slovak",
			fields: fields{
				ManufacturerName: []device.LocalizedString{
					{
						Language: "sk",
						Value:    "zariadenie",
					},
				},
			},
			want: "",
		},
		{
			name: "English",
			fields: fields{
				ManufacturerName: []device.LocalizedString{
					{
						Language: "en",
						Value:    "device",
					},
				},
			},
			want: "device",
		},
		{
			name: "Multiple",
			fields: fields{
				ManufacturerName: []device.LocalizedString{
					{
						Language: "sk",
						Value:    "zariadenie",
					},
					{
						Language: "en",
						Value:    "device",
					},
					{
						Language: "de",
						Value:    "Gerät",
					},
				},
			},
			want: "device",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := device.Device{
				ManufacturerName: tt.fields.ManufacturerName,
			}
			require.Equal(t, tt.want, d.GetManufacturerName())
		})
	}
}

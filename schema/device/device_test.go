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
			require.Equal(t, d.GetManufacturerName(), tt.want)
		})
	}
}

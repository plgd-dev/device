package core

import (
	"fmt"
	"strings"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/ocf"
	"github.com/go-ocf/sdk/schema"
)

type DiscoverDeviceCodec struct{}

// ContentFormat propagates the CoAP media type.
func (c DiscoverDeviceCodec) ContentFormat() gocoap.MediaType { return gocoap.MediaType(0) }

// Encode propagates the payload without any conversions.
func (c DiscoverDeviceCodec) Encode(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("not supported")
}

func anchorToDeviceId(anchor string) string {
	return strings.TrimPrefix(anchor, "ocf://")
}

type deviceLink struct {
	DeviceID string               `codec:"di"`
	Links    schema.ResourceLinks `codec:"links"`
}

func decodeDiscoverDevices(msg gocoap.Message, resources *schema.ResourceLinks) error {
	codec := ocf.VNDOCFCBORCodec{}
	var devices []deviceLink

	if err := codec.Decode(msg, &devices); err != nil {
		return fmt.Errorf("decoding %v failed: %w", ocf.DumpHeader(msg), err)
	}
	var resourceLinks schema.ResourceLinks
	for _, device := range devices {
		for _, link := range device.Links {
			if link.Anchor == "" {
				link.Anchor = "ocf://" + device.DeviceID
			}
			if link.DeviceID == "" {
				link.DeviceID = device.DeviceID
			}
			resourceLinks = append(resourceLinks, link)
		}
	}
	*resources = resourceLinks
	return nil
}

// Decode validates the content format and
// propagates the payload to v as *schema.ResourceLinks
func (c DiscoverDeviceCodec) Decode(msg gocoap.Message, v interface{}) error {
	resources, ok := v.(*schema.ResourceLinks)
	if !ok {
		return fmt.Errorf("invalid type %T", v)
	}

	cf := msg.Option(gocoap.ContentFormat)
	if cf == nil {
		return fmt.Errorf("content format not found")
	}
	mt, _ := cf.(gocoap.MediaType)
	switch mt {
	case gocoap.AppOcfCbor:
		codec := ocf.VNDOCFCBORCodec{}
		if err := codec.Decode(msg, resources); err != nil {
			return fmt.Errorf("decoding %v failed: %w", ocf.DumpHeader(msg), err)
		}
		return nil
	case gocoap.AppCBOR:
		return decodeDiscoverDevices(msg, resources)
	}
	return fmt.Errorf("not a VNDOCFCBOR content format: %v", cf)
}

package core

import (
	"fmt"
	"strings"

	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/ocf"
	"github.com/go-ocf/sdk/schema"
)

type DiscoverDeviceCodec struct{}

// ContentFormat propagates the CoAP media type.
func (c DiscoverDeviceCodec) ContentFormat() message.MediaType { return message.MediaType(0) }

// Encode propagates the payload without any conversions.
func (c DiscoverDeviceCodec) Encode(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("not supported")
}

func anchorToDeviceId(anchor string) string {
	return strings.TrimPrefix(anchor, "ocf://")
}

type deviceLink struct {
	DeviceID string               `json:"di"`
	Links    schema.ResourceLinks `json:"links"`
}

func decodeDiscoverDevices(msg *message.Message, resources *schema.ResourceLinks) error {
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
func (c DiscoverDeviceCodec) Decode(msg *message.Message, v interface{}) error {
	resources, ok := v.(*schema.ResourceLinks)
	if !ok {
		return fmt.Errorf("invalid type %T", v)
	}
	mt, err := msg.Options.ContentFormat()
	if err != nil {
		return fmt.Errorf("content format not found")
	}
	switch mt {
	case message.AppOcfCbor:
		codec := ocf.VNDOCFCBORCodec{}
		if err := codec.Decode(msg, resources); err != nil {
			return fmt.Errorf("decoding %v failed: %w", ocf.DumpHeader(msg), err)
		}
		return nil
	case message.AppCBOR:
		return decodeDiscoverDevices(msg, resources)
	}
	return fmt.Errorf("not a VNDOCFCBOR content format: %v", mt)
}

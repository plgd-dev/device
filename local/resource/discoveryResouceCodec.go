package resource

import (
	"fmt"
	"strings"

	gocoap "github.com/go-ocf/go-coap"
	coap "github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/schema"
)

type DiscoveryResourceCodec struct{}

// ContentFormat propagates the CoAP media type.
func (c DiscoveryResourceCodec) ContentFormat() gocoap.MediaType { return gocoap.MediaType(0) }

// Encode propagates the payload without any conversions.
func (c DiscoveryResourceCodec) Encode(v interface{}) ([]byte, error) {
	return nil, fmt.Errorf("not supported")
}

func anchorToDeviceId(anchor string) string {
	return strings.TrimPrefix(anchor, "ocf://")
}

func decodeDiscoveryOcfCbor(msg gocoap.Message, devices *[]schema.DeviceLinks) error {
	codec := coap.VNDOCFCBORCodec{}
	var resources []schema.ResourceLink

	if err := codec.Decode(msg, &resources); err != nil {
		return fmt.Errorf("decoding failed: %v: %s", err, coap.DumpHeader(msg))
	}
	m := make(map[string]map[string]schema.ResourceLink)
	for _, r := range resources {
		v, ok := m[r.Anchor]
		if !ok {
			v = make(map[string]schema.ResourceLink)
			m[r.Anchor] = v
		}
		v[r.Href] = r
	}
	devicesTmp := make([]schema.DeviceLinks, 0, len(m))
	for anchor, mResources := range m {
		d := schema.DeviceLinks{
			ID:     anchorToDeviceId(anchor),
			Anchor: anchor,
			Links:  make([]schema.ResourceLink, 0, len(mResources)),
		}
		for _, r := range mResources {
			d.Links = append(d.Links, r)
		}
		devicesTmp = append(devicesTmp, d)
	}
	*devices = devicesTmp
	return nil
}

// Decode validates the content format and
// propagates the payload to v as *[]schema.DeviceLinks.
func (c DiscoveryResourceCodec) Decode(msg gocoap.Message, v interface{}) error {
	devices, ok := v.(*[]schema.DeviceLinks)
	if !ok {
		return fmt.Errorf("invalid type %T", v)
	}

	cf := msg.Option(gocoap.ContentFormat)
	if cf == nil {
		return fmt.Errorf("content format not found")
	}
	mt, _ := cf.(gocoap.MediaType)
	switch mt {
	case gocoap.AppCBOR:
		codec := coap.VNDOCFCBORCodec{}
		if err := codec.Decode(msg, devices); err != nil {
			return fmt.Errorf("decoding failed: %v: %s", err, coap.DumpHeader(msg))
		}
		return nil
	case gocoap.AppOcfCbor:
		return decodeDiscoveryOcfCbor(msg, devices)
	}
	return fmt.Errorf("not a VNDOCFCBOR content format: %v", cf)
}

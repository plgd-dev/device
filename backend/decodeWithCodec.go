package backend

import (
	"fmt"

	"github.com/go-ocf/go-coap"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

func ContentTypeToMediaType(contentType string) (coap.MediaType, error) {
	switch contentType {
	case coap.TextPlain.String():
		return coap.TextPlain, nil
	case coap.AppCBOR.String():
		return coap.AppCBOR, nil
	case coap.AppOcfCbor.String():
		return coap.AppOcfCbor, nil
	case coap.AppJSON.String():
		return coap.AppJSON, nil
	default:
		return coap.TextPlain, fmt.Errorf("unknown content format")
	}
}

func DecodeContentWithCodec(codec kitNetCoap.Codec, contentType string, data []byte, response interface{}) error {
	if response == nil {
		return nil
	}
	if val, ok := response.(*[]byte); ok && len(data) == 0 {
		*val = data
		return nil
	}
	mediaType, err := ContentTypeToMediaType(contentType)
	if err != nil {
		return fmt.Errorf("cannot convert response contentype %v to mediatype: %w", contentType, err)
	}
	msg := coap.NewTcpMessage(coap.MessageParams{
		Payload: data,
	})
	msg.SetOption(coap.ContentFormat, mediaType)

	return codec.Decode(msg, response)
}

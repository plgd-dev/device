package backend

import (
	"fmt"

	"github.com/go-ocf/go-coap"
)

// ValidateContentType conbinations of cbor,ocfcbor are considered as valid.
func ValidateContentType(expected coap.MediaType, got string) error {
	if coap.MediaType(expected).String() == got {
		return nil
	}
	if expected == coap.AppCBOR {
		if coap.AppOcfCbor.String() == got {
			return nil
		}
	}
	if expected == coap.AppOcfCbor {
		if coap.AppCBOR.String() == got {
			return nil
		}
	}
	return fmt.Errorf("expected content type %s, got %s", coap.MediaType(expected).String(), got)
}

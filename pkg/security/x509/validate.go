package x509

import (
	"fmt"
	"net/url"
)

// ValidateCRLDistributionPoints validates a slice of CRL distribution point URLs.
// It ensures each URL is properly formatted and returns an error if any URL is invalid.
// Returns nil if all URLs are valid.
func ValidateCRLDistributionPoints(crlDistributionPoints []string) error {
	for _, crl := range crlDistributionPoints {
		if _, err := url.ParseRequestURI(crl); err != nil {
			return fmt.Errorf("invalid CRL distribution point URL %q: %w", crl, err)
		}
	}
	return nil
}

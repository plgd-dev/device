package link

import (
	"fmt"
	"regexp"
	"strings"
)

// Href provides a composite reference to a device resource.
type Href struct{ DeviceID, Href string }

// String formats the composite reference to a device resource.
func (h Href) String() string {
	return fmt.Sprintf("/%s/%s", h.DeviceID, strings.TrimPrefix(h.Href, "/"))
}

var (
	// https://godoc.org/github.com/google/uuid#Parse
	uuidPattern = `[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}`
	hrefRegexp  = regexp.MustCompile(`^/(` + uuidPattern + `)(/.*)$`)
)

// ParseHref parses the composite reference to a device resource.
func ParseHref(href string) (Href, error) {
	m := hrefRegexp.FindStringSubmatch(href)
	if len(m) != 3 {
		return Href{}, fmt.Errorf("invalid href: %s", href)
	}
	return Href{DeviceID: strings.ToLower(m[1]), Href: m[2]}, nil
}

// MustParseHref parses the composite reference to a device resource or panics.
func MustParseHref(href string) Href {
	h, err := ParseHref(href)
	if err != nil {
		panic(err)
	}
	return h
}

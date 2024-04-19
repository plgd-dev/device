package thingDescription

import (
	_ "embed"

	"github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

//go:embed ocfResources.jsonld
var ocfResourcesData []byte
var ocfThingDescription thingDescription.ThingDescription

func init() {
	err := ocfThingDescription.UnmarshalJSON(ocfResourcesData)
	if err != nil {
		panic(err)
	}
	ocfResourcesData = nil
}

func GetOCFThingDescription() thingDescription.ThingDescription {
	return ocfThingDescription
}

func GetOCFResourcePropertyElement(resourceHref string) (thingDescription.PropertyElement, bool) {
	if ocfThingDescription.Properties == nil {
		return thingDescription.PropertyElement{}, false
	}
	prop, ok := ocfThingDescription.Properties[resourceHref]
	if !ok {
		return thingDescription.PropertyElement{}, false
	}
	return prop, true
}

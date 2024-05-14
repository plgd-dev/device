package thingDescription

import (
	_ "embed"

	"github.com/google/uuid"
	bridgeTD "github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	schemaCloud "github.com/plgd-dev/device/v2/schema/cloud"
	schemaCredential "github.com/plgd-dev/device/v2/schema/credential"
	schemaDevice "github.com/plgd-dev/device/v2/schema/device"
	schemaMaintenance "github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/go-coap/v3/message"
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

func patchResourcePropertyElement(pe thingDescription.PropertyElement, deviceID uuid.UUID, resourceTypes []string, resourceHref string, contentType message.MediaType, createForms bridgeTD.CreateFormsFunc) (thingDescription.PropertyElement, error) {
	propOps := bridgeTD.GetPropertyElementOperations(pe)
	return bridgeTD.PatchPropertyElement(pe, resourceTypes, deviceID, resourceHref, propOps.ToSupportedOperations(), contentType, createForms)
}

func PatchDeviceResourcePropertyElement(pe thingDescription.PropertyElement, deviceID uuid.UUID, baseURL string, contentType message.MediaType, deviceType string, createForms bridgeTD.CreateFormsFunc) (thingDescription.PropertyElement, error) {
	var types []string
	if deviceType != "" {
		types = []string{schemaDevice.ResourceType, deviceType}
	}
	return patchResourcePropertyElement(pe, deviceID, types, baseURL+schemaDevice.ResourceURI, contentType, createForms)
}

func PatchMaintenanceResourcePropertyElement(pe thingDescription.PropertyElement, deviceID uuid.UUID, baseURL string, contentType message.MediaType, createForms bridgeTD.CreateFormsFunc) (thingDescription.PropertyElement, error) {
	return patchResourcePropertyElement(pe, deviceID, []string{schemaMaintenance.ResourceType}, baseURL+schemaMaintenance.ResourceURI, contentType, createForms)
}

func PatchCloudResourcePropertyElement(pe thingDescription.PropertyElement, deviceID uuid.UUID, baseURL string, contentType message.MediaType, createForms bridgeTD.CreateFormsFunc) (thingDescription.PropertyElement, error) {
	return patchResourcePropertyElement(pe, deviceID, []string{schemaCloud.ResourceType}, baseURL+schemaCloud.ResourceURI, contentType, createForms)
}

func PatchCredentialResourcePropertyElement(pe thingDescription.PropertyElement, deviceID uuid.UUID, baseURL string, contentType message.MediaType, createForms bridgeTD.CreateFormsFunc) (thingDescription.PropertyElement, error) {
	return patchResourcePropertyElement(pe, deviceID, []string{schemaCredential.ResourceType}, baseURL+schemaCredential.ResourceURI, contentType, createForms)
}

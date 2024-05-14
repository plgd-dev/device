package device

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	bridgeTD "github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

const (
	DeviceResourceType      = "oic.d.virtual"
	TestResourcePropertyKey = "my-property"
	TestResourceType        = "x.plgd.test"
)

func GetTestResourceHref(id int) string {
	return "/test/" + strconv.Itoa(id)
}

func GetPropertyDescriptionForTestResource() thingDescription.PropertyElement {
	objectType := thingDescription.Object
	stringType := thingDescription.String
	return thingDescription.PropertyElement{
		Type: &thingDescription.TypeDeclaration{
			StringArray: []string{TestResourceType},
		},
		Title:               bridgeTD.StringToPtr("Test Property"),
		PropertyElementType: &objectType,
		Properties: &thingDescription.Properties{
			DataSchemaMap: map[string]thingDescription.DataSchema{
				"Name": {
					Title:          bridgeTD.StringToPtr("Name"),
					DataSchemaType: &stringType,
				},
			},
		},
		Observable: bridgeTD.BoolToPtr(true),
	}
}

func PatchTestResourcePropertyElement(pe thingDescription.PropertyElement, deviceID uuid.UUID, href string, contentType message.MediaType, createForms bridgeTD.CreateFormsFunc) (thingDescription.PropertyElement, error) {
	propOps := bridgeTD.GetPropertyElementOperations(pe)
	return bridgeTD.PatchPropertyElement(pe, []string{TestResourceType}, deviceID, href,
		propOps.ToSupportedOperations(), contentType, createForms)
}

func GetAdditionalProperties() map[string]interface{} {
	return map[string]interface{}{
		TestResourcePropertyKey: "my-value",
	}
}

func GetDataSchemaForAdditionalProperties() map[string]thingDescription.DataSchema {
	dsm := map[string]thingDescription.DataSchema{}
	stringType := thingDescription.String
	readOnly := true
	dsm[TestResourcePropertyKey] = thingDescription.DataSchema{
		DataSchemaType: &stringType,
		ReadOnly:       &readOnly,
	}
	return dsm
}

func GetThingDescription(path string, numResources int) (thingDescription.ThingDescription, error) {
	tdJson, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return thingDescription.ThingDescription{}, err
	}
	td, err := thingDescription.UnmarshalThingDescription(tdJson)
	if err != nil {
		return thingDescription.ThingDescription{}, err
	}
	if td.Properties == nil {
		td.Properties = make(map[string]thingDescription.PropertyElement)
	}
	for i := 0; i < numResources; i++ {
		td.Properties[GetTestResourceHref(i)] = GetPropertyDescriptionForTestResource()
	}
	return td, nil
}

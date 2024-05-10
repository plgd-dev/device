package thingDescription

import (
	"net/http"
	"net/url"

	"github.com/fredbi/uri"
	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

var (
	SecurityNoSec       = "nosec_sc"
	SecurityDefinitions = map[string]thingDescription.SecurityScheme{
		SecurityNoSec: {
			Scheme: "nosec",
		},
	}
	HTTPSWWWW3Org2022WotTdV11 = thingDescription.HTTPSWWWW3Org2022WotTdV11
	Context                   = thingDescription.ThingContext{
		Enum: &HTTPSWWWW3Org2022WotTdV11,
	}
)

func SupportedOperationToTDOperations(ops resources.SupportedOperation) []string {
	tdOps := make([]string, 0, 3)
	type translationItem struct {
		resourceOp resources.SupportedOperation
		tdOps      []string
	}
	translationTable := []translationItem{
		{resources.SupportedOperationRead, []string{string(thingDescription.Readproperty)}},
		{resources.SupportedOperationWrite, []string{string(thingDescription.Writeproperty)}},
		{resources.SupportedOperationObserve, []string{string(thingDescription.Observeproperty), string(thingDescription.Unobserveproperty)}},
	}
	for _, t := range translationTable {
		if ops.HasOperation(t.resourceOp) {
			tdOps = append(tdOps, t.tdOps...)
		}
	}
	if len(tdOps) == 0 {
		return nil
	}
	return tdOps
}

func BoolToPtr(v bool) *bool {
	if !v {
		return nil
	}
	return &v
}

func StringToPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

type PropertyElementOperations struct {
	ReadOnly   bool
	WriteOnly  bool
	Observable bool
}

func toPropertyElementOperations(ops resources.SupportedOperation) PropertyElementOperations {
	return PropertyElementOperations{
		Observable: ops.HasOperation(resources.SupportedOperationObserve),
		ReadOnly:   ops.HasOperation(resources.SupportedOperationRead) && !ops.HasOperation(resources.SupportedOperationWrite),
		WriteOnly:  ops.HasOperation(resources.SupportedOperationWrite) && !ops.HasOperation(resources.SupportedOperationRead),
	}
}

func GetPropertyElementOperations(pe thingDescription.PropertyElement) PropertyElementOperations {
	isNotNilAndTrue := func(val *bool) bool {
		return val != nil && *val
	}
	return PropertyElementOperations{
		ReadOnly:   isNotNilAndTrue(pe.ReadOnly),
		WriteOnly:  isNotNilAndTrue(pe.WriteOnly),
		Observable: isNotNilAndTrue(pe.Observable),
	}
}

func (p PropertyElementOperations) ToSupportedOperations() resources.SupportedOperation {
	var ops resources.SupportedOperation
	if p.Observable {
		ops |= resources.SupportedOperationObserve
	}
	if p.ReadOnly {
		return ops | resources.SupportedOperationRead
	}
	if p.WriteOnly {
		return ops | resources.SupportedOperationWrite
	}
	return ops | resources.SupportedOperationRead | resources.SupportedOperationWrite
}

func createForm(hrefUri *url.URL, covMethod string, op thingDescription.StickyDescription, contentType message.MediaType) thingDescription.FormElementProperty {
	additionalFields := map[string]interface{}{
		"cov:method": covMethod,
		"cov:accept": float64(contentType),
	}
	ops := []string{string(op)}
	if op == thingDescription.Observeproperty {
		additionalFields["subprotocol"] = "cov:observe"
		ops = append(ops, string(thingDescription.Unobserveproperty))
	}

	return thingDescription.FormElementProperty{
		ContentType: StringToPtr(contentType.String()),
		Href:        *hrefUri,
		Op: &thingDescription.FormElementPropertyOp{
			StringArray: ops,
		},
		AdditionalFields: additionalFields,
	}
}

func SetForms(hrefUri *url.URL, ops resources.SupportedOperation, contentType message.MediaType) []thingDescription.FormElementProperty {
	forms := make([]thingDescription.FormElementProperty, 0, 3)
	if ops.HasOperation(resources.SupportedOperationWrite) {
		forms = append(forms, createForm(hrefUri, http.MethodPost, thingDescription.Writeproperty, contentType))
	}
	if ops.HasOperation(resources.SupportedOperationRead) {
		forms = append(forms, createForm(hrefUri, http.MethodGet, thingDescription.Readproperty, contentType))
	}
	if ops.HasOperation(resources.SupportedOperationObserve) {
		forms = append(forms, createForm(hrefUri, http.MethodGet, thingDescription.Observeproperty, contentType))
	}
	return forms
}

func PatchPropertyElement(prop thingDescription.PropertyElement, types []string, setForm bool, deviceID uuid.UUID, href string, ops resources.SupportedOperation, contentType message.MediaType) (thingDescription.PropertyElement, error) {
	if len(types) > 0 {
		prop.Type = &thingDescription.TypeDeclaration{
			StringArray: types,
		}
	}
	propOps := toPropertyElementOperations(ops)
	prop.Observable = BoolToPtr(propOps.Observable)
	prop.ReadOnly = BoolToPtr(propOps.ReadOnly)
	prop.WriteOnly = BoolToPtr(propOps.WriteOnly)
	if !setForm {
		return prop, nil
	}
	opsStrs := SupportedOperationToTDOperations(ops)
	if len(opsStrs) == 0 {
		return prop, nil
	}
	var hrefUri *url.URL
	if len(href) > 0 {
		hrefStr := href
		if deviceID != uuid.Nil {
			hrefStr += "?di=" + deviceID.String()
		}
		var err error
		hrefUri, err = url.Parse(hrefStr)
		if err != nil {
			return thingDescription.PropertyElement{}, err
		}
	}
	prop.Forms = SetForms(hrefUri, ops, contentType)
	return prop, nil
}

func GetThingDescriptionID(deviceID string) (uri.URI, error) {
	return uri.Parse("urn:uuid:" + deviceID)
}

func PatchThingDescription(td thingDescription.ThingDescription, device Device, endpoint string, getPropertyElement func(resourceHref string, resource Resource) (thingDescription.PropertyElement, bool)) thingDescription.ThingDescription {
	if td.Context == nil {
		td.Context = &Context
	}
	id, err := GetThingDescriptionID(device.GetID().String())
	if err == nil {
		td.ID = id
	}
	td.Title = device.GetName()
	if endpoint != "" {
		// base
		u, err := url.Parse(endpoint)
		if err == nil {
			td.Base = *u
		}
		// security
		td.Security = &thingDescription.TypeDeclaration{
			String: &SecurityNoSec,
		}
		// securityDefinitions
		td.SecurityDefinitions = SecurityDefinitions
	}

	device.Range(func(resourceHref string, resource Resource) bool {
		pe, ok := getPropertyElement(resourceHref, resource)
		if !ok {
			return true
		}
		if td.Properties == nil {
			td.Properties = make(map[string]thingDescription.PropertyElement)
		}
		td.Properties[resourceHref] = pe
		return true
	})
	return td
}

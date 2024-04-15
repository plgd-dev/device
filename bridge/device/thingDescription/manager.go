package thingDescription

import (
	"net/url"
	"reflect"
	"sync/atomic"

	"github.com/fredbi/uri"
	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/pkg/eventloop"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
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

// Resource to avoid import cycle also it is same as in Device package to avoid wrapping it
type Resource = interface {
	Close()
	ETag() []byte
	GetHref() string
	GetResourceTypes() []string
	GetResourceInterfaces() []string
	HandleRequest(req *net.Request) (*pool.Message, error)
	GetPolicyBitMask() schema.BitMask
	SetObserveHandler(loop *eventloop.Loop, createSubscription resources.CreateSubscriptionFunc)
	UpdateETag()
	SupportsOperations() resources.SupportedOperation
}

type Device = interface {
	GetID() uuid.UUID
	GetName() string
	Range(f func(resourceHref string, resource Resource) bool)
}

type SubscriptionHandler = func(td *thingDescription.ThingDescription, closed bool)

type Manager struct {
	device Device

	subscriptions    *sync.Map[uint64, SubscriptionHandler]
	lastSubscription atomic.Uint64
	loop             *eventloop.Loop
	subChan          chan struct{}
	lastTD           atomic.Pointer[thingDescription.ThingDescription]
	stopped          atomic.Bool
}

func New(device Device, loop *eventloop.Loop) *Manager {
	subChan := make(chan struct{}, 1)
	t := Manager{
		device:        device,
		subscriptions: sync.NewMap[uint64, SubscriptionHandler](),
		loop:          loop,
		subChan:       subChan,
	}
	loop.Add(eventloop.NewReadHandler(reflect.ValueOf(subChan), t.subscriptionHandler))
	return &t
}

func (t *Manager) Close() {
	if !t.stopped.CompareAndSwap(false, true) {
		return
	}
	t.loop.RemoveByChannels(reflect.ValueOf(t.subChan))
	close(t.subChan)
	for _, sub := range t.subscriptions.LoadAndDeleteAll() {
		sub(nil, true)
	}
}

func (t *Manager) subscriptionHandler(_ reflect.Value, closed bool) {
	if closed {
		return
	}

	td := t.lastTD.Load()
	if td == nil {
		return
	}
	t.subscriptions.Range(func(_ uint64, value SubscriptionHandler) bool {
		value(td, false)
		return true
	})
	t.lastTD.CompareAndSwap(td, nil)
}

func (t *Manager) RegisterSubscription(subscription SubscriptionHandler) func() {
	id := t.lastSubscription.Add(1)
	t.subscriptions.Store(id, subscription)
	return func() {
		t.subscriptions.Delete(id)
	}
}

func (t *Manager) NotifySubscriptions(td thingDescription.ThingDescription) {
	t.lastTD.Store(&td)
	select {
	case t.subChan <- struct{}{}:
	default:
	}
}

func supportedOperationToTDOperation(ops resources.SupportedOperation) []string {
	tdOps := make([]string, 0, 3)
	if ops.HasOperation(resources.SupportedOperationRead) {
		tdOps = append(tdOps, string(thingDescription.Readproperty))
	}
	if ops.HasOperation(resources.SupportedOperationWrite) {
		tdOps = append(tdOps, string(thingDescription.Writeproperty))
	}
	if ops.HasOperation(resources.SupportedOperationObserve) {
		tdOps = append(tdOps, string(thingDescription.Observeproperty), string(thingDescription.Unobserveproperty))
	}
	if len(tdOps) == 0 {
		return nil
	}
	return tdOps
}

func boolToPtr(v bool) *bool {
	if !v {
		return nil
	}
	return &v
}

func stringToPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func createForms(deviceID uuid.UUID, href string, supportedOperations resources.SupportedOperation, setForm bool) []thingDescription.FormElementProperty {
	if !setForm {
		return nil
	}
	ops := supportedOperationToTDOperation(supportedOperations)
	if len(ops) > 0 {
		hrefStr := href
		if deviceID != uuid.Nil {
			hrefStr += "?di=" + deviceID.String()
		}
		href, err := url.Parse(hrefStr)
		if err == nil {
			return []thingDescription.FormElementProperty{
				{
					ContentType: stringToPtr(message.AppCBOR.String()),
					Op: &thingDescription.FormElementPropertyOp{
						StringArray: ops,
					},
					Href: *href,
				},
			}
		}
	}
	return nil
}

func PatchPropertyElement(prop thingDescription.PropertyElement, deviceID uuid.UUID, resource Resource, setForm bool) thingDescription.PropertyElement {
	ops := resource.SupportsOperations()
	observable := ops.HasOperation(resources.SupportedOperationObserve)
	isReadOnly := ops.HasOperation(resources.SupportedOperationRead) && !ops.HasOperation(resources.SupportedOperationWrite)
	isWriteOnly := ops.HasOperation(resources.SupportedOperationWrite) && !ops.HasOperation(resources.SupportedOperationRead)
	resourceTypes := resource.GetResourceTypes()

	prop.Type = &thingDescription.TypeDeclaration{
		StringArray: resourceTypes,
	}
	prop.Observable = boolToPtr(observable)
	prop.ReadOnly = boolToPtr(isReadOnly)
	prop.WriteOnly = boolToPtr(isWriteOnly)
	prop.Observable = boolToPtr(observable)
	prop.Forms = createForms(deviceID, resource.GetHref(), ops, setForm)
	return prop
}

func PatchThingDescription(td thingDescription.ThingDescription, device Device, endpoint string, getPropertyElement func(resourceHref string, resource Resource) (thingDescription.PropertyElement, bool)) thingDescription.ThingDescription {
	if td.Context == nil {
		td.Context = &Context
	}
	id, err := uri.Parse("urn:uuid:" + device.GetID().String())
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

package thingDescription

import (
	"reflect"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/pkg/eventloop"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/web-of-things-open-source/thingdescription-go/thingDescription"
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

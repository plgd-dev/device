package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/kit/v2/codec/ocf"
	"go.uber.org/atomic"
)

func (d *Device) ObserveResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec coap.Codec,
	handler ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	return d.observeResource(ctx, link, codec, handler, options...)
}

type ObservationHandler interface {
	Handle(ctx context.Context, body coap.DecodeFunc)
	OnClose()
	Error(err error)
}

func (d *Device) ObserveResource(
	ctx context.Context,
	link schema.ResourceLink,
	handler ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	codec := ocf.VNDOCFCBORCodec{}
	return d.ObserveResourceWithCodec(ctx, link, codec, handler, options...)
}

func (d *Device) stopObservingResource(
	ctx context.Context,
	observationID string,
	byClose bool,
) (bool, error) {
	v, ok := d.observations.Load(observationID)
	if !ok {
		return false, nil
	}
	d.observations.Delete(observationID)
	o := v.(*observation)
	var err error
	if byClose {
		err = o.Close(ctx)
	} else {
		err = o.Stop(ctx)
	}
	if err != nil {
		return false, MakeCanceled(fmt.Errorf("could not cancel observation %s: %w", observationID, err))
	}

	return true, nil
}

func (d *Device) StopObservingResource(
	ctx context.Context,
	observationID string,
) (bool, error) {
	return d.stopObservingResource(ctx, observationID, false)
}

func (d *Device) closeObservations(ctx context.Context) error {
	obs := make([]string, 0, 12)
	d.observations.Range(func(key, value interface{}) bool {
		observationID := key.(string)
		obs = append(obs, observationID)
		return false
	})
	var errors *multierror.Error
	for _, observationID := range obs {
		_, err := d.stopObservingResource(ctx, observationID, true)
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

type observation struct {
	id      string
	handler *observationHandler
	client  *coap.ClientCloseHandler

	lock      sync.Mutex
	onCloseID int
	obs       coap.Observation
}

func (o *observation) Set(onCloseID int, obs coap.Observation) {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.onCloseID = onCloseID
	o.obs = obs
}

func (o *observation) Get() (onCloseID int, obs coap.Observation) {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.onCloseID, o.obs
}

func (o *observation) stop(ctx context.Context, byClose bool) error {
	onCloseID, obs := o.Get()
	o.client.UnregisterCloseHandler(onCloseID)
	if obs != nil {
		if byClose {
			// observation was closed by device.Close()
			o.handler.Close()
		} else {
			// stop was called by user so we don't want to call OnClose handler
			o.handler.disableHandlers()
		}
		err := obs.Cancel(ctx)
		if err != nil {
			return MakeCanceled(fmt.Errorf("cannot cancel observation %s: %w", o.id, err))
		}
		return err
	}
	return nil
}

func (o *observation) Stop(ctx context.Context) error {
	return o.stop(ctx, false)
}

func (o *observation) Close(ctx context.Context) error {
	return o.stop(ctx, true)
}

func (d *Device) observeResource(
	ctx context.Context, link schema.ResourceLink,
	codec coap.Codec,
	handler ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	eps := link.GetSecureEndpoints()
	if len(eps) == 0 {
		eps = link.GetEndpoints()
	}

	_, client, err := d.connectToEndpoints(ctx, eps)
	if err != nil {
		return "", MakeInternal(fmt.Errorf("cannot observe resource %v: %w", link.Href, err))
	}

	options = append(options, coap.WithAccept(codec.ContentFormat()))

	id, err := uuid.NewRandom()
	if err != nil {
		return "", MakeInternal(fmt.Errorf("observation id generation failed: %w", err))
	}
	h := observationHandler{handler: handler}
	o := &observation{
		id:      id.String(),
		handler: &h,
		client:  client,
	}
	onCloseID := client.RegisterCloseHandler(func(error) {
		o.handler.OnClose()
		obsCtx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, errClose := d.StopObservingResource(obsCtx, o.id); errClose != nil {
			o.handler.Error(fmt.Errorf("failed to stop observing resource(%s): %w", link.Href, errClose))
		}
	})

	obs, err := client.Observe(ctx, link.Href, codec, &h, options...)
	if err != nil {
		client.UnregisterCloseHandler(o.onCloseID)
		return "", err
	}

	o.Set(onCloseID, obs)

	d.observations.Store(o.id, o)
	return o.id, nil
}

type observationHandler struct {
	mutex    sync.Mutex
	handler  ObservationHandler
	isClosed atomic.Bool
}

func (h *observationHandler) disableHandlers() {
	h.isClosed.Store(true)
}

func (h *observationHandler) Handle(client *coap.Client, body coap.DecodeFunc) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.isClosed.Load() {
		return
	}
	h.handler.Handle(client.Context(), body)
}

func (h *observationHandler) Error(err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)
	h.handler.Error(err)
}

func (h *observationHandler) Close() {
	h.OnClose()
}

func (h *observationHandler) OnClose() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)
	h.handler.OnClose()
}

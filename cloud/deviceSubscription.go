package cloud

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/go-ocf/grpc-gateway/pb"
	"github.com/go-ocf/kit/net/grpc"
)

// ResourcePublishedHandler handler of events.
type ResourcePublishedHandler = interface {
	HandleResourcePublished(ctx context.Context, val *pb.Event_ResourcePublished) error
	SubscriptionHandler
}

// ResourceUnpublishedHandler handler of events.
type ResourceUnpublishedHandler = interface {
	HandleResourceUnpublished(ctx context.Context, val *pb.Event_ResourceUnpublished) error
	SubscriptionHandler
}

// DeviceSubscription subscription.
type DeviceSubscription struct {
	client                     pb.GrpcGateway_SubscribeForEventsClient
	subscriptionID             string
	handle                     SubscriptionHandler
	resourcePublishedHandler   ResourcePublishedHandler
	resourceUnpublishedHandler ResourceUnpublishedHandler

	wg sync.WaitGroup

	canceled uint32
}

// NewDeviceSubscription creates new devices subscriptions to listen events: resource published, resource unpublished.
func (c *Client) NewDeviceSubscription(ctx context.Context, token, deviceID string, handle SubscriptionHandler) (*DeviceSubscription, error) {
	var resourcePublishedHandler ResourcePublishedHandler
	var resourceUnpublishedHandler ResourceUnpublishedHandler
	filterEvents := make([]pb.SubscribeForEvents_DeviceEventFilter_Event, 0, 1)
	if v, ok := handle.(ResourcePublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_PUBLISHED)
		resourcePublishedHandler = v
	}
	if v, ok := handle.(ResourceUnpublishedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_DeviceEventFilter_RESOURCE_UNPUBLISHED)
		resourceUnpublishedHandler = v
	}

	if resourcePublishedHandler == nil && resourceUnpublishedHandler == nil {
		return nil, fmt.Errorf("invalid handler - it's supports: ResourceContentChangedHandler")
	}
	ctx = grpc.CtxWithToken(ctx, token)
	client, err := c.gateway.SubscribeForEvents(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Send(&pb.SubscribeForEvents{
		FilterBy: &pb.SubscribeForEvents_DeviceEvent{
			DeviceEvent: &pb.SubscribeForEvents_DeviceEventFilter{
				DeviceId:     deviceID,
				FilterEvents: filterEvents,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	ev, err := client.Recv()
	if err != nil {
		return nil, err
	}
	op := ev.GetOperationProcessed()
	if op == nil {
		return nil, fmt.Errorf("unexpected event %+v", ev)
	}
	if op.GetErrorStatus().GetCode() != pb.Event_OperationProcessed_ErrorStatus_OK {
		return nil, fmt.Errorf(op.GetErrorStatus().GetMessage())
	}

	sub := &DeviceSubscription{
		client:                     client,
		handle:                     handle,
		subscriptionID:             ev.GetSubscriptionId(),
		resourcePublishedHandler:   resourcePublishedHandler,
		resourceUnpublishedHandler: resourceUnpublishedHandler,
	}
	sub.wg.Add(1)
	go sub.runRecv()

	return sub, nil
}

// Cancel cancels subscription.
func (s *DeviceSubscription) Cancel() error {
	if !atomic.CompareAndSwapUint32(&s.canceled, s.canceled, 1) {
		return nil
	}
	err := s.client.CloseSend()
	if err != nil {
		return err
	}
	s.wg.Wait()
	return nil
}

// ID returns subscription id.
func (s *DeviceSubscription) ID() string {
	return s.subscriptionID
}

func (s *DeviceSubscription) runRecv() {
	defer s.wg.Done()
	for {
		ev, err := s.client.Recv()
		if err == io.EOF {
			s.handle.OnClose()
			return
		}
		if err != nil {
			s.handle.Error(err)
			return
		}
		cancel := ev.GetSubscriptionCanceled()
		if cancel != nil {
			reason := cancel.GetReason()
			if reason == "" {
				s.handle.OnClose()
			}
			s.handle.Error(fmt.Errorf(reason))
			return
		}

		if ct := ev.GetResourcePublished(); ct != nil {
			err = s.resourcePublishedHandler.HandleResourcePublished(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.handle.Error(err)
				return
			}
		} else if ct := ev.GetResourceUnpublished(); ct != nil {
			err = s.resourceUnpublishedHandler.HandleResourceUnpublished(s.client.Context(), ct)
			if err != nil {
				s.Cancel()
				s.handle.Error(err)
				return
			}
		} else {
			s.Cancel()
			s.handle.Error(fmt.Errorf("unknown event occurs on recv resource content changed: %+v", ev))
			return
		}
	}
}

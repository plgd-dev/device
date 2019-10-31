package cloud

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/go-ocf/grpc-gateway/pb"
)

// SubscriptionHandler handler of events.
type SubscriptionHandler = interface {
	OnClose()
	Error(err error)
}

// ResourceContentChangedHandler handler of events.
type ResourceContentChangedHandler = interface {
	HandleResourceContentChanged(ctx context.Context, val *pb.Event_ResourceContentChanged) error
	SubscriptionHandler
}

// ResourceSubscription subscription.
type ResourceSubscription struct {
	client                        pb.GrpcGateway_SubscribeForEventsClient
	subscriptionID                string
	handle                        SubscriptionHandler
	resourceContentChangedHandler ResourceContentChangedHandler
	wg                            sync.WaitGroup

	canceled uint32
}

// NewResourceSubscription creates new resource content changed subscription.
// JWT token must be stored in context for grpc call.
func (c *Client) NewResourceSubscription(ctx context.Context, resourceID pb.ResourceId, handle SubscriptionHandler) (*ResourceSubscription, error) {
	var resourceContentChangedHandler ResourceContentChangedHandler
	filterEvents := make([]pb.SubscribeForEvents_ResourceEventFilter_Event, 0, 1)
	if v, ok := handle.(ResourceContentChangedHandler); ok {
		filterEvents = append(filterEvents, pb.SubscribeForEvents_ResourceEventFilter_CONTENT_CHANGED)
		resourceContentChangedHandler = v
	}

	if resourceContentChangedHandler == nil {
		return nil, fmt.Errorf("invalid handler - it's supports: ResourceContentChangedHandler")
	}
	client, err := c.gateway.SubscribeForEvents(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Send(&pb.SubscribeForEvents{
		FilterBy: &pb.SubscribeForEvents_ResourceEvent{
			ResourceEvent: &pb.SubscribeForEvents_ResourceEventFilter{
				ResourceId:   &resourceID,
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

	sub := &ResourceSubscription{
		client:                        client,
		handle:                        handle,
		subscriptionID:                ev.GetSubscriptionId(),
		resourceContentChangedHandler: resourceContentChangedHandler,
	}
	sub.wg.Add(1)
	go sub.runRecv()

	return sub, nil
}

// Cancel cancels subscription.
func (s *ResourceSubscription) Cancel() error {
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
func (s *ResourceSubscription) ID() string {
	return s.subscriptionID
}

func (s *ResourceSubscription) runRecv() {
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

		if ct := ev.GetResourceContentChanged(); ct != nil {
			err = s.resourceContentChangedHandler.HandleResourceContentChanged(s.client.Context(), ct)
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

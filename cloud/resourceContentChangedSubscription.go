package cloud

import (
	"context"
	"fmt"
	"io"

	"github.com/go-ocf/grpc-gateway/pb"
	"github.com/go-ocf/kit/net/grpc"
)

// ResourceContentChangedSubscriptionHandler handler of events.
type ResourceContentChangedSubscriptionHandler = interface {
	Handle(ctx context.Context, val *pb.Event_ResourceContentChanged) error
	OnClose()
	Error(err error)
}

// ResourceContentChangedSubscription subscription.
type ResourceContentChangedSubscription struct {
	client         pb.GrpcGateway_SubscribeForEventsClient
	subscriptionID string
	handle         ResourceContentChangedSubscriptionHandler
}

// NewResourceContentChangedSubscription creates new resource content changed subscription.
func (c *Client) NewResourceContentChangedSubscription(ctx context.Context, token string, resourceID pb.ResourceId, handle ResourceContentChangedSubscriptionHandler) (*ResourceContentChangedSubscription, error) {
	ctx = grpc.CtxWithToken(ctx, token)
	client, err := c.gateway.SubscribeForEvents(ctx)
	if err != nil {
		return nil, err
	}

	err = client.Send(&pb.SubscribeForEvents{
		FilterBy: &pb.SubscribeForEvents_ResourceEvent{
			ResourceEvent: &pb.SubscribeForEvents_ResourceEventFilter{
				ResourceId: &resourceID,
				FilterEvents: []pb.SubscribeForEvents_ResourceEventFilter_Event{
					pb.SubscribeForEvents_ResourceEventFilter_CONTENT_CHANGED,
				},
			},
		},
		AuthorizationContext: &pb.AuthorizationContext{
			AccessToken: token,
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

	sub := &ResourceContentChangedSubscription{
		client:         client,
		handle:         handle,
		subscriptionID: ev.GetSubscriptionId(),
	}
	go sub.runRecv()

	return sub, nil
}

// Cancel cancels subscription.
func (s *ResourceContentChangedSubscription) Cancel() error {
	return s.client.CloseSend()
}

// ID returns subscription id.
func (s *ResourceContentChangedSubscription) ID() string {
	return s.subscriptionID
}

func (s *ResourceContentChangedSubscription) runRecv() {
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
		ct := ev.GetResourceContentChanged()
		if ct == nil {
			s.Cancel()
			s.handle.Error(fmt.Errorf("unknown event occurs on recv resource content changed: %+v", ev))
			return
		}
		err = s.handle.Handle(s.client.Context(), ct)
		if err != nil {
			s.Cancel()
			s.handle.Error(err)
			return
		}
	}
}

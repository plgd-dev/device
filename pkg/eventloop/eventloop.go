package eventloop

import (
	"fmt"
	"reflect"
	"sort"
	"sync/atomic"
)

// HandlerFunc is a function type that represents a handler for events in the event loop.
// It takes two parameters: `value` of type `reflect.Value` which represents the event value,
// and `closed` of type `bool` which indicates whether the event loop is closed.
type HandlerFunc func(value reflect.Value, closed bool)

// Handler represents a handler for processing events.
type Handler struct {
	ch reflect.Value // ch is the channel for receiving events. You can get it via calling reflect.ValueOf(ch).
	h  HandlerFunc   // h is the function to handle the events.
}

// NewReadHandler creates a new Handler instance with the given channel and handler function.
// The channel is used to receive events, and the handler function is called to handle each event.
func NewReadHandler(ch reflect.Value, h HandlerFunc) Handler {
	return Handler{ch: ch, h: h}
}

// Loop represents an event loop that handles events using registered handlers.
type Loop struct {
	handlers atomic.Pointer[[]Handler]
	wake     chan struct{}
}

// New creates a new event loop.
func New() *Loop {
	return &Loop{
		wake: make(chan struct{}, 1),
	}
}

// Wake wakes up the event loop by sending a signal through the wake channel.
// If the wake channel is already full, the signal is dropped.
func (el *Loop) Wake() {
	select {
	case el.wake <- struct{}{}:
	default:
	}
}

// Add adds one or more event handlers to the event loop.
// The event handlers will be executed when the event loop is triggered.
// It is safe to call this method concurrently from multiple goroutines.
// The order in which the event handlers are added is preserved.
// The event loop will wake up after the handlers are added.
func (el *Loop) Add(h ...Handler) {
	defer el.Wake()
	for {
		origHandlers := el.handlers.Load()
		origLen := 0
		if origHandlers != nil {
			origLen = len(*origHandlers)
		}
		handlers := make([]Handler, origLen+len(h))
		if origHandlers != nil {
			copy(handlers, *origHandlers)
		}
		copy(handlers[origLen:], h)
		if el.handlers.CompareAndSwap(origHandlers, &handlers) {
			return
		}
	}
}

// Remove removes the specified handlers from the event loop.
// It takes one or more handlers as arguments and removes them by their channels.
// The channels of the handlers are collected into a slice and passed to the RemoveByChannels method.
func (el *Loop) Remove(h ...Handler) {
	chs := make([]reflect.Value, 0, len(h))
	for _, handler := range h {
		chs = append(chs, handler.ch)
	}
	el.RemoveByChannels(chs...)
}

// removeByChannels removes the handlers associated with the specified channels from the event loop without waking it up.
func (el *Loop) removeByChannels(ch ...reflect.Value) {
	sort.Slice(ch, func(i, j int) bool {
		return ch[i].Pointer() < ch[j].Pointer()
	})
	for {
		origHandlers := el.handlers.Load()
		origLen := 0
		if origHandlers != nil {
			origLen = len(*origHandlers)
		}
		handlers := make([]Handler, 0, origLen)
		if origHandlers != nil {
			for _, h := range *origHandlers {
				i := sort.Search(len(ch), func(i int) bool {
					return ch[i].Pointer() >= h.ch.Pointer()
				})
				if i < len(ch) && ch[i].Pointer() == h.ch.Pointer() {
					continue
				}
				handlers = append(handlers, h)
			}
		}
		if el.handlers.CompareAndSwap(origHandlers, &handlers) {
			return
		}
	}
}

// RemoveByChannels removes the handlers associated with the specified channels from the event loop.
// The channels are sorted based on their memory addresses before removing the handlers.
// After removing the handlers, the event loop is woken up to process any pending events.
func (el *Loop) RemoveByChannels(ch ...reflect.Value) {
	el.removeByChannels(ch...)
	el.Wake()
}

// Run starts the event loop and runs until the stop channel is closed.
// It continuously listens for events on the stop channel and the wake channel.
// When an event is received on the stop channel, the event loop stops and returns.
// When an event is received on the wake channel, the event loop continues to the next iteration.
// The event loop executes the registered handlers based on the received events.
// If an event handler returns an error, the event loop removes the handler and calls it with a nil value and the error set to true.
// If an event handler panics, the event loop removes the handler and panics with a descriptive error message.
func (el *Loop) Run(stop <-chan struct{}) {
	lastHandlers, cases := el.getCases(nil, []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(stop)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(el.wake)},
	}, stop)
	for {
		lastHandlers, cases = el.getCases(lastHandlers, cases, stop)
		idx, recv, ok := reflect.Select(cases)
		if idx == 0 {
			return
		}
		if idx == 1 {
			continue
		}
		if idx-2 >= len(*lastHandlers) {
			panic(fmt.Sprintf("eventloop: idx(%v) is greater than number of handlers(%v)", idx-2, len(*lastHandlers)))
		}
		if ok {
			(*lastHandlers)[idx-2].h(recv, false)
		} else {
			el.removeByChannels((*lastHandlers)[idx-2].ch)
			(*lastHandlers)[idx-2].h(reflect.ValueOf(nil), true)
		}
	}
}

func (el *Loop) getCases(lastHandler *[]Handler, cases []reflect.SelectCase, stop <-chan struct{}) (*[]Handler, []reflect.SelectCase) {
	handlers := el.handlers.Load()
	if handlers == lastHandler {
		return handlers, cases
	}
	cases = make([]reflect.SelectCase, len(*handlers)+2)
	cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(stop)}
	cases[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(el.wake)}
	for i, h := range *handlers {
		cases[i+2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: h.ch}
	}
	return handlers, cases
}

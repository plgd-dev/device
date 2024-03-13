package eventloop

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoopRun(t *testing.T) {
	// Create a new Loop instance
	var wg sync.WaitGroup
	el := New()
	wg.Add(1)
	stopCh := make(chan struct{})
	go func() {
		defer wg.Done()
		el.Run(stopCh)
	}()
	defer wg.Wait()

	closeHandlerCh := make(chan interface{})
	var closeHandlerWg sync.WaitGroup
	closeHandlerWg.Add(1)
	closeHandler := NewReadHandler(reflect.ValueOf(closeHandlerCh), func(data reflect.Value, closed bool) {
		if closed {
			closeHandlerWg.Done()
			return
		}
		require.NotNil(t, data)
		close(closeHandlerCh)
	})
	handlerCh := make(chan interface{})
	var closedWg sync.WaitGroup
	closedWg.Add(1)
	handler := NewReadHandler(reflect.ValueOf(handlerCh), func(data reflect.Value, closed bool) {
		if closed {
			closedWg.Done()
			return
		}
		require.NotNil(t, data)
	})
	el.Add(closeHandler, handler)
	handlerCh <- "test data"
	close(handlerCh)
	closedWg.Wait()

	closeHandlerCh <- "test data"
	closeHandlerWg.Wait()

	close(stopCh)
	el.Remove(handler)
	el.Remove(closeHandler)
}

func TestEventLoopAddThreadSafety(t *testing.T) {
	loop := New()
	var wg sync.WaitGroup
	taskCount := 100 // Number of tasks to be added concurrently

	stopCh := make(chan struct{})
	// Start the event loop and allow some time for processing
	go loop.Run(stopCh)

	// Wait for the event loop to start
	startedCh := make(chan struct{})
	wg.Add(1)
	loop.Add(NewReadHandler(reflect.ValueOf(startedCh), func(_ reflect.Value, closed bool) {
		require.True(t, closed, "Channel should be closed")
		wg.Done()
	}))
	close(startedCh)
	wg.Wait()

	wg.Add(taskCount)
	for i := 0; i < taskCount; i++ {
		go func(_ int) {
			ch := make(chan struct{})
			loop.Add(NewReadHandler(reflect.ValueOf(ch), func(_ reflect.Value, closed bool) {
				// Simulate task processing
				require.True(t, closed, "Channel should be closed")
				wg.Done()
			}))
			close(ch)
		}(i)
	}

	wg.Wait() // Wait for all goroutines to finish

	// Verify that the event loop processed all tasks
	require.Equal(t, 0, len(*loop.handlers.Load()), "All tasks should be processed")
}

func TestEventLoopRemoveThreadSafety(t *testing.T) {
	loop := New()
	var wg sync.WaitGroup
	taskCount := 100 // Number of tasks to be added concurrently

	stopCh := make(chan struct{})
	// Start the event loop and allow some time for processing
	go loop.Run(stopCh)

	// Wait for the event loop to start
	startedCh := make(chan struct{})
	wg.Add(1)
	loop.Add(NewReadHandler(reflect.ValueOf(startedCh), func(_ reflect.Value, closed bool) {
		require.True(t, closed, "Channel should be closed")
		wg.Done()
	}))
	close(startedCh)
	wg.Wait()

	wg.Add(taskCount * 2) // Twice the taskCount because of add and remove operations
	for i := 0; i < taskCount; i++ {
		go func() {
			defer wg.Done()
			// Simulate adding a task
			ch := make(chan struct{})
			loop.Add(NewReadHandler(reflect.ValueOf(ch), func(reflect.Value, bool) {
				require.Fail(t, "Task should not be processed")
			}))
			go func() {
				defer wg.Done()
				// Simulate removing a task
				loop.RemoveByChannels(reflect.ValueOf(ch))
			}()
		}()
	}
	wg.Wait() // Wait for all goroutines to finish

	// Verify that the event loop removed all tasks
	require.Equal(t, 0, len(*loop.handlers.Load()), "All tasks should be removed")
}

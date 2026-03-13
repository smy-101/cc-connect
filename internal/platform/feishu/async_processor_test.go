package feishu

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestAsyncEventProcessing(t *testing.T) {
	t.Run("start and stop processor", func(t *testing.T) {
		processor := NewAsyncEventProcessor(DefaultAsyncEventProcessorConfig())

		ctx := context.Background()
		if err := processor.Start(ctx); err != nil {
			t.Errorf("Start() failed: %v", err)
		}

		// Start again should fail
		if err := processor.Start(ctx); err == nil {
			t.Error("second Start() should fail")
		}

		if err := processor.Stop(); err != nil {
			t.Errorf("Stop() failed: %v", err)
		}

		// Stop again should succeed
		if err := processor.Stop(); err != nil {
			t.Errorf("second Stop() should succeed: %v", err)
		}
	})

	t.Run("submit and process event", func(t *testing.T) {
		processor := NewAsyncEventProcessor(DefaultAsyncEventProcessorConfig())

		var processedCount atomic.Int32
		processor.SetHandler(func(ctx context.Context, event *MessageReceiveEvent) error {
			processedCount.Add(1)
			return nil
		})

		ctx := context.Background()
		_ = processor.Start(ctx)
		defer processor.Stop()

		event := &MessageReceiveEvent{
			EventID:   "test_event",
			MessageID: "test_msg",
		}

		err := processor.Submit(ctx, event)
		if err != nil {
			t.Errorf("Submit() failed: %v", err)
		}

		// Wait for processing
		time.Sleep(50 * time.Millisecond)

		if processedCount.Load() != 1 {
			t.Errorf("expected 1 event processed, got %d", processedCount.Load())
		}
	})

	t.Run("submit returns quickly for ACK", func(t *testing.T) {
		processor := NewAsyncEventProcessor(DefaultAsyncEventProcessorConfig())

		// Handler that takes a bit but not too long
		processor.SetHandler(func(ctx context.Context, event *MessageReceiveEvent) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})

		ctx := context.Background()
		_ = processor.Start(ctx)
		defer processor.Stop()

		event := &MessageReceiveEvent{
			EventID:   "test_event",
			MessageID: "test_msg",
		}

		start := time.Now()
		err := processor.Submit(ctx, event)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("Submit() failed: %v", err)
		}

		// Submit should return within 100ms (even if handler takes longer)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Submit() took %v, should return within 100ms", elapsed)
		}

		// Give time for the handler to complete before Stop()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("handler error doesn't affect other events", func(t *testing.T) {
		processor := NewAsyncEventProcessor(DefaultAsyncEventProcessorConfig())

		var processedCount atomic.Int32
		callCount := atomic.Int32{}

		processor.SetHandler(func(ctx context.Context, event *MessageReceiveEvent) error {
			count := callCount.Add(1)
			if count == 1 {
				// First call returns error
				return context.DeadlineExceeded
			}
			processedCount.Add(1)
			return nil
		})

		ctx := context.Background()
		_ = processor.Start(ctx)
		defer processor.Stop()

		// Submit two events
		for i := range 2 {
			event := &MessageReceiveEvent{
				EventID:   "test_event",
				MessageID: "test_msg",
			}
			_ = processor.Submit(ctx, event)
			_ = i // Avoid unused variable warning
		}

		// Wait for processing
		time.Sleep(50 * time.Millisecond)

		// Both events should be processed (even if first one errored)
		if processedCount.Load() != 1 {
			t.Errorf("expected 1 successful event, got %d", processedCount.Load())
		}
	})

	t.Run("drain events on stop", func(t *testing.T) {
		processor := NewAsyncEventProcessor(AsyncEventProcessorConfig{
			BufferSize: 10,
		})

		var processedCount atomic.Int32
		processor.SetHandler(func(ctx context.Context, event *MessageReceiveEvent) error {
			processedCount.Add(1)
			return nil
		})

		ctx := context.Background()
		_ = processor.Start(ctx)

		// Submit multiple events
		for range 5 {
			event := &MessageReceiveEvent{
				EventID:   "test_event",
				MessageID: "test_msg",
			}
			_ = processor.Submit(ctx, event)
		}

		// Stop should drain remaining events
		_ = processor.Stop()

		// All events should be processed
		if processedCount.Load() != 5 {
			t.Errorf("expected 5 events processed, got %d", processedCount.Load())
		}
	})

	t.Run("submit to full channel fails", func(t *testing.T) {
		processor := NewAsyncEventProcessor(AsyncEventProcessorConfig{
			BufferSize: 2,
		})

		// Block the processor to fill up the channel
		blockChan := make(chan struct{})
		processor.SetHandler(func(ctx context.Context, event *MessageReceiveEvent) error {
			<-blockChan // Block until test completes
			return nil
		})

		ctx := context.Background()
		_ = processor.Start(ctx)
		defer processor.Stop()
		defer close(blockChan) // Unblock at end of test

		// Fill the buffer (+1 in progress)
		for range 3 {
			event := &MessageReceiveEvent{
				EventID:   "test_event",
				MessageID: "test_msg",
			}
			_ = processor.Submit(ctx, event)
		}

		// Next submit should fail (channel full)
		event := &MessageReceiveEvent{
			EventID:   "overflow_event",
			MessageID: "overflow_msg",
		}
		err := processor.Submit(ctx, event)
		if err == nil {
			t.Error("Submit() to full channel should fail")
		}
	})
}

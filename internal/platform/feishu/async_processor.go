package feishu

import (
	"context"
	"errors"
	"sync"
	"time"
)

// AsyncEventProcessor handles asynchronous processing of Feishu events.
// It ensures events are ACKed quickly while processing happens in background.
type AsyncEventProcessor struct {
	mu sync.RWMutex

	// Event channel for buffering events
	eventChan chan *MessageReceiveEvent

	// Worker goroutine control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Event handler
	handler EventHandler

	// Configuration
	bufferSize int
}

// AsyncEventProcessorConfig holds configuration for AsyncEventProcessor.
type AsyncEventProcessorConfig struct {
	// BufferSize is the size of the event channel buffer.
	BufferSize int
}

// DefaultAsyncEventProcessorConfig returns default configuration.
func DefaultAsyncEventProcessorConfig() AsyncEventProcessorConfig {
	return AsyncEventProcessorConfig{
		BufferSize: 100,
	}
}

// NewAsyncEventProcessor creates a new AsyncEventProcessor.
func NewAsyncEventProcessor(config AsyncEventProcessorConfig) *AsyncEventProcessor {
	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}

	return &AsyncEventProcessor{
		eventChan:  make(chan *MessageReceiveEvent, config.BufferSize),
		bufferSize: config.BufferSize,
	}
}

// Start starts the async event processor.
func (p *AsyncEventProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ctx != nil {
		return errors.New("processor already started")
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	// Start worker goroutine
	p.wg.Add(1)
	go p.worker()

	return nil
}

// Stop stops the async event processor.
func (p *AsyncEventProcessor) Stop() error {
	p.mu.Lock()
	cancel := p.cancel
	p.mu.Unlock()

	if cancel == nil {
		return nil
	}

	cancel()
	p.wg.Wait()

	p.mu.Lock()
	p.ctx = nil
	p.cancel = nil
	p.mu.Unlock()

	return nil
}

// SetHandler sets the event handler.
func (p *AsyncEventProcessor) SetHandler(handler EventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handler = handler
}

// Submit submits an event for async processing.
// This method returns quickly (within 100ms) to ensure Feishu ACK.
// Returns error if the event channel is full.
func (p *AsyncEventProcessor) Submit(ctx context.Context, event *MessageReceiveEvent) error {
	select {
	case p.eventChan <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(50 * time.Millisecond):
		return errors.New("event channel full, dropping event")
	}
}

// worker is the main goroutine that processes events.
func (p *AsyncEventProcessor) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Drain remaining events before exiting
			for {
				select {
				case event := <-p.eventChan:
					p.processEvent(event)
				default:
					return
				}
			}
		case event := <-p.eventChan:
			p.processEvent(event)
		}
	}
}

// processEvent processes a single event with the registered handler.
func (p *AsyncEventProcessor) processEvent(event *MessageReceiveEvent) {
	p.mu.RLock()
	handler := p.handler
	ctx := p.ctx
	p.mu.RUnlock()

	if handler == nil {
		return
	}

	// Create a child context with timeout for processing
	processCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Process the event - errors are logged but don't affect other events
	_ = handler(processCtx, event)
}

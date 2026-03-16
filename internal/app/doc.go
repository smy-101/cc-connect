// Package app provides the application integration layer for cc-connect.
//
// The app package is responsible for:
//   - Component initialization: Creating and wiring together Router, Agent, Feishu Adapter, and Command Executor
//   - Lifecycle management: Starting, stopping, and graceful shutdown of all components
//   - Message handler registration: Registering text and command message handlers with the Router
//   - Error handling: Capturing panics, handling timeouts, and providing user-friendly error messages
//
// Architecture Overview:
//
//	┌─────────────────────────────────────────────────────────┐
//	│                         App                              │
//	│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐ │
//	│  │ Router  │  │  Agent  │  │ Feishu  │  │  Executor   │ │
//	│  │         │  │         │  │ Adapter │  │  (command)  │ │
//	│  └────┬────┘  └────┬────┘  └────┬────┘  └──────┬──────┘ │
//	│       │            │            │               │        │
//	│       └────────────┴────────────┴───────────────┘        │
//	│                          │                                │
//	│                   HandlerContext                          │
//	│                    └── ReplySender                        │
//	└─────────────────────────────────────────────────────────┘
//
// Key Types:
//   - App: The main application struct that manages all components
//   - HandlerContext: Context passed to message handlers containing message, session, and reply sender
//   - ReplySender: Interface for sending replies back to the message source
//   - Handler: Function signature for message handlers using HandlerContext
//
// Usage:
//
//	config := &core.AppConfig{...}
//	app, err := app.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if err := app.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Wait for shutdown signal
//	app.WaitForShutdown()
package app

// Package feishu provides a platform adapter for Feishu (飞书) messaging platform.
//
// # Overview
//
// This package implements the WebSocket long connection client for Feishu,
// enabling cc-connect to receive and send messages without requiring a public IP.
//
// # Architecture
//
// The package consists of the following components:
//
//   - FeishuClient: Interface defining the client contract
//   - feishuSDKClient: SDK-based implementation of FeishuClient
//   - MockFeishuClient: Mock implementation for testing
//   - MessageConverter: Converts between Feishu and unified message formats
//   - EventParser: Parses im.message.receive_v1 events
//   - Sender: Sends messages to Feishu API
//
// # Usage
//
// Basic usage:
//
//	client := feishu.NewSDKClient(appID, appSecret)
//	client.OnEvent(func(ctx context.Context, event *feishu.MessageReceiveEvent) error {
//	    // Handle the event
//	    return nil
//	})
//
//	if err := client.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Testing
//
// Use MockFeishuClient for unit testing:
//
//	mockClient := feishu.NewMockClient()
//	mockClient.SimulateMessageEvent(testEvent)
package feishu

package feishu

import (
	"errors"
	"strings"

	"github.com/smy-101/cc-connect/internal/core"
)

var (
	ErrMissingAction      = errors.New("card callback missing action")
	ErrMissingActionValue = errors.New("card callback action missing value")
	ErrInvalidAction      = errors.New("invalid action format")
)

// ParseCardCallback parses a Feishu card callback into a unified message.
// The callback data format is from Feishu's card interaction callback.
func ParseCardCallback(data map[string]any) (*core.Message, error) {
	if data == nil {
		return nil, ErrMissingAction
	}

	// Extract action value
	actionValue, err := extractActionValue(data)
	if err != nil {
		return nil, err
	}

	// Parse the action string to a command
	cmd, err := parseActionToCommand(actionValue)
	if err != nil {
		return nil, err
	}

	// Extract user and message context
	msg := &core.Message{
		Type:    core.MessageTypeCommand,
		Content: cmd,
	}

	if openID, ok := data["open_id"].(string); ok {
		msg.UserID = openID
	}
	if msgID, ok := data["open_message_id"].(string); ok {
		msg.ChannelID = msgID
	}

	return msg, nil
}

// extractActionValue extracts the action value from the callback data.
func extractActionValue(data map[string]any) (string, error) {
	action, ok := data["action"].(map[string]any)
	if !ok {
		return "", ErrMissingAction
	}

	value, ok := action["value"].(map[string]any)
	if !ok {
		return "", ErrMissingActionValue
	}

	actionStr, ok := value["action"].(string)
	if !ok || actionStr == "" {
		return "", ErrMissingActionValue
	}

	return actionStr, nil
}

// parseActionToCommand converts an action string to a slash command.
// Action format: "prefix:requestID:extra" or "prefix:requestID"
// Command format: "/prefix requestID extra"
func parseActionToCommand(action string) (string, error) {
	if action == "" {
		return "", ErrInvalidAction
	}

	parts := strings.SplitN(action, ":", 3)
	if len(parts) < 2 {
		return "", ErrInvalidAction
	}

	prefix := parts[0]
	requestID := parts[1]

	// Map action prefixes to commands
	var cmd string
	switch prefix {
	case "perm":
		if len(parts) < 3 {
			return "", ErrInvalidAction
		}
		actionType := parts[1] // "allow" or "deny"
		requestID = parts[2]
		cmd = "/" + actionType + " " + requestID
	case "ans":
		if len(parts) < 3 {
			return "", ErrInvalidAction
		}
		requestID = parts[1]
		answer := parts[2]
		cmd = "/answer " + requestID + " " + answer
	default:
		// Generic fallback: /prefix requestID extra
		if len(parts) >= 3 {
			cmd = "/" + prefix + " " + requestID + " " + parts[2]
		} else {
			cmd = "/" + prefix + " " + requestID
		}
	}

	return cmd, nil
}

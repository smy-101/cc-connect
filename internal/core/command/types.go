package command

import "errors"

// Common errors for command execution
var (
	// ErrEmptyCommand is returned when an empty command is executed
	ErrEmptyCommand = errors.New("empty command")
	// ErrUnknownCommand is returned when an unknown command is executed
	ErrUnknownCommand = errors.New("unknown command")
)

// Command represents a parsed slash command.
// For example, "/mode yolo" becomes Command{Name: "mode", Args: ["yolo"]}
type Command struct {
	// Name is the command name without the leading slash (e.g., "mode", "help")
	Name string
	// Args is the list of arguments passed to the command
	Args []string
}

// IsEmpty returns true if the command has no name (empty/invalid command).
func (c Command) IsEmpty() bool {
	return c.Name == ""
}

// CommandResult represents the result of executing a command.
type CommandResult struct {
	// Message is the human-readable result message to send back to the user
	Message string
	// Error indicates if the command execution failed
	Error error
}

// IsError returns true if the command execution resulted in an error.
func (r CommandResult) IsError() bool {
	return r.Error != nil
}

package command

import "strings"

// IsCommand checks if the given text is a slash command.
// A command must start with '/' (no leading whitespace) and have at least one character after the slash.
func IsCommand(text string) bool {
	// Must have at least 2 characters: '/' plus at least one command character
	return len(text) >= 2 && text[0] == '/'
}

// Parse parses a command string into a Command struct.
// It removes the leading '/' and splits the rest by whitespace.
// If the input is not a valid command, returns an empty Command.
func Parse(text string) Command {
	// Check for empty or non-command input
	if len(text) < 2 || text[0] != '/' {
		return Command{}
	}

	// Remove leading '/' and split by whitespace
	parts := strings.Fields(text[1:])
	if len(parts) == 0 {
		return Command{}
	}

	return Command{
		Name: parts[0],
		Args: parts[1:],
	}
}

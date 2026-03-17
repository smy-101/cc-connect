package command

import (
	"strings"
)

// IsCommand checks if the given text is a slash command.
// A command must start with '/' (no leading whitespace) and have at least one character after the slash.
func IsCommand(text string) bool {
	// Must have at least 2 characters: '/' plus at least one command character
	return len(text) >= 2 && text[0] == '/'
}

// Parse parses a command string into a Command struct.
// It removes the leading '/' and splits the rest by whitespace.
// Supports:
//   - Long flags: --flag, --flag=value
//   - Short flags: -f, -f value
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

	name := parts[0]
	args := make([]string, 0) // Initialize as empty slice, not nil
	flags := make(map[string]string)

	// Parse remaining parts for args and flags
	i := 1
	for i < len(parts) {
		part := parts[i]

		if strings.HasPrefix(part, "--") {
			// Long flag
			flagPart := part[2:]
			if flagPart == "" {
				// Just "--", skip it
				i++
				continue
			}

			// Check for --flag=value format
			if idx := strings.Index(flagPart, "="); idx >= 0 {
				flagName := flagPart[:idx]
				flagValue := flagPart[idx+1:]
				flags[flagName] = flagValue
			} else {
				// --flag format, check if next part is a value
				if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
					flags[flagPart] = parts[i+1]
					i++
				} else {
					flags[flagPart] = "true"
				}
			}
		} else if strings.HasPrefix(part, "-") && len(part) > 1 {
			// Short flag
			flagName := part[1:]

			// Check if next part is a value (not starting with -)
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
				flags[flagName] = parts[i+1]
				i++
			} else {
				flags[flagName] = "true"
			}
		} else {
			// Regular argument
			args = append(args, part)
		}
		i++
	}

	return Command{
		Name:  name,
		Args:  args,
		Flags: flags,
	}
}

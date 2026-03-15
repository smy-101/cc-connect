package claudecode

import (
	"fmt"

	"github.com/smy-101/cc-connect/internal/agent"
)

// modeAliases maps user-friendly aliases to canonical permission modes
var modeAliases = map[string]agent.PermissionMode{
	"default":           agent.PermissionModeDefault,
	"edit":              agent.PermissionModeAcceptEdits,
	"acceptEdits":       agent.PermissionModeAcceptEdits,
	"plan":              agent.PermissionModePlan,
	"yolo":              agent.PermissionModeBypassPermissions,
	"bypassPermissions": agent.PermissionModeBypassPermissions,
}

// modeDescriptions provides human-readable descriptions for permission modes
var modeDescriptions = map[agent.PermissionMode]string{
	agent.PermissionModeDefault:           "All tools require approval",
	agent.PermissionModeAcceptEdits:       "Edit tools are auto-approved",
	agent.PermissionModePlan:              "Read-only tools are auto-approved",
	agent.PermissionModeBypassPermissions: "All tools are auto-approved",
}

// ParsePermissionMode parses a string into a PermissionMode.
// It supports both canonical names and aliases.
func ParsePermissionMode(s string) (agent.PermissionMode, error) {
	if s == "" {
		return "", fmt.Errorf("empty permission mode")
	}

	mode, ok := modeAliases[s]
	if !ok {
		return "", fmt.Errorf("invalid permission mode: %q", s)
	}
	return mode, nil
}

// PermissionModeToCLIArg converts a PermissionMode to the CLI argument format.
func PermissionModeToCLIArg(mode agent.PermissionMode) string {
	return string(mode)
}

// IsValidPermissionMode checks if the given string is a valid permission mode or alias.
func IsValidPermissionMode(s string) bool {
	_, ok := modeAliases[s]
	return ok
}

// PermissionModeDescription returns a human-readable description of the permission mode.
// Returns empty string for invalid modes.
func PermissionModeDescription(mode agent.PermissionMode) string {
	return modeDescriptions[mode]
}

// AllPermissionModes returns all valid permission mode strings (including aliases).
func AllPermissionModes() []string {
	modes := make([]string, 0, len(modeAliases))
	for mode := range modeAliases {
		modes = append(modes, mode)
	}
	return modes
}

// CanonicalPermissionModes returns only the canonical permission modes (no aliases).
func CanonicalPermissionModes() []agent.PermissionMode {
	return []agent.PermissionMode{
		agent.PermissionModeDefault,
		agent.PermissionModeAcceptEdits,
		agent.PermissionModePlan,
		agent.PermissionModeBypassPermissions,
	}
}

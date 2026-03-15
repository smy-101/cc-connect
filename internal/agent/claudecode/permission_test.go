package claudecode

import (
	"testing"

	"github.com/smy-101/cc-connect/internal/agent"
)

// TestPermissionModeMapping tests permission mode mapping and aliases
func TestPermissionModeMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected agent.PermissionMode
		wantErr  bool
	}{
		{"default", agent.PermissionModeDefault, false},
		{"edit", agent.PermissionModeAcceptEdits, false},
		{"acceptEdits", agent.PermissionModeAcceptEdits, false},
		{"plan", agent.PermissionModePlan, false},
		{"yolo", agent.PermissionModeBypassPermissions, false},
		{"bypassPermissions", agent.PermissionModeBypassPermissions, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := ParsePermissionMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePermissionMode(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePermissionMode(%q) unexpected error: %v", tt.input, err)
				return
			}
			if mode != tt.expected {
				t.Errorf("ParsePermissionMode(%q) = %v, want %v", tt.input, mode, tt.expected)
			}
		})
	}
}

// TestPermissionModeToCLIArg tests converting permission modes to CLI arguments
func TestPermissionModeToCLIArg(t *testing.T) {
	tests := []struct {
		mode     agent.PermissionMode
		expected string
	}{
		{agent.PermissionModeDefault, "default"},
		{agent.PermissionModeAcceptEdits, "acceptEdits"},
		{agent.PermissionModePlan, "plan"},
		{agent.PermissionModeBypassPermissions, "bypassPermissions"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			arg := PermissionModeToCLIArg(tt.mode)
			if arg != tt.expected {
				t.Errorf("PermissionModeToCLIArg(%v) = %v, want %v", tt.mode, arg, tt.expected)
			}
		})
	}
}

// TestPermissionModeAliases tests all aliases are correctly resolved
func TestPermissionModeAliases(t *testing.T) {
	// Test edit alias
	mode, err := ParsePermissionMode("edit")
	if err != nil {
		t.Fatalf("ParsePermissionMode(edit) error: %v", err)
	}
	if mode != agent.PermissionModeAcceptEdits {
		t.Errorf("edit should map to acceptEdits, got %v", mode)
	}

	// Test yolo alias
	mode, err = ParsePermissionMode("yolo")
	if err != nil {
		t.Fatalf("ParsePermissionMode(yolo) error: %v", err)
	}
	if mode != agent.PermissionModeBypassPermissions {
		t.Errorf("yolo should map to bypassPermissions, got %v", mode)
	}
}

// TestPermissionModeValid tests IsValidPermissionMode function
func TestPermissionModeValid(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"default", true},
		{"edit", true},
		{"yolo", true},
		{"plan", true},
		{"acceptEdits", true},
		{"bypassPermissions", true},
		{"invalid", false},
		{"", false},
		{"random", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			valid := IsValidPermissionMode(tt.input)
			if valid != tt.expected {
				t.Errorf("IsValidPermissionMode(%q) = %v, want %v", tt.input, valid, tt.expected)
			}
		})
	}
}

// TestPermissionModeDescription tests getting descriptions for modes
func TestPermissionModeDescription(t *testing.T) {
	tests := []struct {
		mode     agent.PermissionMode
		hasDesc  bool
	}{
		{agent.PermissionModeDefault, true},
		{agent.PermissionModeAcceptEdits, true},
		{agent.PermissionModePlan, true},
		{agent.PermissionModeBypassPermissions, true},
		{agent.PermissionMode("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			desc := PermissionModeDescription(tt.mode)
			hasDesc := desc != ""
			if hasDesc != tt.hasDesc {
				t.Errorf("PermissionModeDescription(%v) returned desc=%q, hasDesc=%v, want %v",
					tt.mode, desc, hasDesc, tt.hasDesc)
			}
		})
	}
}

// TestCanonicalPermissionModes tests CanonicalPermissionModes function
func TestCanonicalPermissionModes(t *testing.T) {
	modes := CanonicalPermissionModes()

	if len(modes) != 4 {
		t.Errorf("CanonicalPermissionModes() returned %d modes, want 4", len(modes))
	}

	// Check that all canonical modes are present
	expected := map[agent.PermissionMode]bool{
		agent.PermissionModeDefault:           false,
		agent.PermissionModeAcceptEdits:       false,
		agent.PermissionModePlan:              false,
		agent.PermissionModeBypassPermissions: false,
	}

	for _, mode := range modes {
		if _, ok := expected[mode]; ok {
			expected[mode] = true
		}
	}

	for mode, found := range expected {
		if !found {
			t.Errorf("CanonicalPermissionModes() missing mode %v", mode)
		}
	}
}

// TestAllPermissionModes tests AllPermissionModes function
func TestAllPermissionModes(t *testing.T) {
	modes := AllPermissionModes()

	// Should include aliases too
	if len(modes) < 4 {
		t.Errorf("AllPermissionModes() returned %d modes, want at least 4", len(modes))
	}

	// Check that default is included
	found := false
	for _, mode := range modes {
		if mode == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllPermissionModes() should include 'default'")
	}
}

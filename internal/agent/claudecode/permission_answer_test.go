package claudecode

import (
	"testing"
)

// TestSendPermissionResponseAnswer tests handling answer responses
func TestSendPermissionResponseAnswer(t *testing.T) {
	tests := []struct {
		name           string
		result         PermissionResult
		expectedBehavior string
		expectAnswers  bool
	}{
		{
			name: "allow behavior",
			result: PermissionResult{
				Behavior: "allow",
			},
			expectedBehavior: "allow",
			expectAnswers: false,
		},
		{
			name: "deny behavior",
			result: PermissionResult{
				Behavior: "deny",
			},
			expectedBehavior: "deny",
			expectAnswers: false,
		},
		{
			name: "answer behavior",
			result: PermissionResult{
				Behavior: "answer:PostgreSQL",
			},
			expectedBehavior: "allow",
			expectAnswers: true,
		},
		{
			name: "answer with complex value",
			result: PermissionResult{
				Behavior: "answer:Use Docker for development",
			},
			expectedBehavior: "allow",
			expectAnswers: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			behavior, updatedInput := parsePermissionBehavior(tt.result.Behavior)

			if behavior != tt.expectedBehavior {
				t.Errorf("expected behavior %q, got %q", tt.expectedBehavior, behavior)
			}

			if tt.expectAnswers {
				if updatedInput == nil {
					t.Error("expected updatedInput to be set")
					return
				}
				answers, ok := updatedInput["answers"].([]string)
				if !ok {
					t.Error("expected answers to be []string")
					return
				}
				if len(answers) == 0 {
					t.Error("expected at least one answer")
				}
			}
		})
	}
}

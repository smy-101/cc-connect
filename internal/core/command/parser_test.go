package command

import (
	"errors"
	"reflect"
	"testing"
)

func TestIsCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid commands
		{
			name:     "simple command",
			input:    "/mode",
			expected: true,
		},
		{
			name:     "command with argument",
			input:    "/mode yolo",
			expected: true,
		},
		{
			name:     "command with multiple arguments",
			input:    "/new my-session",
			expected: true,
		},
		{
			name:     "single slash",
			input:    "/",
			expected: false, // Must have content after slash
		},
		// Invalid commands
		{
			name:     "plain text",
			input:    "hello world",
			expected: false,
		},
		{
			name:     "leading space before slash",
			input:    " /mode",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "text starting with letter s",
			input:    "slash",
			expected: false,
		},
		{
			name:     "multiline with slash at start of second line",
			input:    "hello\n/mode",
			expected: false,
		},
		{
			name:     "tab before slash",
			input:    "\t/mode",
			expected: false,
		},
		{
			name:     "command with extra spaces between name and args",
			input:    "/mode   yolo",
			expected: true,
		},
		{
			name:     "help command",
			input:    "/help",
			expected: true,
		},
		{
			name:     "new command",
			input:    "/new",
			expected: true,
		},
		{
			name:     "list command",
			input:    "/list",
			expected: true,
		},
		{
			name:     "stop command",
			input:    "/stop",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCommand(tt.input)
			if result != tt.expected {
				t.Errorf("IsCommand(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedName   string
		expectedArgs   []string
		expectedEmpty  bool
	}{
		{
			name:          "help command no args",
			input:         "/help",
			expectedName:  "help",
			expectedArgs:  []string{},
			expectedEmpty: false,
		},
		{
			name:          "mode command with single arg",
			input:         "/mode yolo",
			expectedName:  "mode",
			expectedArgs:  []string{"yolo"},
			expectedEmpty: false,
		},
		{
			name:          "new command with name arg",
			input:         "/new my-session",
			expectedName:  "new",
			expectedArgs:  []string{"my-session"},
			expectedEmpty: false,
		},
		{
			name:          "mode command with multiple extra spaces",
			input:         "/mode   yolo",
			expectedName:  "mode",
			expectedArgs:  []string{"yolo"},
			expectedEmpty: false,
		},
		{
			name:          "list command no args",
			input:         "/list",
			expectedName:  "list",
			expectedArgs:  []string{},
			expectedEmpty: false,
		},
		{
			name:          "stop command no args",
			input:         "/stop",
			expectedName:  "stop",
			expectedArgs:  []string{},
			expectedEmpty: false,
		},
		{
			name:          "single slash only",
			input:         "/",
			expectedName:  "",
			expectedArgs:  nil,
			expectedEmpty: true,
		},
		{
			name:          "command with trailing space",
			input:         "/help ",
			expectedName:  "help",
			expectedArgs:  []string{},
			expectedEmpty: false,
		},
		{
			name:          "command with leading and trailing spaces",
			input:         "/mode yolo ",
			expectedName:  "mode",
			expectedArgs:  []string{"yolo"},
			expectedEmpty: false,
		},
		{
			name:          "command with multiple args",
			input:         "/new session name",
			expectedName:  "new",
			expectedArgs:  []string{"session", "name"},
			expectedEmpty: false,
		},
		{
			name:          "empty string",
			input:         "",
			expectedName:  "",
			expectedArgs:  nil,
			expectedEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Parse(tt.input)

			if cmd.Name != tt.expectedName {
				t.Errorf("Parse(%q).Name = %q, want %q", tt.input, cmd.Name, tt.expectedName)
			}

			if !reflect.DeepEqual(cmd.Args, tt.expectedArgs) {
				t.Errorf("Parse(%q).Args = %v, want %v", tt.input, cmd.Args, tt.expectedArgs)
			}

			if cmd.IsEmpty() != tt.expectedEmpty {
				t.Errorf("Parse(%q).IsEmpty() = %v, want %v", tt.input, cmd.IsEmpty(), tt.expectedEmpty)
			}
		})
	}
}

func TestCommandResultIsError(t *testing.T) {
	tests := []struct {
		name     string
		result   CommandResult
		expected bool
	}{
		{
			name: "no error",
			result: CommandResult{
				Message: "success",
				Error:   nil,
			},
			expected: false,
		},
		{
			name: "with error",
			result: CommandResult{
				Message: "failed",
				Error:   errors.New("something went wrong"),
			},
			expected: true,
		},
		{
			name: "empty message no error",
			result: CommandResult{
				Message: "",
				Error:   nil,
			},
			expected: false,
		},
		{
			name: "empty message with error",
			result: CommandResult{
				Message: "",
				Error:   errors.New("error"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.IsError() != tt.expected {
				t.Errorf("CommandResult.IsError() = %v, want %v", tt.result.IsError(), tt.expected)
			}
		})
	}
}

func TestCommandIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		cmd      Command
		expected bool
	}{
		{
			name:     "empty command",
			cmd:      Command{},
			expected: true,
		},
		{
			name: "empty name with args",
			cmd: Command{
				Name: "",
				Args: []string{"arg"},
			},
			expected: true,
		},
		{
			name: "name only",
			cmd: Command{
				Name: "help",
				Args: nil,
			},
			expected: false,
		},
		{
			name: "name with args",
			cmd: Command{
				Name: "mode",
				Args: []string{"yolo"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.IsEmpty() != tt.expected {
				t.Errorf("Command.IsEmpty() = %v, want %v", tt.cmd.IsEmpty(), tt.expected)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedName  string
		expectedArgs  []string
		expectedFlags map[string]string
	}{
		{
			name:          "no flags",
			input:         "/project backend",
			expectedName:  "project",
			expectedArgs:  []string{"backend"},
			expectedFlags: map[string]string{},
		},
		{
			name:          "long flag --keep",
			input:         "/project backend --keep",
			expectedName:  "project",
			expectedArgs:  []string{"backend"},
			expectedFlags: map[string]string{"keep": "true"},
		},
		{
			name:          "short flag -k",
			input:         "/project backend -k",
			expectedName:  "project",
			expectedArgs:  []string{"backend"},
			expectedFlags: map[string]string{"k": "true"},
		},
		{
			name:          "multiple flags",
			input:         "/project backend --keep --verbose",
			expectedName:  "project",
			expectedArgs:  []string{"backend"},
			expectedFlags: map[string]string{"keep": "true", "verbose": "true"},
		},
		{
			name:          "flag with value",
			input:         "/cmd --name myvalue",
			expectedName:  "cmd",
			expectedArgs:  []string{},
			expectedFlags: map[string]string{"name": "myvalue"},
		},
		{
			name:          "arg before flag",
			input:         "/project myproject --keep",
			expectedName:  "project",
			expectedArgs:  []string{"myproject"},
			expectedFlags: map[string]string{"keep": "true"},
		},
		{
			name:          "multiple args and flags",
			input:         "/cmd arg1 arg2 --flag1 --flag2 value2",
			expectedName:  "cmd",
			expectedArgs:  []string{"arg1", "arg2"},
			expectedFlags: map[string]string{"flag1": "true", "flag2": "value2"},
		},
		{
			name:          "short and long flags mixed",
			input:         "/cmd -k --verbose",
			expectedName:  "cmd",
			expectedArgs:  []string{},
			expectedFlags: map[string]string{"k": "true", "verbose": "true"},
		},
		{
			name:          "flag with equals sign",
			input:         "/cmd --name=value",
			expectedName:  "cmd",
			expectedArgs:  []string{},
			expectedFlags: map[string]string{"name": "value"},
		},
		{
			name:          "short flag with value",
			input:         "/cmd -n value",
			expectedName:  "cmd",
			expectedArgs:  []string{},
			expectedFlags: map[string]string{"n": "value"},
		},
		{
			name:          "empty command with flags",
			input:         "/cmd --flag",
			expectedName:  "cmd",
			expectedArgs:  []string{},
			expectedFlags: map[string]string{"flag": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Parse(tt.input)

			if cmd.Name != tt.expectedName {
				t.Errorf("Parse(%q).Name = %q, want %q", tt.input, cmd.Name, tt.expectedName)
			}

			if !reflect.DeepEqual(cmd.Args, tt.expectedArgs) {
				t.Errorf("Parse(%q).Args = %v, want %v", tt.input, cmd.Args, tt.expectedArgs)
			}

			if cmd.Flags == nil {
				cmd.Flags = make(map[string]string)
			}
			if !reflect.DeepEqual(cmd.Flags, tt.expectedFlags) {
				t.Errorf("Parse(%q).Flags = %v, want %v", tt.input, cmd.Flags, tt.expectedFlags)
			}
		})
	}
}

func TestCommandHasFlag(t *testing.T) {
	tests := []struct {
		name     string
		cmd      Command
		flag     string
		expected bool
	}{
		{
			name: "flag exists",
			cmd: Command{
				Name:  "project",
				Args:  []string{"backend"},
				Flags: map[string]string{"keep": "true"},
			},
			flag:     "keep",
			expected: true,
		},
		{
			name: "flag does not exist",
			cmd: Command{
				Name:  "project",
				Args:  []string{"backend"},
				Flags: map[string]string{},
			},
			flag:     "keep",
			expected: false,
		},
		{
			name: "nil flags",
			cmd: Command{
				Name: "project",
				Args: []string{"backend"},
			},
			flag:     "keep",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.HasFlag(tt.flag) != tt.expected {
				t.Errorf("Command.HasFlag(%q) = %v, want %v", tt.flag, tt.cmd.HasFlag(tt.flag), tt.expected)
			}
		})
	}
}

func TestCommandFlagValue(t *testing.T) {
	tests := []struct {
		name         string
		cmd          Command
		flag         string
		expectedVal  string
		expectedBool bool
	}{
		{
			name: "flag with value",
			cmd: Command{
				Name:  "cmd",
				Flags: map[string]string{"name": "myvalue"},
			},
			flag:         "name",
			expectedVal:  "myvalue",
			expectedBool: true,
		},
		{
			name: "flag with true",
			cmd: Command{
				Name:  "cmd",
				Flags: map[string]string{"keep": "true"},
			},
			flag:         "keep",
			expectedVal:  "true",
			expectedBool: true,
		},
		{
			name: "flag does not exist",
			cmd: Command{
				Name:  "cmd",
				Flags: map[string]string{},
			},
			flag:         "keep",
			expectedVal:  "",
			expectedBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := tt.cmd.FlagValue(tt.flag)
			if val != tt.expectedVal {
				t.Errorf("Command.FlagValue(%q) = %q, want %q", tt.flag, val, tt.expectedVal)
			}
			if ok != tt.expectedBool {
				t.Errorf("Command.FlagValue(%q) ok = %v, want %v", tt.flag, ok, tt.expectedBool)
			}
		})
	}
}

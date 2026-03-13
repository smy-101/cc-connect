package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ============================================================================
// 2.1 AppConfig 基础字段测试
// ============================================================================

func TestAppConfigBasicFields(t *testing.T) {
	config := &AppConfig{
		LogLevel:       "debug",
		DefaultProject: "my-project",
		Projects:       []ProjectConfig{},
	}

	if config.LogLevel != "debug" {
		t.Errorf("expected LogLevel to be 'debug', got '%s'", config.LogLevel)
	}

	if config.DefaultProject != "my-project" {
		t.Errorf("expected DefaultProject to be 'my-project', got '%s'", config.DefaultProject)
	}

	if len(config.Projects) != 0 {
		t.Errorf("expected empty Projects slice, got %d items", len(config.Projects))
	}
}

func TestAppConfigDefaultLogLevel(t *testing.T) {
	config := &AppConfig{}

	// 默认日志级别应为 "info"
	if config.LogLevel != "" {
		// 字段默认为空，加载器应设置默认值
		t.Errorf("expected empty LogLevel by default, got '%s'", config.LogLevel)
	}
}

// ============================================================================
// 2.3 ProjectConfig 及子配置测试
// ============================================================================

func TestProjectConfigBasicFields(t *testing.T) {
	project := ProjectConfig{
		Name:        "test-project",
		Description: "A test project",
		WorkingDir:  "/home/user/project",
	}

	if project.Name != "test-project" {
		t.Errorf("expected Name to be 'test-project', got '%s'", project.Name)
	}

	if project.Description != "A test project" {
		t.Errorf("expected Description to be 'A test project', got '%s'", project.Description)
	}

	if project.WorkingDir != "/home/user/project" {
		t.Errorf("expected WorkingDir to be '/home/user/project', got '%s'", project.WorkingDir)
	}
}

func TestFeishuConfig(t *testing.T) {
	feishu := FeishuConfig{
		AppID:     "cli_test123",
		AppSecret: "secret123",
		Enabled:   true,
	}

	if feishu.AppID != "cli_test123" {
		t.Errorf("expected AppID to be 'cli_test123', got '%s'", feishu.AppID)
	}

	if feishu.AppSecret != "secret123" {
		t.Errorf("expected AppSecret to be 'secret123', got '%s'", feishu.AppSecret)
	}

	if !feishu.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestFeishuConfigDefaultDisabled(t *testing.T) {
	feishu := FeishuConfig{}

	if feishu.Enabled {
		t.Error("expected Enabled to be false by default")
	}
}

func TestClaudeCodeConfig(t *testing.T) {
	cc := ClaudeCodeConfig{
		DefaultPermissionMode: "yolo",
		Enabled:               true,
	}

	if cc.DefaultPermissionMode != "yolo" {
		t.Errorf("expected DefaultPermissionMode to be 'yolo', got '%s'", cc.DefaultPermissionMode)
	}

	if !cc.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestClaudeCodeConfigDefaultPermissionMode(t *testing.T) {
	cc := ClaudeCodeConfig{}

	// 默认权限模式应为空，验证器或加载器应设置默认值 "default"
	if cc.DefaultPermissionMode != "" {
		t.Errorf("expected empty DefaultPermissionMode by default, got '%s'", cc.DefaultPermissionMode)
	}
}

func TestProjectWithFeishuAndClaudeCode(t *testing.T) {
	project := ProjectConfig{
		Name:       "full-project",
		WorkingDir: "/workspace",
		Feishu: FeishuConfig{
			AppID:     "cli_xxx",
			AppSecret: "secret",
			Enabled:   true,
		},
		ClaudeCode: ClaudeCodeConfig{
			DefaultPermissionMode: "plan",
			Enabled:               true,
		},
	}

	if project.Feishu.AppID != "cli_xxx" {
		t.Errorf("expected Feishu.AppID to be 'cli_xxx', got '%s'", project.Feishu.AppID)
	}

	if project.ClaudeCode.DefaultPermissionMode != "plan" {
		t.Errorf("expected ClaudeCode.DefaultPermissionMode to be 'plan', got '%s'", project.ClaudeCode.DefaultPermissionMode)
	}
}

// ============================================================================
// 3.1 TOML 加载器测试 - 有效配置
// ============================================================================

func TestLoadValidConfig(t *testing.T) {
	// 创建临时测试文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
log_level = "debug"
default_project = "my-project"

[[projects]]
name = "my-project"
description = "Test project"
working_dir = "/home/user/project"

[projects.feishu]
app_id = "cli_test123"
app_secret = "secret123"
enabled = true

[projects.claude_code]
default_permission_mode = "yolo"
enabled = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewTOMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if config.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", config.LogLevel)
	}

	if config.DefaultProject != "my-project" {
		t.Errorf("expected DefaultProject 'my-project', got '%s'", config.DefaultProject)
	}

	if len(config.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(config.Projects))
	}

	project := config.Projects[0]
	if project.Name != "my-project" {
		t.Errorf("expected project name 'my-project', got '%s'", project.Name)
	}

	if project.WorkingDir != "/home/user/project" {
		t.Errorf("expected WorkingDir '/home/user/project', got '%s'", project.WorkingDir)
	}

	if project.Feishu.AppID != "cli_test123" {
		t.Errorf("expected Feishu.AppID 'cli_test123', got '%s'", project.Feishu.AppID)
	}

	if !project.Feishu.Enabled {
		t.Error("expected Feishu.Enabled to be true")
	}

	if project.ClaudeCode.DefaultPermissionMode != "yolo" {
		t.Errorf("expected ClaudeCode.DefaultPermissionMode 'yolo', got '%s'", project.ClaudeCode.DefaultPermissionMode)
	}
}

func TestLoadMultipleProjects(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
[[projects]]
name = "project-1"
working_dir = "/path/1"

[[projects]]
name = "project-2"
working_dir = "/path/2"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewTOMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(config.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(config.Projects))
	}

	if config.Projects[0].Name != "project-1" {
		t.Errorf("expected first project name 'project-1', got '%s'", config.Projects[0].Name)
	}

	if config.Projects[1].Name != "project-2" {
		t.Errorf("expected second project name 'project-2', got '%s'", config.Projects[1].Name)
	}
}

// ============================================================================
// 3.3 TOML 加载器测试 - 错误场景
// ============================================================================

func TestLoadNonExistentFile(t *testing.T) {
	loader := NewTOMLLoader()
	_, err := loader.Load("/nonexistent/path/config.toml")

	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got: %v", err)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.toml")

	content := `
[invalid
missing bracket
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewTOMLLoader()
	_, err := loader.Load(configPath)

	if err == nil {
		t.Fatal("expected error for invalid TOML, got nil")
	}

	if !errors.Is(err, ErrConfigParseFailed) {
		t.Errorf("expected ErrConfigParseFailed, got: %v", err)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.toml")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewTOMLLoader()
	config, err := loader.Load(configPath)

	if err != nil {
		t.Fatalf("expected no error for empty file, got: %v", err)
	}

	// 空文件应返回默认配置
	if config == nil {
		t.Fatal("expected config, got nil")
	}

	// 验证默认值
	if config.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got '%s'", config.LogLevel)
	}
}

// ============================================================================
// 4.1 环境变量展开测试
// ============================================================================

func TestExpandEnvVar(t *testing.T) {
	t.Setenv("TEST_APP_ID", "cli_from_env")
	t.Setenv("TEST_SECRET", "secret_from_env")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple expansion",
			input:    "${TEST_APP_ID}",
			expected: "cli_from_env",
		},
		{
			name:     "literal value unchanged",
			input:    "cli_literal_value",
			expected: "cli_literal_value",
		},
		{
			name:     "mixed syntax",
			input:    "prefix_${TEST_SECRET}_suffix",
			expected: "prefix_secret_from_env_suffix",
		},
		{
			name:     "missing env var becomes empty",
			input:    "${NONEXISTENT_VAR}",
			expected: "",
		},
		{
			name:     "multiple env vars",
			input:    "${TEST_APP_ID}_${TEST_SECRET}",
			expected: "cli_from_env_secret_from_env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoadWithEnvVarExpansion(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "cli_env_id")
	t.Setenv("FEISHU_APP_SECRET", "env_secret_12345")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
[[projects]]
name = "test-project"
working_dir = "/workspace"

[projects.feishu]
app_id = "${FEISHU_APP_ID}"
app_secret = "${FEISHU_APP_SECRET}"
enabled = true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	loader := NewTOMLLoader()
	config, err := loader.Load(configPath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(config.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(config.Projects))
	}

	project := config.Projects[0]
	if project.Feishu.AppID != "cli_env_id" {
		t.Errorf("expected expanded AppID 'cli_env_id', got '%s'", project.Feishu.AppID)
	}

	if project.Feishu.AppSecret != "env_secret_12345" {
		t.Errorf("expected expanded AppSecret 'env_secret_12345', got '%s'", project.Feishu.AppSecret)
	}
}

// ============================================================================
// 5.1 验证测试：必填字段
// ============================================================================

func TestValidateMissingRequiredFields(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name      string
		config    *AppConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "missing project name",
			config: &AppConfig{
				Projects: []ProjectConfig{
					{WorkingDir: "/workspace"},
				},
			},
			expectErr: true,
			errMsg:    "name",
		},
		{
			name: "missing working_dir",
			config: &AppConfig{
				Projects: []ProjectConfig{
					{Name: "test-project"},
				},
			},
			expectErr: true,
			errMsg:    "working_dir",
		},
		{
			name: "all required fields present",
			config: &AppConfig{
				Projects: []ProjectConfig{
					{Name: "test-project", WorkingDir: "/tmp"},
				},
			},
			expectErr: false,
		},
		{
			name:      "empty projects list is valid",
			config:    &AppConfig{Projects: []ProjectConfig{}},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.config)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if !errors.Is(err, ErrValidationFailed) {
					t.Errorf("expected ErrValidationFailed, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}
		})
	}
}

// ============================================================================
// 5.3 验证测试：值有效性
// ============================================================================

func TestValidateInvalidPermissionMode(t *testing.T) {
	validator := NewConfigValidator()

	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "workspace")
	os.Mkdir(workingDir, 0755)

	config := &AppConfig{
		Projects: []ProjectConfig{
			{
				Name:       "test",
				WorkingDir: workingDir,
				ClaudeCode: ClaudeCodeConfig{
					DefaultPermissionMode: "invalid_mode",
				},
			},
		},
	}

	err := validator.Validate(config)
	if err == nil {
		t.Error("expected error for invalid permission mode")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got: %v", err)
	}
}

func TestValidateValidPermissionModes(t *testing.T) {
	validator := NewConfigValidator()

	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "workspace")
	os.Mkdir(workingDir, 0755)

	validModes := []string{"default", "edit", "acceptEdits", "plan", "yolo", "bypassPermissions", ""}

	for _, mode := range validModes {
		t.Run("mode_"+mode, func(t *testing.T) {
			config := &AppConfig{
				Projects: []ProjectConfig{
					{
						Name:       "test",
						WorkingDir: workingDir,
						ClaudeCode: ClaudeCodeConfig{
							DefaultPermissionMode: mode,
						},
					},
				},
			}

			err := validator.Validate(config)
			if err != nil {
				t.Errorf("expected no error for mode %q, got: %v", mode, err)
			}
		})
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {
	validator := NewConfigValidator()

	config := &AppConfig{
		LogLevel: "invalid_level",
		Projects: []ProjectConfig{
			{Name: "test", WorkingDir: "/tmp"},
		},
	}

	err := validator.Validate(config)
	if err == nil {
		t.Error("expected error for invalid log level")
	}
}

func TestValidateValidLogLevels(t *testing.T) {
	validator := NewConfigValidator()

	validLevels := []string{"debug", "info", "warn", "error", ""}

	for _, level := range validLevels {
		t.Run("level_"+level, func(t *testing.T) {
			config := &AppConfig{
				LogLevel: level,
				Projects: []ProjectConfig{
					{Name: "test", WorkingDir: "/tmp"},
				},
			}

			err := validator.Validate(config)
			if err != nil {
				t.Errorf("expected no error for level %q, got: %v", level, err)
			}
		})
	}
}

func TestValidateNonExistentWorkingDir(t *testing.T) {
	validator := NewConfigValidator()

	config := &AppConfig{
		Projects: []ProjectConfig{
			{
				Name:       "test",
				WorkingDir: "/nonexistent/path/that/does/not/exist",
			},
		},
	}

	err := validator.Validate(config)
	if err == nil {
		t.Error("expected error for non-existent working directory")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got: %v", err)
	}
}

// ============================================================================
// 5.5 验证测试：多项目约束
// ============================================================================

func TestValidateDuplicateProjectNames(t *testing.T) {
	validator := NewConfigValidator()

	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "dir1")
	dir2 := filepath.Join(tmpDir, "dir2")
	os.Mkdir(dir1, 0755)
	os.Mkdir(dir2, 0755)

	config := &AppConfig{
		Projects: []ProjectConfig{
			{Name: "duplicate", WorkingDir: dir1},
			{Name: "duplicate", WorkingDir: dir2},
		},
	}

	err := validator.Validate(config)
	if err == nil {
		t.Error("expected error for duplicate project names")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got: %v", err)
	}
	if err != nil && !contains(err.Error(), "duplicate project name") {
		t.Errorf("expected error to contain 'duplicate project name', got: %v", err)
	}
}

func TestValidateDefaultProjectNotFound(t *testing.T) {
	validator := NewConfigValidator()

	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "workspace")
	os.Mkdir(workingDir, 0755)

	config := &AppConfig{
		DefaultProject: "nonexistent",
		Projects: []ProjectConfig{
			{Name: "other-project", WorkingDir: workingDir},
		},
	}

	err := validator.Validate(config)
	if err == nil {
		t.Error("expected error for default project not found")
	}
	if !errors.Is(err, ErrValidationFailed) {
		t.Errorf("expected ErrValidationFailed, got: %v", err)
	}
	if err != nil && !contains(err.Error(), "default_project not found") {
		t.Errorf("expected error to contain 'default_project not found', got: %v", err)
	}
}

// ============================================================================
// 5.7 验证测试：警告收集
// ============================================================================

func TestValidateWarningsForEmptyEnvVars(t *testing.T) {
	validator := NewConfigValidator()

	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "workspace")
	os.Mkdir(workingDir, 0755)

	// 当环境变量展开后为空字符串时，应产生警告
	config := &AppConfig{
		Projects: []ProjectConfig{
			{
				Name:       "test",
				WorkingDir: workingDir,
				Feishu: FeishuConfig{
					AppID:     "", // 空的 AppID 应产生警告
					AppSecret: "", // 空的 AppSecret 应产生警告
					Enabled:   true,
				},
			},
		},
	}

	warnings := validator.Warnings(config)
	if len(warnings) == 0 {
		t.Error("expected warnings for empty Feishu credentials when enabled")
	}
}

func TestValidateNoWarningsForDisabledFeishu(t *testing.T) {
	validator := NewConfigValidator()

	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "workspace")
	os.Mkdir(workingDir, 0755)

	config := &AppConfig{
		Projects: []ProjectConfig{
			{
				Name:       "test",
				WorkingDir: workingDir,
				Feishu: FeishuConfig{
					AppID:     "",
					AppSecret: "",
					Enabled:   false, // 禁用时不应产生警告
				},
			},
		},
	}

	warnings := validator.Warnings(config)
	for _, w := range warnings {
		if contains(w, "Feishu") || contains(w, "AppID") || contains(w, "AppSecret") {
			t.Errorf("unexpected warning for disabled Feishu: %s", w)
		}
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// 6.1 摘要测试：基本信息
// ============================================================================

func TestSummaryBasic(t *testing.T) {
	config := &AppConfig{
		LogLevel:       "debug",
		DefaultProject: "main-project",
		Projects: []ProjectConfig{
			{Name: "main-project", WorkingDir: "/workspace/main"},
			{Name: "side-project", WorkingDir: "/workspace/side"},
		},
	}

	summary := Summary(config)

	if summary == nil {
		t.Fatal("expected summary, got nil")
	}

	if summary.TotalProjects != 2 {
		t.Errorf("expected TotalProjects 2, got %d", summary.TotalProjects)
	}

	if summary.DefaultProject != "main-project" {
		t.Errorf("expected DefaultProject 'main-project', got '%s'", summary.DefaultProject)
	}

	if summary.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", summary.LogLevel)
	}
}

func TestSummaryProjectList(t *testing.T) {
	config := &AppConfig{
		Projects: []ProjectConfig{
			{Name: "project-a", WorkingDir: "/a", Feishu: FeishuConfig{Enabled: true}},
			{Name: "project-b", WorkingDir: "/b", ClaudeCode: ClaudeCodeConfig{Enabled: true}},
		},
	}

	summary := Summary(config)

	if len(summary.Projects) != 2 {
		t.Fatalf("expected 2 project summaries, got %d", len(summary.Projects))
	}

	if summary.Projects[0].Name != "project-a" {
		t.Errorf("expected first project name 'project-a', got '%s'", summary.Projects[0].Name)
	}

	if !summary.Projects[0].FeishuEnabled {
		t.Error("expected FeishuEnabled to be true for project-a")
	}

	if !summary.Projects[1].ClaudeCodeEnabled {
		t.Error("expected ClaudeCodeEnabled to be true for project-b")
	}
}

// ============================================================================
// 6.3 摘要测试：敏感值脱敏
// ============================================================================

func TestSummaryMasking(t *testing.T) {
	config := &AppConfig{
		Projects: []ProjectConfig{
			{
				Name:       "test",
				WorkingDir: "/tmp",
				Feishu: FeishuConfig{
					AppID:     "cli_abc123xyz",
					AppSecret: "secret_long_value_here",
					Enabled:   true,
				},
			},
		},
	}

	summary := Summary(config)

	if len(summary.Projects) != 1 {
		t.Fatalf("expected 1 project summary, got %d", len(summary.Projects))
	}

	// AppID 应显示前 4 个字符 + ***
	maskedAppID := summary.Projects[0].MaskedAppID
	if maskedAppID != "cli_***" {
		t.Errorf("expected MaskedAppID 'cli_***', got '%s'", maskedAppID)
	}

	// AppSecret 应显示前 4 个字符 + ***
	maskedSecret := summary.Projects[0].MaskedAppSecret
	if maskedSecret != "secr***" {
		t.Errorf("expected MaskedAppSecret 'secr***', got '%s'", maskedSecret)
	}
}

func TestSummaryMaskingShortValues(t *testing.T) {
	config := &AppConfig{
		Projects: []ProjectConfig{
			{
				Name:       "test",
				WorkingDir: "/tmp",
				Feishu: FeishuConfig{
					AppID:     "ab", // 少于 4 个字符
					AppSecret: "",
					Enabled:   true,
				},
			},
		},
	}

	summary := Summary(config)

	// 短值应完全显示 + ***
	maskedAppID := summary.Projects[0].MaskedAppID
	if maskedAppID != "ab***" {
		t.Errorf("expected MaskedAppID 'ab***', got '%s'", maskedAppID)
	}

	// 空值应显示 ***
	maskedSecret := summary.Projects[0].MaskedAppSecret
	if maskedSecret != "***" {
		t.Errorf("expected MaskedAppSecret '***', got '%s'", maskedSecret)
	}
}

// ============================================================================
// 6.5 摘要测试：格式化输出
// ============================================================================

func TestSummaryString(t *testing.T) {
	config := &AppConfig{
		LogLevel:       "info",
		DefaultProject: "main",
		Projects: []ProjectConfig{
			{
				Name:       "main",
				WorkingDir: "/workspace",
				Feishu:     FeishuConfig{AppID: "cli_test123", Enabled: true},
				ClaudeCode: ClaudeCodeConfig{Enabled: true},
			},
		},
	}

	summary := Summary(config)
	output := summary.String()

	if output == "" {
		t.Error("expected non-empty string output")
	}

	// 验证输出包含关键信息
	if !contains(output, "main") {
		t.Error("expected output to contain project name 'main'")
	}
	if !contains(output, "info") {
		t.Error("expected output to contain log level 'info'")
	}
}

// ============================================================================
// 7.1 项目查找测试
// ============================================================================

func TestGetProject(t *testing.T) {
	config := &AppConfig{
		Projects: []ProjectConfig{
			{Name: "project-a", WorkingDir: "/path/a"},
			{Name: "project-b", WorkingDir: "/path/b"},
		},
	}

	t.Run("find existing project", func(t *testing.T) {
		project, found := config.GetProject("project-a")
		if !found {
			t.Error("expected to find project-a")
		}
		if project == nil {
			t.Fatal("expected project, got nil")
		}
		if project.Name != "project-a" {
			t.Errorf("expected name 'project-a', got '%s'", project.Name)
		}
		if project.WorkingDir != "/path/a" {
			t.Errorf("expected WorkingDir '/path/a', got '%s'", project.WorkingDir)
		}
	})

	t.Run("find non-existent project", func(t *testing.T) {
		project, found := config.GetProject("nonexistent")
		if found {
			t.Error("expected not to find nonexistent project")
		}
		if project != nil {
			t.Errorf("expected nil project, got %v", project)
		}
	})
}

func TestGetDefaultProject(t *testing.T) {
	t.Run("default project exists", func(t *testing.T) {
		config := &AppConfig{
			DefaultProject: "main",
			Projects: []ProjectConfig{
				{Name: "main", WorkingDir: "/main"},
				{Name: "other", WorkingDir: "/other"},
			},
		}

		project, found := config.GetDefaultProject()
		if !found {
			t.Error("expected to find default project")
		}
		if project == nil {
			t.Fatal("expected project, got nil")
		}
		if project.Name != "main" {
			t.Errorf("expected name 'main', got '%s'", project.Name)
		}
	})

	t.Run("no default project set", func(t *testing.T) {
		config := &AppConfig{
			Projects: []ProjectConfig{
				{Name: "only-project", WorkingDir: "/path"},
			},
		}

		project, found := config.GetDefaultProject()
		if found {
			t.Error("expected not to find default project when not set")
		}
		if project != nil {
			t.Errorf("expected nil project, got %v", project)
		}
	})

	t.Run("default project not in list", func(t *testing.T) {
		config := &AppConfig{
			DefaultProject: "missing",
			Projects: []ProjectConfig{
				{Name: "other", WorkingDir: "/other"},
			},
		}

		project, found := config.GetDefaultProject()
		if found {
			t.Error("expected not to find default project when missing from list")
		}
		if project != nil {
			t.Errorf("expected nil project, got %v", project)
		}
	})
}

// ============================================================================
// 8.1 会话配置覆盖测试
// ============================================================================

func TestGetSessionConfig(t *testing.T) {
	t.Run("no session override uses defaults", func(t *testing.T) {
		project := &ProjectConfig{
			Name:       "test",
			WorkingDir: "/tmp",
		}

		sessionConfig := project.GetSessionConfig()
		defaultConfig := DefaultSessionConfig()

		if sessionConfig.ActiveTTL != defaultConfig.ActiveTTL {
			t.Errorf("expected ActiveTTL %v, got %v", defaultConfig.ActiveTTL, sessionConfig.ActiveTTL)
		}
		if sessionConfig.ArchivedTTL != defaultConfig.ArchivedTTL {
			t.Errorf("expected ArchivedTTL %v, got %v", defaultConfig.ArchivedTTL, sessionConfig.ArchivedTTL)
		}
		if sessionConfig.CleanupInterval != defaultConfig.CleanupInterval {
			t.Errorf("expected CleanupInterval %v, got %v", defaultConfig.CleanupInterval, sessionConfig.CleanupInterval)
		}
	})

	t.Run("partial override", func(t *testing.T) {
		customTTL := 1 * time.Hour
		project := &ProjectConfig{
			Name:       "test",
			WorkingDir: "/tmp",
			Session: &SessionConfigOpt{
				ActiveTTL: &timeDuration{Duration: customTTL},
			},
		}

		sessionConfig := project.GetSessionConfig()
		defaultConfig := DefaultSessionConfig()

		if sessionConfig.ActiveTTL != customTTL {
			t.Errorf("expected ActiveTTL %v, got %v", customTTL, sessionConfig.ActiveTTL)
		}
		// 未覆盖的值应使用默认值
		if sessionConfig.ArchivedTTL != defaultConfig.ArchivedTTL {
			t.Errorf("expected ArchivedTTL %v, got %v", defaultConfig.ArchivedTTL, sessionConfig.ArchivedTTL)
		}
	})

	t.Run("full override", func(t *testing.T) {
		customActive := 2 * time.Hour
		customArchived := 48 * time.Hour
		customCleanup := 10 * time.Minute

		project := &ProjectConfig{
			Name:       "test",
			WorkingDir: "/tmp",
			Session: &SessionConfigOpt{
				ActiveTTL:       &timeDuration{Duration: customActive},
				ArchivedTTL:     &timeDuration{Duration: customArchived},
				CleanupInterval: &timeDuration{Duration: customCleanup},
			},
		}

		sessionConfig := project.GetSessionConfig()

		if sessionConfig.ActiveTTL != customActive {
			t.Errorf("expected ActiveTTL %v, got %v", customActive, sessionConfig.ActiveTTL)
		}
		if sessionConfig.ArchivedTTL != customArchived {
			t.Errorf("expected ArchivedTTL %v, got %v", customArchived, sessionConfig.ArchivedTTL)
		}
		if sessionConfig.CleanupInterval != customCleanup {
			t.Errorf("expected CleanupInterval %v, got %v", customCleanup, sessionConfig.CleanupInterval)
		}
	})
}

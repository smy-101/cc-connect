// Package core provides core domain types and interfaces for cc-connect.
// This file contains configuration structures and interfaces.
package core

import (
	"errors"
	"fmt"
	"os"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

// 配置相关错误定义
var (
	// ErrConfigNotFound 配置文件不存在
	ErrConfigNotFound = errors.New("config file not found")
	// ErrConfigParseFailed 配置解析失败
	ErrConfigParseFailed = errors.New("failed to parse config file")
	// ErrValidationFailed 配置验证失败
	ErrValidationFailed = errors.New("config validation failed")
	// ErrMissingRequired 必填字段缺失
	ErrMissingRequired = errors.New("missing required field")
	// ErrInvalidValue 无效的字段值
	ErrInvalidValue = errors.New("invalid field value")
	// ErrDuplicateProject 项目名称重复
	ErrDuplicateProject = errors.New("duplicate project name")
	// ErrDefaultProjectNotFound 默认项目不存在
	ErrDefaultProjectNotFound = errors.New("default project not found")
)

// AppConfig 应用级配置
type AppConfig struct {
	LogLevel       string          `toml:"log_level"`
	DefaultProject string          `toml:"default_project"`
	Projects       []ProjectConfig `toml:"projects"`
}

// ProjectConfig 项目级配置
type ProjectConfig struct {
	Name        string            `toml:"name"`
	Description string            `toml:"description"`
	WorkingDir  string            `toml:"working_dir"`
	Feishu      FeishuConfig      `toml:"feishu"`
	ClaudeCode  ClaudeCodeConfig  `toml:"claude_code"`
	Session     *SessionConfigOpt `toml:"session"`
}

// FeishuConfig 飞书平台配置
type FeishuConfig struct {
	AppID     string `toml:"app_id"`
	AppSecret string `toml:"app_secret"`
	Enabled   bool   `toml:"enabled"`
}

// ClaudeCodeConfig Claude Code 代理配置
type ClaudeCodeConfig struct {
	DefaultPermissionMode string `toml:"default_permission_mode"`
	Enabled               bool   `toml:"enabled"`
}

// SessionConfigOpt 可选的会话配置（用于项目级覆盖）
type SessionConfigOpt struct {
	ActiveTTL       *timeDuration `toml:"active_ttl"`
	ArchivedTTL     *timeDuration `toml:"archived_ttl"`
	CleanupInterval *timeDuration `toml:"cleanup_interval"`
}

// timeDuration 包装器用于 TOML 解析
type timeDuration struct {
	Duration time.Duration
}

// ConfigLoader 配置加载器接口
type ConfigLoader interface {
	Load(path string) (*AppConfig, error)
}

// TOMLLoader TOML 配置加载器
type TOMLLoader struct{}

// NewTOMLLoader 创建 TOML 加载器
func NewTOMLLoader() *TOMLLoader {
	return &TOMLLoader{}
}

// Load 从指定路径加载 TOML 配置文件
func (l *TOMLLoader) Load(path string) (*AppConfig, error) {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
	}

	// 读取文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析 TOML
	var config AppConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigParseFailed, err)
	}

	// 设置默认值
	l.setDefaults(&config)

	// 展开环境变量
	l.expandConfigEnvVars(&config)

	return &config, nil
}

// setDefaults 设置配置默认值
func (l *TOMLLoader) setDefaults(config *AppConfig) {
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
}

// expandEnvVars 展开字符串中的环境变量
// 支持 ${VAR} 语法，使用 os.ExpandEnv
func expandEnvVars(s string) string {
	return os.ExpandEnv(s)
}

// expandConfigEnvVars 展开配置中的环境变量
// 对敏感字段进行环境变量展开
func (l *TOMLLoader) expandConfigEnvVars(config *AppConfig) {
	for i := range config.Projects {
		project := &config.Projects[i]
		// 展开飞书配置中的敏感字段
		project.Feishu.AppID = expandEnvVars(project.Feishu.AppID)
		project.Feishu.AppSecret = expandEnvVars(project.Feishu.AppSecret)
	}
}

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	Validate(config *AppConfig) error
	Warnings(config *AppConfig) []string
}

// DefaultConfigValidator 默认配置验证器
type DefaultConfigValidator struct{}

// NewConfigValidator 创建配置验证器
func NewConfigValidator() *DefaultConfigValidator {
	return &DefaultConfigValidator{}
}

// validPermissionModes 有效的权限模式
var validPermissionModes = map[string]bool{
	"default":           true,
	"edit":              true,
	"acceptEdits":       true,
	"plan":              true,
	"yolo":              true,
	"bypassPermissions": true,
	"":                  true, // 空值有效（使用默认）
}

// validLogLevels 有效的日志级别
var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
	"":      true, // 空值有效（使用默认）
}

// Validate 验证配置
func (v *DefaultConfigValidator) Validate(config *AppConfig) error {
	var errors []string

	// 验证日志级别
	if !validLogLevels[config.LogLevel] {
		errors = append(errors, fmt.Sprintf("invalid log_level: %s", config.LogLevel))
	}

	// 验证项目
	projectNames := make(map[string]bool)
	for i, project := range config.Projects {
		// 验证必填字段
		if project.Name == "" {
			errors = append(errors, fmt.Sprintf("projects[%d].name: required field missing", i))
		}
		if project.WorkingDir == "" {
			errors = append(errors, fmt.Sprintf("projects[%d].working_dir: required field missing", i))
		}

		// 验证工作目录存在性
		if project.WorkingDir != "" {
			if _, err := os.Stat(project.WorkingDir); os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("projects[%d].working_dir: directory does not exist: %s", i, project.WorkingDir))
			}
		}

		// 验证权限模式
		if !validPermissionModes[project.ClaudeCode.DefaultPermissionMode] {
			errors = append(errors, fmt.Sprintf("projects[%d].claude_code.default_permission_mode: invalid mode: %s", i, project.ClaudeCode.DefaultPermissionMode))
		}

		// 检查项目名称重复
		if project.Name != "" {
			if projectNames[project.Name] {
				errors = append(errors, fmt.Sprintf("duplicate project name: %s", project.Name))
			}
			projectNames[project.Name] = true
		}
	}

	// 验证默认项目存在
	if config.DefaultProject != "" && !projectNames[config.DefaultProject] {
		errors = append(errors, fmt.Sprintf("default_project not found: %s", config.DefaultProject))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%w: %v", ErrValidationFailed, errors)
	}

	return nil
}

// Warnings 收集配置警告
func (v *DefaultConfigValidator) Warnings(config *AppConfig) []string {
	var warnings []string

	for _, project := range config.Projects {
		// 当飞书启用但凭证为空时产生警告
		if project.Feishu.Enabled {
			if project.Feishu.AppID == "" {
				warnings = append(warnings, fmt.Sprintf("project '%s': Feishu enabled but AppID is empty", project.Name))
			}
			if project.Feishu.AppSecret == "" {
				warnings = append(warnings, fmt.Sprintf("project '%s': Feishu enabled but AppSecret is empty", project.Name))
			}
		}

		// 当 Claude Code 启用但工作目录为空时产生警告
		if project.ClaudeCode.Enabled && project.WorkingDir == "" {
			warnings = append(warnings, fmt.Sprintf("project '%s': Claude Code enabled but WorkingDir is empty", project.Name))
		}
	}

	return warnings
}

// ConfigSummary 配置摘要
type ConfigSummary struct {
	LogLevel        string
	DefaultProject  string
	TotalProjects   int
	FeishuEnabled   int
	ClaudeCodeEnabled int
	Projects        []ProjectSummary
}

// ProjectSummary 项目摘要
type ProjectSummary struct {
	Name               string
	WorkingDir         string
	FeishuEnabled      bool
	ClaudeCodeEnabled  bool
	MaskedAppID        string
	MaskedAppSecret    string
}

// Summary 生成配置摘要
func Summary(config *AppConfig) *ConfigSummary {
	if config == nil {
		return &ConfigSummary{}
	}

	summary := &ConfigSummary{
		LogLevel:       config.LogLevel,
		DefaultProject: config.DefaultProject,
		TotalProjects:  len(config.Projects),
		Projects:       make([]ProjectSummary, 0, len(config.Projects)),
	}

	for _, project := range config.Projects {
		projectSummary := ProjectSummary{
			Name:              project.Name,
			WorkingDir:        project.WorkingDir,
			FeishuEnabled:     project.Feishu.Enabled,
			ClaudeCodeEnabled: project.ClaudeCode.Enabled,
			MaskedAppID:       maskSensitive(project.Feishu.AppID),
			MaskedAppSecret:   maskSensitive(project.Feishu.AppSecret),
		}
		summary.Projects = append(summary.Projects, projectSummary)

		if project.Feishu.Enabled {
			summary.FeishuEnabled++
		}
		if project.ClaudeCode.Enabled {
			summary.ClaudeCodeEnabled++
		}
	}

	return summary
}

// maskSensitive 脱敏敏感值，显示前 4 个字符 + ***
func maskSensitive(value string) string {
	if value == "" {
		return "***"
	}
	if len(value) <= 4 {
		return value + "***"
	}
	return value[:4] + "***"
}

// String 返回摘要的格式化字符串
func (s *ConfigSummary) String() string {
	var result string
	result += fmt.Sprintf("Configuration Summary\n")
	result += fmt.Sprintf("====================\n")
	result += fmt.Sprintf("Log Level: %s\n", s.LogLevel)
	result += fmt.Sprintf("Default Project: %s\n", s.DefaultProject)
	result += fmt.Sprintf("Total Projects: %d\n", s.TotalProjects)
	result += fmt.Sprintf("Feishu Enabled: %d/%d\n", s.FeishuEnabled, s.TotalProjects)
	result += fmt.Sprintf("Claude Code Enabled: %d/%d\n", s.ClaudeCodeEnabled, s.TotalProjects)
	result += "\nProjects:\n"

	for _, p := range s.Projects {
		result += fmt.Sprintf("  - %s\n", p.Name)
		result += fmt.Sprintf("    Working Dir: %s\n", p.WorkingDir)
		result += fmt.Sprintf("    Feishu: enabled=%v, app_id=%s\n", p.FeishuEnabled, p.MaskedAppID)
		result += fmt.Sprintf("    Claude Code: enabled=%v\n", p.ClaudeCodeEnabled)
	}

	return result
}

// GetProject 按名称查找项目配置
// 返回项目的副本和是否找到的标志
func (c *AppConfig) GetProject(name string) (*ProjectConfig, bool) {
	for i := range c.Projects {
		if c.Projects[i].Name == name {
			// 返回副本以避免外部修改
			copy := c.Projects[i]
			return &copy, true
		}
	}
	return nil, false
}

// GetDefaultProject 获取默认项目配置
// 返回默认项目的副本和是否找到的标志
func (c *AppConfig) GetDefaultProject() (*ProjectConfig, bool) {
	if c.DefaultProject == "" {
		return nil, false
	}
	return c.GetProject(c.DefaultProject)
}

// GetSessionConfig 获取项目的会话配置
// 如果项目没有自定义配置，返回默认配置
func (p *ProjectConfig) GetSessionConfig() SessionConfig {
	defaults := DefaultSessionConfig()

	if p.Session == nil {
		return defaults
	}

	// 合并自定义配置和默认值
	config := defaults
	if p.Session.ActiveTTL != nil {
		config.ActiveTTL = p.Session.ActiveTTL.Duration
	}
	if p.Session.ArchivedTTL != nil {
		config.ArchivedTTL = p.Session.ArchivedTTL.Duration
	}
	if p.Session.CleanupInterval != nil {
		config.CleanupInterval = p.Session.CleanupInterval.Duration
	}

	return config
}

# 配置管理规格

## Purpose

配置管理模块负责 cc-connect 应用的配置加载、验证和管理。它提供 TOML 文件解析、环境变量展开、多项目支持和启动摘要报告功能，是连接应用启动和各适配器的配置基础。

## ADDED Requirements

### Requirement: 配置文件加载

系统 SHALL 支持从 TOML 格式文件加载配置。

#### Scenario: 加载有效 TOML 文件
- **WHEN** 调用 `Load("/path/to/config.toml")` 且文件存在且格式正确
- **THEN** 系统返回解析后的 `AppConfig` 结构，包含所有配置信息

#### Scenario: 加载不存在的文件
- **WHEN** 调用 `Load("/path/to/nonexistent.toml")` 且文件不存在
- **THEN** 系统返回 `ErrConfigNotFound` 错误

#### Scenario: 加载格式错误的 TOML
- **WHEN** 调用 `Load("/path/to/invalid.toml")` 且文件内容不是有效 TOML
- **THEN** 系统返回 `ErrConfigParseFailed` 错误，包含解析错误详情

#### Scenario: 加载空文件
- **WHEN** 调用 `Load("/path/to/empty.toml")` 且文件为空
- **THEN** 系统返回使用默认值的 `AppConfig`

### Requirement: 应用级配置

系统 SHALL 支持应用级配置字段。

#### Scenario: 解析日志级别
- **WHEN** 配置文件包含 `log_level = "debug"`
- **THEN** `AppConfig.LogLevel` 被设置为 "debug"

#### Scenario: 解析默认项目
- **WHEN** 配置文件包含 `default_project = "my-project"`
- **THEN** `AppConfig.DefaultProject` 被设置为 "my-project"

#### Scenario: 默认值
- **WHEN** 配置文件未指定 `log_level`
- **THEN** `AppConfig.LogLevel` 默认为 "info"

### Requirement: 项目级配置

系统 SHALL 支持多个项目的独立配置。

#### Scenario: 解析单个项目
- **WHEN** 配置文件包含一个 `[[projects]]` 表
- **THEN** `AppConfig.Projects` 包含一个 `ProjectConfig` 元素

#### Scenario: 解析多个项目
- **WHEN** 配置文件包含多个 `[[projects]]` 表
- **THEN** `AppConfig.Projects` 包含相应数量的 `ProjectConfig` 元素，顺序与文件一致

#### Scenario: 项目必填字段
- **WHEN** 项目配置缺少 `name` 字段
- **THEN** 验证返回 `ErrMissingRequired` 错误，指出 "name" 缺失

#### Scenario: 项目工作目录
- **WHEN** 配置文件包含 `working_dir = "/home/user/project"`
- **THEN** `ProjectConfig.WorkingDir` 被设置为 "/home/user/project"

### Requirement: 飞书平台配置

系统 SHALL 支持飞书平台的配置结构。

#### Scenario: 解析飞书配置
- **WHEN** 配置文件包含 `[projects.feishu]` 表
- **THEN** `ProjectConfig.Feishu` 包含 `AppID`、`AppSecret`、`Enabled` 字段

#### Scenario: 飞书启用状态
- **WHEN** 配置文件包含 `enabled = true`
- **THEN** `FeishuConfig.Enabled` 为 true

#### Scenario: 飞书禁用状态
- **WHEN** 配置文件包含 `enabled = false` 或省略该字段
- **THEN** `FeishuConfig.Enabled` 为 false

### Requirement: Claude Code 代理配置

系统 SHALL 支持 Claude Code 代理的配置结构。

#### Scenario: 解析 Claude Code 配置
- **WHEN** 配置文件包含 `[projects.claude_code]` 表
- **THEN** `ProjectConfig.ClaudeCode` 包含 `DefaultPermissionMode`、`Enabled` 字段

#### Scenario: 默认权限模式
- **WHEN** 配置文件包含 `default_permission_mode = "yolo"`
- **THEN** `ClaudeCodeConfig.DefaultPermissionMode` 为 "yolo"

#### Scenario: 权限模式默认值
- **WHEN** 配置文件未指定 `default_permission_mode`
- **THEN** `ClaudeCodeConfig.DefaultPermissionMode` 默认为 "default"

### Requirement: 环境变量展开

系统 SHALL 支持 `${VAR}` 语法展开环境变量。

#### Scenario: 展开存在的环境变量
- **WHEN** 配置值 `${FEISHU_APP_ID}` 且环境变量 `FEISHU_APP_ID` 值为 "cli_xxx"
- **THEN** 配置值被展开为 "cli_xxx"

#### Scenario: 字面值不展开
- **WHEN** 配置值 `cli_literal_value` 不包含 `${}` 语法
- **THEN** 配置值保持原样 "cli_literal_value"

#### Scenario: 环境变量不存在
- **WHEN** 配置值 `${MISSING_VAR}` 且环境变量 `MISSING_VAR` 不存在
- **THEN** 配置值展开为空字符串，验证器生成警告

#### Scenario: 混合语法
- **WHEN** 配置值 `prefix_${VAR}_suffix` 且 `VAR` 值为 "middle"
- **THEN** 配置值展开为 "prefix_middle_suffix"

### Requirement: 配置验证

系统 SHALL 在加载后验证配置有效性。

#### Scenario: 必填字段验证
- **WHEN** 项目配置缺少必填字段（`name` 或 `working_dir`）
- **THEN** 验证返回错误，指出缺失的字段路径

#### Scenario: 工作目录存在性验证
- **WHEN** 项目的 `working_dir` 指向不存在的目录
- **THEN** 验证返回错误，指出目录不存在

#### Scenario: 权限模式有效性验证
- **WHEN** `default_permission_mode` 不是有效值（default/edit/plan/yolo）
- **THEN** 验证返回错误，指出无效的模式值

#### Scenario: 日志级别有效性验证
- **WHEN** `log_level` 不是有效值（debug/info/warn/error）
- **THEN** 验证返回错误，指出无效的日志级别

#### Scenario: 项目名称重复验证
- **WHEN** 多个项目具有相同的 `name`
- **THEN** 验证返回 `ErrDuplicateProject` 错误

#### Scenario: 默认项目存在性验证
- **WHEN** `default_project` 指向不存在的项目名称
- **THEN** 验证返回 `ErrDefaultProjectNotFound` 错误

### Requirement: 配置摘要报告

系统 SHALL 生成配置摘要报告。

#### Scenario: 基本摘要信息
- **WHEN** 调用 `Summary(config)`
- **THEN** 返回包含所有项目列表、启用状态、关键参数的摘要

#### Scenario: 敏感值脱敏
- **WHEN** 生成摘要且配置包含 App Secret
- **THEN** 敏感值显示为 "cli_***" 格式（仅显示前 4 个字符）

#### Scenario: 警告信息
- **WHEN** 配置存在非阻断问题（如环境变量未设置）
- **THEN** 摘要包含警告列表

#### Scenario: 启用汇总
- **WHEN** 生成摘要
- **THEN** 摘要包含平台和代理的启用统计（如 "2/2 项目启用 Claude Code"）

### Requirement: 项目查找

系统 SHALL 支持按名称查找项目配置。

#### Scenario: 按名称查找存在的项目
- **WHEN** 调用 `config.GetProject("my-project")` 且项目存在
- **THEN** 返回该项目的 `ProjectConfig` 副本和 true

#### Scenario: 按名称查找不存在的项目
- **WHEN** 调用 `config.GetProject("nonexistent")` 且项目不存在
- **THEN** 返回 nil 和 false

#### Scenario: 获取默认项目
- **WHEN** 调用 `config.GetDefaultProject()`
- **THEN** 返回 `default_project` 指定的项目配置

### Requirement: 会话配置覆盖

系统 SHALL 允许项目级配置覆盖默认会话配置。

#### Scenario: 项目级会话配置
- **WHEN** 项目配置包含 `[projects.session]` 表
- **THEN** 该项目的 `SessionConfig` 使用项目配置值

#### Scenario: 会话配置继承默认值
- **WHEN** 项目配置不包含 `[projects.session]` 表
- **THEN** 该项目使用全局默认的 `SessionConfig`

#### Scenario: 部分覆盖
- **WHEN** 项目会话配置仅指定 `active_ttl`
- **THEN** `active_ttl` 使用项目值，`archived_ttl` 使用默认值

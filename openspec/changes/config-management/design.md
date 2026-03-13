# 配置管理系统设计

## Context

cc-connect 当前已完成核心消息系统的三个模块：统一消息结构、消息路由和会话管理。这些模块都已具备自身的配置结构（如 `SessionConfig`），但缺少统一的应用级配置加载和管理机制。

后续的飞书适配器需要 App ID/Secret 凭证，Claude Code 适配器需要工作目录和权限模式配置。在进入阶段 2 之前，需要先建立配置管理的基础设施。

### 当前状态

```
internal/core/
├── message.go      # 消息结构（无外部配置依赖）
├── router.go       # 路由器（无外部配置依赖）
└── session.go      # 会话管理（自带 SessionConfig）
```

### 约束

- 配置必须支持敏感信息通过环境变量传递
- 配置加载不能依赖网络（纯本地文件）
- 必须支持测试友好（可注入、可 mock）

## Goals / Non-Goals

**Goals:**

1. 实现 TOML 配置文件的标准加载流程
2. 支持两层配置结构：应用级 + 项目级
3. 支持 `${VAR}` 环境变量语法展开
4. 提供配置验证和摘要报告功能
5. 建立可测试的配置接口抽象

**Non-Goals:**

1. 不实现热重载（运行时配置更新）
2. 不实现配置加密存储
3. 不实现配置文件自动生成向导
4. 不实现远程配置拉取
5. 不实现多项目的并发运行时管理

## Decisions

### 决策 1: TOML 库选择

**选择**: `github.com/pelletier/go-toml/v2`

**理由**:
- 活跃维护，Go 社区推荐
- 性能优于 BurntSushi/toml
- 支持 TOML v1.0.0 规范
- 提供树形导航能力（未来扩展）

**替代方案**:
- `BurntSushi/toml`: 曾经的标准，但已不再维护
- `encoding/json`: 不支持注释，可读性差

### 决策 2: 配置层次结构

**选择**: 两层结构（应用级 + 项目级）

```
AppConfig
├── LogLevel: string
├── DefaultProject: string
└── Projects: []ProjectConfig

ProjectConfig
├── Name: string (必填)
├── Description: string
├── WorkingDir: string (必填)
├── Feishu: FeishuConfig
├── ClaudeCode: ClaudeCodeConfig
└── Session: SessionConfig (可选覆盖)
```

**理由**:
- 应用级配置管理全局设置
- 项目级配置隔离不同工作空间
- 为后续多项目支持预留结构
- 避免过度扁平化导致的命名冲突

**替代方案**:
- 单层配置：无法支持多项目场景
- 三层配置（应用/项目/环境）：过度设计

### 决策 3: 环境变量处理

**选择**: `${VAR}` 语法 + os.ExpandEnv

**理由**:
- Shell 风格语法，用户熟悉
- Go 标准库原生支持
- 可在配置文件任意位置使用

**处理流程**:
```
1. TOML 解析原始字符串
2. 对敏感字段应用 os.ExpandEnv
3. 验证展开后的值
4. 生成警告（如果引用的环境变量不存在）
```

**替代方案**:
- 专有语法 `{{env.VAR}}`: 需要自定义解析器
- 仅支持特定字段前缀: 不够灵活

### 决策 4: 配置验证策略

**选择**: 加载后全量验证 + 警告收集

**验证类型**:
| 类型 | 处理方式 |
|------|----------|
| 必填字段缺失 | 错误（阻断启动） |
| 工作目录不存在 | 错误（阻断启动） |
| 无效权限模式 | 错误（阻断启动） |
| 环境变量未设置 | 警告（允许启动） |
| 项目名称重复 | 错误（阻断启动） |
| 默认项目不存在 | 错误（阻断启动） |

**理由**:
- 严格验证防止运行时错误
- 警告机制允许灵活配置
- 收集所有问题一次性报告

### 决策 5: 接口设计

**选择**: 小接口组合

```go
// ConfigLoader 配置加载器
type ConfigLoader interface {
    Load(path string) (*AppConfig, error)
}

// ConfigValidator 配置验证器
type ConfigValidator interface {
    Validate(config *AppConfig) error
    Warnings(config *AppConfig) []string
}

// ConfigSummarizer 配置摘要生成器
type ConfigSummarizer interface {
    Summary(config *AppConfig) *ConfigSummary
}
```

**理由**:
- 单一职责，易于测试
- 可独立 mock 各个能力
- 符合 Go 接口设计最佳实践

**替代方案**:
- 大接口包含所有方法: 难以 mock
- 无接口直接使用结构体: 测试困难

## Risks / Trade-offs

### 风险 1: 敏感信息泄露

**风险**: 配置文件可能包含 App Secret 等敏感信息

**缓解**:
- 支持 `${VAR}` 环境变量语法
- 摘要报告对敏感值脱敏（只显示前 4 个字符 + ***）
- 文档建议使用环境变量

### 风险 2: TOML 格式变更

**风险**: 配置格式升级导致旧配置失效

**缓解**:
- 明确的版本兼容性承诺
- 变更时提供迁移指南
- 使用 TOML 的表结构保持扩展性

### 风险 3: 配置复杂度过高

**风险**: 多层配置结构增加学习成本

**缓解**:
- 提供完整示例配置文件
- 清晰的错误信息指出问题位置
- 摘要报告直观展示配置状态

### Trade-off: 不支持热重载

**取舍**: MVP 阶段不支持运行时配置更新

**影响**: 修改配置需要重启应用

**接受理由**: 简化实现，阶段 6 可扩展

## Architecture

### 包结构

```
internal/core/
├── config.go           # 配置结构和接口定义
├── config_loader.go    # TOML 加载实现
├── config_validator.go # 验证逻辑
├── config_summary.go   # 摘要报告生成
├── config_test.go      # 测试
└── session.go          # 现有会话管理（SessionConfig 可复用）
```

### 层次归属

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           配置管理层次                                   │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ 核心域 (internal/core)                                          │   │
│   │                                                                 │   │
│   │   • AppConfig, ProjectConfig   ← 配置数据模型                   │   │
│   │   • ConfigLoader interface     ← 加载抽象                       │   │
│   │   • ConfigValidator interface  ← 验证抽象                       │   │
│   │   • FeishuConfig               ← 平台配置结构（定义在 core）     │   │
│   │   • ClaudeCodeConfig           ← 代理配置结构（定义在 core）     │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│                              ▼                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ 基础设施层 (internal/core)                                      │   │
│   │                                                                 │   │
│   │   • TOMLLoader                 ← TOML 文件加载实现               │   │
│   │   • DefaultValidator           ← 默认验证实现                    │   │
│   │   • SummaryReporter            ← 摘要报告实现                    │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│                              ▼                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ 消费者 (后续阶段)                                               │   │
│   │                                                                 │   │
│   │   • platform/feishu            ← 读取 FeishuConfig              │   │
│   │   • agent/claudecode           ← 读取 ClaudeCodeConfig          │   │
│   │   • tui                        ← 读取 AppConfig 展示             │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 数据流

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  TOML File   │────▶│  TOMLLoader  │────▶│  Validator   │────▶│  AppConfig   │
│  config.toml │     │  Load()      │     │  Validate()  │     │  (内存结构)  │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                            │                    │                     │
                            │                    ▼                     │
                            │           ┌──────────────┐              │
                            │           │   Warnings   │              │
                            │           │  (非阻断)    │              │
                            │           └──────────────┘              │
                            │                    │                     │
                            ▼                    ▼                     ▼
                     ┌─────────────────────────────────────────────────────┐
                     │              SummaryReporter                        │
                     │         生成配置摘要报告                             │
                     └─────────────────────────────────────────────────────┘
```

## Error Handling

### 错误类型

```go
var (
    ErrConfigNotFound      = errors.New("config file not found")
    ErrConfigParseFailed   = errors.New("failed to parse config file")
    ErrValidationFailed    = errors.New("config validation failed")
    ErrMissingRequired     = errors.New("missing required field")
    ErrInvalidValue        = errors.New("invalid field value")
    ErrDuplicateProject    = errors.New("duplicate project name")
    ErrDefaultProjectNotFound = errors.New("default project not found")
)
```

### 错误处理策略

| 错误类型 | 处理方式 | 用户可见信息 |
|----------|----------|--------------|
| 文件不存在 | 返回错误 | "配置文件不存在: /path/to/config.toml" |
| 解析失败 | 返回错误 | "配置解析失败 (行 15): invalid TOML syntax" |
| 必填缺失 | 返回错误 | "projects[0].name: 必填字段缺失" |
| 验证失败 | 返回错误 + 详情 | "配置验证失败:\n- 项目名称重复: my-project\n- 工作目录不存在: /foo/bar" |

## Testability

### 测试策略

1. **单元测试**: 使用内存中的 TOML 字符串，不依赖文件系统
2. **Mock 接口**: ConfigLoader 可替换为返回固定配置
3. **环境变量隔离**: 使用 t.Setenv 在测试中设置环境变量

### 测试用例覆盖

```go
// 加载测试
TestLoadValidConfig
TestLoadNonExistentFile
TestLoadInvalidTOML
TestLoadEmptyFile

// 环境变量测试
TestEnvVarExpansion
TestEnvVarNotFound (警告)
TestLiteralValue

// 验证测试
TestValidateMissingRequired
TestValidateInvalidPermissionMode
TestValidateNonExistentWorkingDir
TestValidateDuplicateProjectName
TestValidateDefaultProjectNotFound

// 摘要测试
TestSummaryGeneration
TestSummaryWithWarnings
TestSensitiveValueMasking

// 多项目测试
TestMultipleProjects
TestProjectLookup
```

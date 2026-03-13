# 配置管理系统实现任务

## 1. 基础设施准备

- [x] 1.1 添加 pelletier/go-toml/v2 依赖到 go.mod
  - **影响**: `go.mod`
  - **验证**: `go mod tidy` 成功

- [x] 1.2 创建配置模块文件结构
  - **影响**: `internal/core/config.go`, `internal/core/config_test.go`
  - **验证**: 文件创建成功，包编译通过

## 2. 配置数据模型 (TDD)

- [x] 2.1 编写配置结构测试：AppConfig 基础字段
  - **影响**: `internal/core/config_test.go`
  - **TDD**: 先写失败测试
  - **验证**: `go test -run TestAppConfig ./internal/core/` 失败（红）

- [x] 2.2 实现 AppConfig 结构体
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现使测试通过
  - **验证**: `go test -run TestAppConfig ./internal/core/` 通过（绿）

- [x] 2.3 编写配置结构测试：ProjectConfig 及子配置
  - **影响**: `internal/core/config_test.go`
  - **TDD**: 先写失败测试
  - **验证**: `go test -run TestProjectConfig ./internal/core/` 失败（红）

- [x] 2.4 实现 ProjectConfig、FeishuConfig、ClaudeCodeConfig 结构体
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现使测试通过
  - **验证**: `go test -run TestProjectConfig ./internal/core/` 通过（绿）

## 3. TOML 加载器 (TDD)

- [x] 3.1 编写加载测试：有效 TOML 文件
  - **影响**: `internal/core/config_test.go`
  - **TDD**: 先写失败测试，使用测试数据目录
  - **验证**: `go test -run TestLoadValidConfig ./internal/core/` 失败（红）

- [x] 3.2 实现 TOMLLoader.Load 方法
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestLoadValidConfig ./internal/core/` 通过（绿）

- [x] 3.3 编写加载测试：错误场景
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: 文件不存在、格式错误、空文件
  - **验证**: `go test -run TestLoadErrors ./internal/core/` 失败（红）

- [x] 3.4 实现错误处理逻辑
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestLoadErrors ./internal/core/` 通过（绿）

## 4. 环境变量展开 (TDD)

- [x] 4.1 编写环境变量展开测试
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: `${VAR}` 展开、字面值不变、混合语法、变量不存在
  - **验证**: `go test -run TestEnvVarExpansion ./internal/core/` 失败（红）

- [x] 4.2 实现环境变量展开函数
  - **影响**: `internal/core/config.go`
  - **TDD**: 使用 `os.ExpandEnv`
  - **验证**: `go test -run TestEnvVarExpansion ./internal/core/` 通过（绿）

- [x] 4.3 集成环境变量展开到加载流程
  - **影响**: `internal/core/config.go`
  - **TDD**: 在加载后自动展开敏感字段
  - **验证**: `go test -run TestLoadWithEnvVar ./internal/core/` 通过

## 5. 配置验证器 (TDD)

- [x] 5.1 编写验证测试：必填字段
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: name 缺失、working_dir 缺失
  - **验证**: `go test -run TestValidateRequired ./internal/core/` 失败（红）

- [x] 5.2 实现必填字段验证
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestValidateRequired ./internal/core/` 通过（绿）

- [x] 5.3 编写验证测试：值有效性
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: 无效权限模式、无效日志级别、工作目录不存在
  - **验证**: `go test -run TestValidateValues ./internal/core/` 失败（红）

- [x] 5.4 实现值有效性验证
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestValidateValues ./internal/core/` 通过（绿）

- [x] 5.5 编写验证测试：多项目约束
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: 项目名称重复、默认项目不存在
  - **验证**: `go test -run TestValidateProjects ./internal/core/` 失败（红）

- [x] 5.6 实现多项目约束验证
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestValidateProjects ./internal/core/` 通过（绿）

- [x] 5.7 编写验证测试：警告收集
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: 环境变量未设置产生警告（非错误）
  - **验证**: `go test -run TestValidateWarnings ./internal/core/` 通过

## 6. 配置摘要报告 (TDD)

- [x] 6.1 编写摘要测试：基本信息
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: 项目列表、启用状态
  - **验证**: `go test -run TestSummaryBasic ./internal/core/` 失败（红）

- [x] 6.2 实现 Summary 函数
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestSummaryBasic ./internal/core/` 通过（绿）

- [x] 6.3 编写摘要测试：敏感值脱敏
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: App Secret 显示为 "cli_***"
  - **验证**: `go test -run TestSummaryMasking ./internal/core/` 失败（红）

- [x] 6.4 实现敏感值脱敏
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestSummaryMasking ./internal/core/` 通过（绿）

- [x] 6.5 编写摘要测试：格式化输出
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: String() 方法返回可读的格式化文本
  - **验证**: `go test -run TestSummaryString ./internal/core/` 通过

## 7. 项目查找功能 (TDD)

- [x] 7.1 编写项目查找测试
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: 按名称查找、获取默认项目
  - **验证**: `go test -run TestGetProject ./internal/core/` 失败（红）

- [x] 7.2 实现 GetProject 和 GetDefaultProject 方法
  - **影响**: `internal/core/config.go`
  - **TDD**: 最小实现
  - **验证**: `go test -run TestGetProject ./internal/core/` 通过（绿）

## 8. 会话配置覆盖 (TDD)

- [x] 8.1 编写会话配置覆盖测试
  - **影响**: `internal/core/config_test.go`
  - **测试场景**: 项目级配置覆盖默认值、部分覆盖
  - **验证**: `go test -run TestSessionConfigOverride ./internal/core/` 失败（红）

- [x] 8.2 实现会话配置合并逻辑
  - **影响**: `internal/core/config.go`
  - **TDD**: 使用现有 SessionConfig 结构
  - **验证**: `go test -run TestSessionConfigOverride ./internal/core/` 通过（绿）

## 9. 重构与文档

- [x] 9.1 代码重构：提取接口到独立文件
  - **影响**: `internal/core/config.go` → 拆分为 `config.go`, `config_loader.go`, `config_validator.go`, `config_summary.go`
  - **验证**: 所有测试仍然通过

- [x] 9.2 添加示例配置文件
  - **影响**: `config.example.toml`
  - **验证**: 示例文件可被正确加载

- [x] 9.3 更新 CLAUDE.md 配置相关文档
  - **影响**: `CLAUDE.md`
  - **验证**: 文档包含配置文件说明
  - **结果**: 完成 ✓

## 10. 最终验证

- [x] 10.1 运行完整测试套件
  - **验证**: `go test ./internal/core/... -v -cover` 覆盖率 ≥ 85%
  - **结果**: 覆盖率 96.8% ✓

- [x] 10.2 运行竞态检测
  - **验证**: `go test ./internal/core/... -race` 通过
  - **结果**: 通过 ✓

- [x] 10.3 运行全部测试
  - **验证**: `go test ./...` 通过
  - **结果**: 通过 ✓

---

## 任务依赖关系

```
1.1 ──► 1.2 ──► 2.x ──► 3.x ──► 4.x ──► 5.x ──► 6.x ──► 7.x ──► 8.x ──► 9.x ──► 10.x
 │                                                                  │
 └──────────────────────────────────────────────────────────────────┘
                              (可并行)
```

**可并行任务**:
- 6.x (摘要) 和 7.x (项目查找) 和 8.x (会话覆盖) 可并行
- 9.x (重构) 和 9.y (文档) 可并行

**依赖说明**:
- 2.x 依赖 1.x (需要基础结构)
- 3.x 依赖 2.x (需要数据模型)
- 4.x 依赖 3.x (需要加载器)
- 5.x 依赖 4.x (需要环境变量展开)
- 6.x/7.x/8.x 依赖 5.x (需要验证器)
- 10.x 依赖所有前置任务

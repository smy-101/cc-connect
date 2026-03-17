## ADDED Requirements

### Requirement: 过时文档归档

系统文档 SHALL 将已过时的规划文档移动到归档目录，并保留原始内容供历史参考。

#### Scenario: 归档早期规划文档
- **WHEN** 执行代码清理
- **THEN** `docs/Go 语言 TDD 实现 cc-connect 完整方案（含功能详解）.md` 被移动到 `docs/archive/early-planning-doc.md`
- **AND** 原始文件内容保持不变

#### Scenario: 归档目录结构
- **WHEN** 执行代码清理
- **THEN** `docs/archive/` 目录存在
- **AND** 归档文件可在该目录中访问

### Requirement: 测试文件命名规范

测试文件 SHALL 使用描述性名称，清晰表达测试内容。

#### Scenario: 重命名 claudecode 覆盖率测试
- **WHEN** 执行代码清理
- **THEN** `internal/agent/claudecode/coverage_test.go` 被重命名为 `edge_cases_test.go`
- **AND** 文件内容保持不变

#### Scenario: 重命名 feishu 覆盖率测试
- **WHEN** 执行代码清理
- **THEN** `internal/platform/feishu/coverage_test.go` 被重命名为 `edge_cases_test.go`
- **AND** 文件内容保持不变

### Requirement: 清理后验证

代码清理 SHALL 不影响项目构建和测试。

#### Scenario: 测试通过
- **WHEN** 执行代码清理后
- **THEN** `go test ./...` 全部通过

#### Scenario: 无 vet 警告
- **WHEN** 执行代码清理后
- **THEN** `go vet ./...` 无警告输出

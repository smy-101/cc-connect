## Why

项目已完成 MVP 阶段（飞书 + Claude Code 集成），但存在一些历史遗留文档和代码组织问题需要清理。主要问题：
1. `docs/Go 语言 TDD 实现 cc-connect 完整方案（含功能详解）.md` 是早期规划文档，部分内容已过时
2. 测试文件命名不够清晰（如 `coverage_test.go` 可改为更具描述性的名称）
3. 需要确保项目文档与当前实现状态保持一致

## What Changes

- **归档过时规划文档**：将早期规划文档移动到 `docs/archive/` 目录，保留历史参考价值但标记为已归档
- **重命名测试文件**：将 `coverage_test.go` 重命名为更具描述性的名称（如 `edge_cases_test.go`）
- **更新 README.md**：确保文档反映当前项目状态（经检查已是最新，无需修改）
- **清理确认**：确认项目无无用代码、无过度日志、无死代码

## Capabilities

### New Capabilities

无。本次变更为代码清理，不引入新功能能力。

### Modified Capabilities

无。本次变更不影响任何现有功能的行为规范。

## Impact

- **文档影响**：
  - `docs/Go 语言 TDD 实现 cc-connect 完整方案（含功能详解）.md` → `docs/archive/early-planning-doc.md`
  - 新建 `docs/archive/` 目录

- **代码影响**：
  - `internal/agent/claudecode/coverage_test.go` → `internal/agent/claudecode/edge_cases_test.go`
  - `internal/platform/feishu/coverage_test.go` → `internal/platform/feishu/edge_cases_test.go`

- **无影响**：
  - 不影响任何运行时行为
  - 不影响测试覆盖率
  - 不影响 API 或接口

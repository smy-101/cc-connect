## 1. 文档归档

- [x] 1.1 创建 `docs/archive/` 目录
- [x] 1.2 将 `docs/Go 语言 TDD 实现 cc-connect 完整方案（含功能详解）.md` 移动到 `docs/archive/early-planning-doc.md`

## 2. 测试文件重命名

- [x] 2.1 将 `internal/agent/claudecode/coverage_test.go` 重命名为 `edge_cases_test.go`
- [x] 2.2 将 `internal/platform/feishu/coverage_test.go` 重命名为 `edge_cases_test.go`

## 3. 验证清理结果

- [x] 3.1 运行 `go test ./...` 确认所有测试通过
- [x] 3.2 运行 `go vet ./...` 确认无警告
- [x] 3.3 确认归档文件 `docs/archive/early-planning-doc.md` 可正常访问

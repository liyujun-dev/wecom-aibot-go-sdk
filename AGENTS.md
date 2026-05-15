# AGENTS.md

## 开发原则

### TDD（测试驱动开发）

- 编写实现代码之前，先写测试用例
- 遵循 Red → Green → Refactor 循环
- 使用 Go 标准库 `testing` 包，不引入第三方断言库
- 测试文件命名：`*_test.go`，与源文件同目录

### 标准库优先

- 优先使用 Go 标准库解决问题，避免引入外部依赖
- 当前唯一允许的外部依赖：`gorilla/websocket`（WebSocket 长连接）
- HTTP 请求用 `net/http`，日志用 `log/slog`，加密用 `crypto/*`
- 新依赖引入前需充分论证必要性

### 其他约定

- 所有 I/O 方法接受 `context.Context` 作为第一个参数
- 哨兵错误（sentinel errors）处理错误场景
- 函数式选项模式（Functional Options）构造 `Client`
- 单一 `aibot` 包，不拆分子包

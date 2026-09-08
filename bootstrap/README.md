# Bootstrap 启动与退出

服务依赖初始化完成后才创建并启动 Cron。`AutoStart = false` 时，校验不会启动调度器或执行任务。

退出前由主程序停止接收 HTTP 请求；bootstrap 随后停止并等待健康检查、Cron、NSQ 消费者和 Kafka 消费者，再关闭数据库、其他客户端及日志。所有关闭操作遵守传入的截止时间。任务排空失败时会返回错误并保留依赖，避免仍在执行的任务访问已关闭的资源。第三方驱动若不支持取消，已开始的关闭操作可能继续执行，但不会在超时后访问或修改全局资源指针。

Casbin 启动校验和关闭过程不修改策略。策略变更由 Enforcer 的 AutoSave 持久化，退出时不会使用本机快照覆盖共享策略表。Redis 启动健康检查只执行 `PING`。

## 配置调整

- **Manticore**：配置了用户名或密码时，`Endpoints` 必须为 `https://...`，并使用受信任的证书。可以指向提供 TLS 和认证的反向代理；HTTP Basic Auth 使用标准 Base64 编码。`Port` 仅用于未带协议的主机名。健康检查执行 `SELECT 1`，认证失败、HTTP 错误及 SQL 错误都会使初始化失败。
- **TDengine**：使用官方 `taosWS` 驱动，连接 taosAdapter；默认端口为 `6041`，原生协议端口 `6030` 不适用。`Protocol` 支持 `ws` 和 `wss`。超时和连接池时长使用 Go duration 字符串，例如 `"10s"`、`"30s"`、`"1h"`；旧配置中的毫秒整数需要按示例转换。参见 `conf/service/tdengine.toml`。运行环境仍须提供可用的 TDengine/taosAdapter 服务。
- **NSQ**：消息处理器在 `InitConsumer` 中注册，`pkg/nsq.Consumer` 负责建立连接。退出既等待 SDK 的 `StopChan`，也等待应用处理器执行完毕，随后才关闭生产者。

## 验证

不依赖外部数据库的测试：

```sh
go test -race ./bootstrap/... . ./servers ./servers/httpserver ./library/middleware -timeout 120s
go build . ./bootstrap/... ./servers/... ./pkg/... ./library/...
```

仓库中的认证集成测试需要可访问的 MySQL；`go build ./...` 还会包含 `examples/` 下各自定义 `main` 的独立演示文件，应分别运行这些演示。

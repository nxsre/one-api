# logwatch — one-api 上游健康日志分析工具

解析 one-api 运行日志（`[GIN]` 访问日志 + `[ERROR]` 中继错误日志），按**上游 host** 与**渠道**聚合健康度，自动发现上游异常（EOF / 超时 / 连接拒绝 / DNS / TLS / 5xx / 高延迟），并区分「上游故障」与「客户端问题」。纯 Go 标准库，独立 `go.mod`，不影响主仓库。

## 为什么需要它

one-api 对一次中继错误会打印多行（`[ErrorWrapper]` + `[relayWithRetry]` + `[processChannelRelayError]`），且 `[GIN]` 访问日志不带渠道号。logwatch 做了：
- **去重**：每次错误只按规范行计一次，避免 ×2/×3 虚高。
- **归类**：把 `unexpected EOF`、`context canceled`、`deadline exceeded`、`connection refused`、`no such host`、`x509/tls`、`proto is invalid` 等归类，并区分**上游故障**（EOF/超时/拒绝/DNS/TLS/5xx）与**客户端问题**（4xx/bad_request/客户端取消）。
- **归因**：从错误信息里的 URL 提取上游 host（如 `api.novita.ai`），并关联渠道号。

## 用法

```bash
cd logwatch
go build -o logwatch ./cmd/logwatch    # 或直接 go run ./cmd/logwatch ...

# 一次性分析（三选一输入源）
docker logs --since 30m one-api 2>&1 | go run ./cmd/logwatch     # 管道
go run ./cmd/logwatch -file oneapi.log                            # 文件
go run ./cmd/logwatch -docker one-api -since 30m                  # 直接抓 docker logs

# 持续监控：每 30s 抓最近 10m 日志，发现异常打印 ⚠
go run ./cmd/logwatch -docker one-api -watch -window 10m -interval 30s

# JSON 输出 + 自定义阈值（接入 CI/告警；有异常退出码 1）
go run ./cmd/logwatch -docker one-api -since 1h -json -conn-errors 5 -err-rate 10 -p95 20000
```

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-docker <name>` | - | 直接抓该容器 `docker logs` |
| `-file <path>` | - | 从文件读取（不指定且无 `-docker` 时读 stdin） |
| `-since` | `30m` | docker logs 时间窗（`30m`/`1h`/RFC3339） |
| `-watch` | false | 持续监控（需配合 `-docker`） |
| `-window` / `-interval` | `10m` / `30s` | watch 每次回看窗口 / 刷新间隔 |
| `-json` | false | JSON 输出 |
| `-conn-errors` | 3 | 单上游连接类错误数告警阈值 |
| `-err-rate` | 5 | 整体错误率(%)告警阈值 |
| `-p95` | 15000 | p95 延迟(ms)告警阈值 |

退出码：检出异常返回 `1`（适合 cron/CI）。

## 输出示例（真实数据）

```
时间窗口: 09:43:51 ~ 10:24:54 | 日志行: 488 | 访问: 412 | 错误: 9
延迟(ms): p50=2 p95=7426 p99=26759 max=167637
状态码: 200=410  400=1  500=1
错误分类: bad_request=2, client_canceled=1, eof=4, http_4xx=1, http_5xx=1

—— 上游 (按错误数) ——
UPSTREAM       ERRORS  CLASSES  CHANNELS  LAST
api.novita.ai  1       eof=1    ch1       09:53:50

—— 渠道 (按错误数) ——
CHANNEL  ERRORS  CLASSES                                  UPSTREAMS      LAST
1        6       bad_request=1, client_canceled=1, eof=4  api.novita.ai  09:59:54

—— 诊断 ——
⚠ 出现 1 个 5xx 响应
```

## 错误分类

| class | 含义 | 归属 |
|-------|------|------|
| `eof` / `timeout` / `conn_refused` / `dns` / `tls` / `http_5xx` | 连接/上游层故障 | **上游** |
| `http_4xx` / `bad_request` | 请求被拒（含 content 格式错误） | 客户端 |
| `client_canceled` | 客户端中途断开（`context canceled`） | 客户端 |

> 说明：流式途中的 EOF/取消（`stream_read_failed`）日志不含上游 URL，会归到对应**渠道**但无法归到具体 host；带 URL 的 `do_request_failed` 才能归到 host。

## 测试

```bash
go test ./...   # 用真实日志样本校验解析、分类、去重与异常判定
```

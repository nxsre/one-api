# bench — one-api 压测工具

依赖零（纯标准库）的负载生成器 + OpenAI 兼容 mock 上游，用于压测 one-api 网关本身的热路径，而不依赖真实大模型厂商。

## 构建

```bash
go build -o bench ./cmd/bench
```

## 子命令

### `bench mock` — mock 上游

模拟一个 OpenAI 兼容上游，可配置思考延迟、流式 chunk 数与节奏、usage token 数。
把 one-api 的某个渠道 `base_url` 指向它，即可压测 **relay 全链路**（鉴权 → 选路 → 配额预扣 → 转发 → 计费 → 日志），上游开销完全可控。

```bash
bench mock -listen :19000 -latency 80ms -chunks 20 -chunk-delay 10ms \
           -prompt-tokens 32 -completion-tokens 256
```

| flag | 说明 | 默认 |
|---|---|---|
| `-listen` | 监听地址 | `:9000` |
| `-latency` | 响应前的模拟思考延迟 | `50ms` |
| `-chunks` | 流式模式的 SSE chunk 数 | `16` |
| `-chunk-delay` | chunk 间隔（模拟出 token 速率） | `8ms` |
| `-prompt-tokens` / `-completion-tokens` | usage 中上报的 token 数 | `32` / `64` |

支持 `/v1/chat/completions`（流式与非流式）、`/v1/models`、`/healthz`。

### `bench load` — 负载生成器

```bash
# 闭环：固定并发打满，测极限吞吐
bench load -url http://127.0.0.1:3000/healthz -c 200 -d 30s

# 开环：固定目标 QPS，测该速率下的延迟（含排队）
bench load -url http://127.0.0.1:3000/healthz -rps 2000 -d 30s

# relay 全链路（先配好指向 mock 的渠道与 token）
bench load -url http://127.0.0.1:3000/v1/chat/completions \
           -H "Authorization: Bearer sk-xxx" \
           -body @prompt.json -c 100 -d 60s -stream
```

| flag | 说明 | 默认 |
|---|---|---|
| `-url` | 目标 URL（必填） | — |
| `-method` | HTTP 方法 | 有 body 时 POST，否则 GET |
| `-body` | 请求体字符串，或 `@文件` | — |
| `-H` | 请求头 `Key: Value`（可重复） | — |
| `-c` | 并发连接数（闭环） | `50` |
| `-rps` | 目标 QPS（开环）；0 = 闭环 | `0` |
| `-d` | 持续时长 | `30s` |
| `-n` | 总请求数（>0 时覆盖 `-d`） | `0` |
| `-timeout` | 单请求超时 | `30s` |
| `-stream` | 按流处理：单独测 TTFB（首 token），并读完整个流 | `false` |

输出：吞吐、状态码分布、传输错误分类、延迟分位（min/mean/max + p50/p90/p95/p99）；
`-stream` 时额外给出 **TTFB（首字节/首 token）** 分位，便于区分「首 token 延迟」与「整段生成耗时」。

## 端到端压测 relay 的步骤

1. 启动 mock 上游：`bench mock -listen :19000 -latency 80ms`
2. 在 one-api 后台新建一个渠道，`base_url = http://127.0.0.1:19000`，key 任意，分组/模型按需（如 `gpt-4o-mini`）。
3. 建一个 token，写好 `prompt.json`（OpenAI chat 请求体）。
4. 施压：`bench load -url http://<one-api>/v1/chat/completions -H "Authorization: Bearer <token>" -body @prompt.json -c 100 -d 60s`
5. 观察 one-api 的吞吐/延迟，以及 DB/Redis 负载，定位瓶颈。

## 验证连接池调优效果

对比 relay 上游连接池调优前后（`RELAY_MAX_IDLE_CONNS_PER_HOST`，默认已从 Go 的 2 提到 100）：

```bash
# 调小到 2（模拟旧默认），高并发下应看到吞吐下降、p99 上升（连接反复重建）
RELAY_MAX_IDLE_CONNS_PER_HOST=2 ./one-api ...
bench load -url http://<one-api>/v1/chat/completions -H "Authorization: Bearer <token>" -body @prompt.json -c 200 -d 30s

# 默认 100，吞吐更高、尾延迟更稳
./one-api ...
bench load ... 同上
```

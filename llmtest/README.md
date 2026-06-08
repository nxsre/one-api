# llmtest — 大模型接口测试工具（库 + CLI）

一套协议无关的大模型测试框架：把「测试场景」翻译成各家协议的线格式，发请求、归一化响应、做**严格断言 + 语义检查**，最后输出通过率 / 延迟 / token 统计报告。

- **协议**：OpenAI（`chat` 全形式 / `completions` / `embeddings`）、Anthropic `messages`、Gemini `generateContent`
- **目标**：既能打 one-api 网关（OpenAI 兼容入口），也能打各家原生端点 —— 由 `protocol + base_url` 决定
- **输入**：每个场景内置随机生成（带可复现种子），也可在配置里按场景自定义
- **零外部依赖**：纯 Go 标准库实现，单独 `go.mod`，不影响主仓库

## 快速开始

```bash
cd llmtest

# 1) 生成示例配置
go run ./cmd/llmtest -init > config.json

# 2) 填好 base_url / api_key / 模型名（api_key 支持 ${ENV} 占位，从环境变量取）
export ONEAPI_KEY=sk-xxx

# 3) 列出全部场景
go run ./cmd/llmtest -list

# 4) 运行
go run ./cmd/llmtest -config config.json                       # 全部场景 × 全部 target
go run ./cmd/llmtest -config config.json -scenarios math,tool_call,embeddings -v
go run ./cmd/llmtest -config config.json -targets one-api-gateway -json report.json

# 编译成二进制
go build -o llmtest ./cmd/llmtest
```

退出码：存在 `FAIL`/`ERROR` 时返回 `1`，便于接入 CI。

## 命令行参数

| 参数 | 说明 |
|------|------|
| `-config` | 配置文件路径（默认 `config.json`） |
| `-scenarios` | 仅运行这些场景（逗号分隔） |
| `-targets` | 仅运行这些 target（按 name 逗号分隔） |
| `-json` | 把结果同时写入 JSON 文件 |
| `-v` | 输出失败明细与原始响应 |
| `-timeout` / `-concurrency` / `-seed` | 覆盖配置中的对应项 |
| `-list` / `-init` | 列出场景 / 打印示例配置 |

## 配置

```json
{
  "targets": [
    { "name": "one-api-gateway", "protocol": "openai",
      "base_url": "http://localhost:13000", "api_key": "${ONEAPI_KEY}",
      "models": { "chat": "gpt-4o-mini", "completion": "gpt-3.5-turbo-instruct", "embedding": "text-embedding-3-small" } },
    { "name": "anthropic-native", "protocol": "anthropic",
      "base_url": "https://api.anthropic.com", "api_key": "${ANTHROPIC_API_KEY}",
      "models": { "chat": "claude-3-5-sonnet-20241022" } },
    { "name": "gemini-native", "protocol": "gemini",
      "base_url": "https://generativelanguage.googleapis.com", "api_key": "${GEMINI_API_KEY}",
      "models": { "chat": "gemini-1.5-flash" } }
  ],
  "timeout_seconds": 60,
  "concurrency": 4,
  "seed": 0,
  "custom_inputs": { "simple_chat": "用一句话介绍长江。" }
}
```

- `base_url` 填到主机根即可（带不带 `/v1`、`/v1beta` 都行，工具会规整）。
- `custom_inputs[场景ID]`：用自定义提示覆盖该场景的随机输入；此时只保留结构性断言（非空 / 流式分块 / 工具调用 / JSON / 向量维度），放宽依赖标准答案的语义断言。
- 某 target 未配置某形态的模型，对应场景自动标记 `SKIP`；协议不支持的形态（如 Anthropic/Gemini 的 embeddings）同样 `SKIP`。

## 内置场景

| ID | 形态 | 校验点 |
|----|------|--------|
| `simple_chat` | chat | 事实问答语义校验 + usage |
| `math` | chat | 随机乘法，结果精确匹配 |
| `multi_turn` | chat | 多轮记忆（随机数字回取） |
| `system_prompt` | chat | system 指令遵循（结尾词） |
| `streaming` | chat | SSE 分块数 ≥ 2 |
| `tool_call` | chat | 触发 `get_weather` 且参数含 `location`（流式自动拼接参数） |
| `json_mode` | chat | 合法 JSON 且含 `name`/`age` 键 |
| `vision` | chat | 现场生成纯色 PNG，校验模型说出对应颜色 |
| `long_context` | chat | 长文本中检索随机「针」 |
| `completions` | completion | 传统 `/v1/completions` 补全 |
| `embeddings` | embedding | 条数/维度一致 + 余弦相似度语义有效性 |

## 作为库使用

```go
import "llmtest/core"

cfg, _ := core.LoadConfig("config.json")
results := core.Run(cfg, []string{"math", "tool_call"}, true)
core.WriteReport(os.Stdout, results, true)
```

核心抽象：`Provider`（协议实现）、`Scenario`/`Case`（场景与实例）、`Expectation`+`Evaluate`（断言）、`Run`（编排，构建期顺序保证复现、执行期并发）。新增协议实现 `core.Provider` 接口并在 `NewProvider` 注册即可；新增场景往 `AllScenarios()` 追加一项。

## 测试

```bash
go test ./...   # httptest 离线校验请求构造 / 流式拼接 / 解析 / 断言，无需真实密钥
```

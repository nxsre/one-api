# Relay API curl 测试用例

本文档覆盖 One API 对外暴露的 **Relay（转发）** 端点 curl 示例，便于联调与回归。

## 环境变量

```bash
export ONE_API_BASE="http://127.0.0.1:3000"   # One API 服务地址
export ONE_API_TOKEN="sk-xxxx"                # 控制台创建的 API Token
export OPENAI_BETA="assistants=v2"            # Assistants / Threads 必需
export MODEL_CHAT="gpt-4o"
export MODEL_EMBED="text-embedding-3-small"
export MODEL_IMAGE="dall-e-3"
export MODEL_TTS="tts-1"
export MODEL_WHISPER="whisper-1"
export MODEL_FT="gpt-4o-mini-2024-07-18"
```

通用请求头：

```bash
AUTH=(-H "Authorization: Bearer ${ONE_API_TOKEN}")
BETA=(-H "OpenAI-Beta: ${OPENAI_BETA}")
JSON=(-H "Content-Type: application/json")
```

以下示例默认使用 `/v1` 前缀；等价路径还有 `/openai/v1/*`（OpenAI 专用前缀）。

---

## 1. 模型列表

```bash
# 列出可用模型
curl -sS "${ONE_API_BASE}/v1/models" "${AUTH[@]}"

# 查询单个模型
curl -sS "${ONE_API_BASE}/v1/models/${MODEL_CHAT}" "${AUTH[@]}"

# 删除微调模型（透传至上游 OpenAI）
curl -sS -X DELETE "${ONE_API_BASE}/v1/models/ft:gpt-4o-mini:org:xxx" "${AUTH[@]}"
```

---

## 2. Chat Completions

```bash
curl -sS "${ONE_API_BASE}/v1/chat/completions" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_CHAT}"'",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# 流式
curl -sS "${ONE_API_BASE}/v1/chat/completions" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_CHAT}"'",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to 3"}]
  }'
```

---

## 3. Completions（Legacy）

```bash
curl -sS "${ONE_API_BASE}/v1/completions" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_CHAT}"'",
    "prompt": "Say hi",
    "max_tokens": 16
  }'
```

---

## 4. Responses API

```bash
# 创建响应
curl -sS "${ONE_API_BASE}/v1/responses" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_CHAT}"'",
    "input": "What is 2+2?"
  }'

# 获取响应（需替换 RESPONSE_ID）
curl -sS "${ONE_API_BASE}/v1/responses/resp_xxx?model=${MODEL_CHAT}" "${AUTH[@]}"

# 取消响应
curl -sS -X POST "${ONE_API_BASE}/v1/responses/resp_xxx/cancel?model=${MODEL_CHAT}" \
  "${AUTH[@]}" "${JSON[@]}" -d '{}'
```

---

## 5. Realtime

```bash
# 创建 Realtime 会话
curl -sS "${ONE_API_BASE}/v1/realtime/sessions" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "gpt-4o-realtime-preview",
    "modalities": ["text", "audio"]
  }'

# WebSocket（需 wscat 等工具）
# wscat -c "${ONE_API_BASE}/v1/realtime?model=gpt-4o-realtime-preview" \
#   -H "Authorization: Bearer ${ONE_API_TOKEN}"
```

---

## 6. Embeddings

```bash
curl -sS "${ONE_API_BASE}/v1/embeddings" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_EMBED}"'",
    "input": "The food was delicious."
  }'

# Azure 风格路径
curl -sS "${ONE_API_BASE}/v1/engines/${MODEL_EMBED}/embeddings" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{"input": "hello"}'
```

---

## 7. Images

```bash
# 文生图
curl -sS "${ONE_API_BASE}/v1/images/generations" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_IMAGE}"'",
    "prompt": "A white cat",
    "n": 1,
    "size": "1024x1024"
  }'

# 图像编辑（multipart 透传）
curl -sS "${ONE_API_BASE}/v1/images/edits" \
  "${AUTH[@]}" \
  -F "model=${MODEL_IMAGE}" \
  -F "image=@/path/to/image.png" \
  -F "prompt=Add a hat"

# 图像变体（multipart 透传）
curl -sS "${ONE_API_BASE}/v1/images/variations" \
  "${AUTH[@]}" \
  -F "model=dall-e-2" \
  -F "image=@/path/to/image.png" \
  -F "n=1"
```

---

## 8. Audio

```bash
# 语音转文字
curl -sS "${ONE_API_BASE}/v1/audio/transcriptions" \
  "${AUTH[@]}" \
  -F "file=@/path/to/audio.mp3" \
  -F "model=${MODEL_WHISPER}"

# 语音翻译
curl -sS "${ONE_API_BASE}/v1/audio/translations" \
  "${AUTH[@]}" \
  -F "file=@/path/to/audio.mp3" \
  -F "model=${MODEL_WHISPER}"

# 文字转语音
curl -sS "${ONE_API_BASE}/v1/audio/speech" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_TTS}"'",
    "input": "Hello world",
    "voice": "alloy"
  }' --output speech.mp3
```

---

## 9. Files（透传）

```bash
# 上传文件
curl -sS "${ONE_API_BASE}/v1/files" \
  "${AUTH[@]}" \
  -F "purpose=assistants" \
  -F "file=@/path/to/doc.pdf"

# 列出文件
curl -sS "${ONE_API_BASE}/v1/files" "${AUTH[@]}"

# 获取文件元数据（替换 FILE_ID）
curl -sS "${ONE_API_BASE}/v1/files/file-xxx" "${AUTH[@]}"

# 下载文件内容
curl -sS "${ONE_API_BASE}/v1/files/file-xxx/content" "${AUTH[@]}" -o downloaded.bin

# 删除文件
curl -sS -X DELETE "${ONE_API_BASE}/v1/files/file-xxx" "${AUTH[@]}"
```

---

## 10. Fine-tuning（透传）

```bash
# 创建微调任务
curl -sS "${ONE_API_BASE}/v1/fine_tuning/jobs" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_FT}"'",
    "training_file": "file-xxx"
  }'

# 列出任务
curl -sS "${ONE_API_BASE}/v1/fine_tuning/jobs" "${AUTH[@]}"

# 获取任务详情（替换 JOB_ID）
curl -sS "${ONE_API_BASE}/v1/fine_tuning/jobs/ftjob-xxx" "${AUTH[@]}"

# 取消任务
curl -sS -X POST "${ONE_API_BASE}/v1/fine_tuning/jobs/ftjob-xxx/cancel" \
  "${AUTH[@]}" "${JSON[@]}" -d '{}'

# 事件流（SSE）
curl -sS -N "${ONE_API_BASE}/v1/fine_tuning/jobs/ftjob-xxx/events" "${AUTH[@]}"
```

---

## 11. Assistants（透传，需 OpenAI-Beta）

```bash
# 创建 Assistant
curl -sS "${ONE_API_BASE}/v1/assistants" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{
    "model": "'"${MODEL_CHAT}"'",
    "name": "Demo Assistant",
    "instructions": "You are helpful."
  }'

# 列出 Assistants
curl -sS "${ONE_API_BASE}/v1/assistants" "${AUTH[@]}" "${BETA[@]}"

# 获取 / 更新 / 删除（替换 ASST_ID）
curl -sS "${ONE_API_BASE}/v1/assistants/asst_xxx" "${AUTH[@]}" "${BETA[@]}"

curl -sS -X POST "${ONE_API_BASE}/v1/assistants/asst_xxx" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{"name": "Renamed Assistant"}'

curl -sS -X DELETE "${ONE_API_BASE}/v1/assistants/asst_xxx" \
  "${AUTH[@]}" "${BETA[@]}"

# Assistant 文件
curl -sS -X POST "${ONE_API_BASE}/v1/assistants/asst_xxx/files" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{"file_id": "file-xxx"}'

curl -sS "${ONE_API_BASE}/v1/assistants/asst_xxx/files" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS "${ONE_API_BASE}/v1/assistants/asst_xxx/files/file-xxx" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS -X DELETE "${ONE_API_BASE}/v1/assistants/asst_xxx/files/file-xxx" \
  "${AUTH[@]}" "${BETA[@]}"
```

---

## 12. Threads（透传，需 OpenAI-Beta）

```bash
# 创建 Thread
curl -sS "${ONE_API_BASE}/v1/threads" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{
    "messages": [{"role": "user", "content": "Hi"}]
  }'

# 获取 / 更新 / 删除 Thread（替换 THREAD_ID）
curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx" "${AUTH[@]}" "${BETA[@]}"

curl -sS -X POST "${ONE_API_BASE}/v1/threads/thread_xxx" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{"metadata": {"topic": "demo"}}'

curl -sS -X DELETE "${ONE_API_BASE}/v1/threads/thread_xxx" \
  "${AUTH[@]}" "${BETA[@]}"

# Messages
curl -sS -X POST "${ONE_API_BASE}/v1/threads/thread_xxx/messages" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{"role": "user", "content": "Follow up question"}'

curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/messages" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/messages/msg_xxx" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS -X POST "${ONE_API_BASE}/v1/threads/thread_xxx/messages/msg_xxx" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{"metadata": {"edited": "true"}}'

curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/messages/msg_xxx/files" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/messages/msg_xxx/files/file-xxx" \
  "${AUTH[@]}" "${BETA[@]}"

# Runs
curl -sS -X POST "${ONE_API_BASE}/v1/threads/thread_xxx/runs" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{"assistant_id": "asst_xxx"}'

curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/runs" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/runs/run_xxx" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS -X POST "${ONE_API_BASE}/v1/threads/thread_xxx/runs/run_xxx" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{"metadata": {"note": "update"}}'

curl -sS -X POST "${ONE_API_BASE}/v1/threads/thread_xxx/runs/run_xxx/cancel" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" -d '{}'

curl -sS -X POST "${ONE_API_BASE}/v1/threads/thread_xxx/runs/run_xxx/submit_tool_outputs" \
  "${AUTH[@]}" "${BETA[@]}" "${JSON[@]}" \
  -d '{
    "tool_outputs": [{"tool_call_id": "call_xxx", "output": "42"}]
  }'

# Run Steps
curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/runs/run_xxx/steps" \
  "${AUTH[@]}" "${BETA[@]}"

curl -sS "${ONE_API_BASE}/v1/threads/thread_xxx/runs/run_xxx/steps/step_xxx" \
  "${AUTH[@]}" "${BETA[@]}"
```

---

## 13. Moderations

```bash
curl -sS "${ONE_API_BASE}/v1/moderations" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{"input": "I want to hurt someone."}'
```

---

## 14. Edits（Legacy）

```bash
curl -sS "${ONE_API_BASE}/v1/edits" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "model": "gpt-3.5-turbo-instruct",
    "input": "What day of the wek is it?",
    "instruction": "Fix spelling"
  }'
```

---

## 15. Anthropic Messages

```bash
# /v1/messages（OpenAI 风格入口，经协议桥接）
curl -sS "${ONE_API_BASE}/v1/messages" \
  -H "x-api-key: ${ONE_API_TOKEN}" \
  -H "anthropic-version: 2023-06-01" \
  "${JSON[@]}" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# 原生 Anthropic 前缀
curl -sS "${ONE_API_BASE}/anthropic/v1/messages" \
  -H "x-api-key: ${ONE_API_TOKEN}" \
  -H "anthropic-version: 2023-06-01" \
  "${JSON[@]}" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 256,
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Token 计数
curl -sS "${ONE_API_BASE}/anthropic/v1/messages/count_tokens" \
  -H "x-api-key: ${ONE_API_TOKEN}" \
  -H "anthropic-version: 2023-06-01" \
  "${JSON[@]}" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Anthropic 模型列表
curl -sS "${ONE_API_BASE}/anthropic/v1/models" \
  -H "x-api-key: ${ONE_API_TOKEN}"
```

---

## 16. Gemini 原生

```bash
export GEMINI_MODEL="gemini-2.0-flash"

# 模型列表
curl -sS "${ONE_API_BASE}/v1beta/models" "${AUTH[@]}"
curl -sS "${ONE_API_BASE}/gemini/v1beta/models" "${AUTH[@]}"

# generateContent
curl -sS "${ONE_API_BASE}/v1beta/models/${GEMINI_MODEL}:generateContent" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "contents": [{"role": "user", "parts": [{"text": "Hello"}]}]
  }'

# embedContent
curl -sS "${ONE_API_BASE}/v1beta/models/text-embedding-004:embedContent" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "content": {"parts": [{"text": "hello world"}]}
  }'

# 兼容 /gemini/models/{model}:method
curl -sS "${ONE_API_BASE}/gemini/models/${GEMINI_MODEL}:generateContent" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "contents": [{"role": "user", "parts": [{"text": "Hello"}]}]
  }'
```

---

## 17. 自定义渠道代理

```bash
# 将请求转发到指定渠道（channel id = 1）
curl -sS "${ONE_API_BASE}/v1/oneapi/proxy/1/https://httpbin.org/get" \
  "${AUTH[@]}"
```

---

## 18. 高德地图代理

```bash
curl -sS "${ONE_API_BASE}/amap" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{
    "path": "/v3/geocode/geo",
    "params": {"address": "北京市朝阳区"}
  }'
```

---

## 注意事项

1. **渠道要求**：Assistants / Threads / Fine-tuning / Files 等透传 API 需要上游为 **OpenAI 官方或完全兼容** 的渠道；非 OpenAI 渠道可能返回 404 或格式错误。
2. **OpenAI-Beta**：Assistants v2 必须携带 `OpenAI-Beta: assistants=v2`。
3. **模型路由**：GET/DELETE 类透传请求若无 body 中的 `model`，One API 会使用默认模型（如 `gpt-4o`）选择渠道；可在 query 中附加 `?model=xxx` 指定。
4. **计费**：透传 API 按预扣额度计费，不按实际上游 token 精确结算。
5. **前缀等价**：除特别说明外，`/v1/...` 与 `/openai/v1/...` 行为一致。

## 快速冒烟脚本

可执行脚本见同目录 [`relay-api-curl-examples.sh`](./relay-api-curl-examples.sh)（需先设置环境变量，部分用例会跳过需真实资源的步骤）。

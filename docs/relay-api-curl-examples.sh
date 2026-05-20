#!/usr/bin/env bash
# Relay API 冒烟测试脚本。用法：
#   export ONE_API_BASE=http://127.0.0.1:3000 ONE_API_TOKEN=sk-xxx
#   ./docs/relay-api-curl-examples.sh
set -euo pipefail

: "${ONE_API_BASE:?set ONE_API_BASE}"
: "${ONE_API_TOKEN:?set ONE_API_TOKEN}"

OPENAI_BETA="${OPENAI_BETA:-assistants=v2}"
MODEL_CHAT="${MODEL_CHAT:-gpt-4o}"
MODEL_EMBED="${MODEL_EMBED:-text-embedding-3-small}"
MODEL_IMAGE="${MODEL_IMAGE:-dall-e-3}"
MODEL_TTS="${MODEL_TTS:-tts-1}"
MODEL_WHISPER="${MODEL_WHISPER:-whisper-1}"
MODEL_FT="${MODEL_FT:-gpt-4o-mini-2024-07-18}"
GEMINI_MODEL="${GEMINI_MODEL:-gemini-2.0-flash}"

AUTH=(-H "Authorization: Bearer ${ONE_API_TOKEN}")
BETA=(-H "OpenAI-Beta: ${OPENAI_BETA}")
JSON=(-H "Content-Type: application/json")

run() {
  echo "==> $*"
  "$@"
  echo
}

echo "=== Models ==="
run curl -sS "${ONE_API_BASE}/v1/models" "${AUTH[@]}"
run curl -sS "${ONE_API_BASE}/v1/models/${MODEL_CHAT}" "${AUTH[@]}"

echo "=== Chat ==="
run curl -sS "${ONE_API_BASE}/v1/chat/completions" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d "{\"model\":\"${MODEL_CHAT}\",\"messages\":[{\"role\":\"user\",\"content\":\"ping\"}],\"max_tokens\":8}"

echo "=== Embeddings ==="
run curl -sS "${ONE_API_BASE}/v1/embeddings" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d "{\"model\":\"${MODEL_EMBED}\",\"input\":\"hello\"}"

echo "=== Moderations ==="
run curl -sS "${ONE_API_BASE}/v1/moderations" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{"input":"hello world"}'

echo "=== Files list ==="
run curl -sS "${ONE_API_BASE}/v1/files" "${AUTH[@]}"

echo "=== Fine-tuning list ==="
run curl -sS "${ONE_API_BASE}/v1/fine_tuning/jobs" "${AUTH[@]}"

echo "=== Assistants list ==="
run curl -sS "${ONE_API_BASE}/v1/assistants" "${AUTH[@]}" "${BETA[@]}"

echo "=== Gemini models ==="
run curl -sS "${ONE_API_BASE}/v1beta/models" "${AUTH[@]}" || true

echo "=== Anthropic models ==="
run curl -sS "${ONE_API_BASE}/anthropic/v1/models" \
  -H "x-api-key: ${ONE_API_TOKEN}" || true

echo "Done. See docs/relay-api-curl-examples.md for full examples."

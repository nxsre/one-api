#!/usr/bin/env bash
# 验证登录 AES + proof 改造（不依赖 Node）。
# 用法:
#   BASE=http://127.0.0.1:13000 USER=root PASS='your-password' ./scripts/verify-login-api.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BASE="${BASE:-http://127.0.0.1:13000}"
USER="${USER:-root}"
PASS="${PASS:-}"
COOKIEJAR="${COOKIEJAR:-/tmp/oneapi_verify_login.cookies}"
PROOF_JSON="${PROOF_JSON:-/tmp/oneapi_proof.json}"

if ! command -v jq >/dev/null 2>&1; then
  echo "需要 jq: brew install jq 或 apt install jq" >&2
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "需要 curl" >&2
  exit 1
fi

encrypt_aes() {
  (cd "$ROOT" && go run ./scripts/login_encrypt "$1" "$2")
}

echo "== 1) GET /api/status：不应含 login_password_rsa_public_key =="
STATUS_JSON="$(curl -sS "${BASE}/api/status")"
echo "$STATUS_JSON" | jq -e '.success == true' >/dev/null
if echo "$STATUS_JSON" | jq -e '.data | has("login_password_rsa_public_key")' >/dev/null 2>&1; then
  echo "FAIL: status 仍返回 login_password_rsa_public_key" >&2
  exit 1
fi
echo "OK: 无 RSA 公钥字段"
SCOPE="$(echo "$STATUS_JSON" | jq -r '.data.status_scope // "unknown"')"
SECURE_LOGIN="$(echo "$STATUS_JSON" | jq -r '.data.secure_password_login // false')"
echo "status_scope=$SCOPE (anonymous 应为 public)"
echo "secure_password_login=$SECURE_LOGIN (默认 false；开启安全登录后为 true)"
if echo "$STATUS_JSON" | jq -e '.data | has("nacos_enabled")' >/dev/null 2>&1; then
  echo "WARN: 未登录不应返回 nacos_enabled" >&2
fi

echo ""
echo "== 2) GET /api/user/login/request-proof：应含 login_enc_key =="
rm -f "$COOKIEJAR"
curl -sS -c "$COOKIEJAR" -b "$COOKIEJAR" \
  "${BASE}/api/user/login/request-proof" | tee "$PROOF_JSON" | jq .

LOGIN_ID="$(jq -r '.data.login_request_id' "$PROOF_JSON")"
LOGIN_TS="$(jq -r '.data.login_request_ts' "$PROOF_JSON")"
LOGIN_SIG="$(jq -r '.data.login_request_sig' "$PROOF_JSON")"
LOGIN_ENC_KEY="$(jq -r '.data.login_enc_key' "$PROOF_JSON")"
PROOF_OK="$(jq -r '.success' "$PROOF_JSON")"

if [[ "$PROOF_OK" != "true" || -z "$LOGIN_ID" || "$LOGIN_ID" == "null" || -z "$LOGIN_ENC_KEY" || "$LOGIN_ENC_KEY" == "null" ]]; then
  echo "FAIL: request-proof 未返回完整凭证" >&2
  exit 1
fi
echo "OK: proof id=${LOGIN_ID}"

echo ""
echo "== 3) 负例：错误密文应登录失败 =="
curl -sS -b "$COOKIEJAR" -c "$COOKIEJAR" \
  -H 'Content-Type: application/json' \
  -d "$(jq -n \
    --arg u "$USER" \
    --arg id "$LOGIN_ID" \
    --argjson ts "$LOGIN_TS" \
    --arg sig "$LOGIN_SIG" \
    '{username:$u, password:"not-valid-ciphertext", login_request_id:$id, login_request_ts:$ts, login_request_sig:$sig}')" \
  "${BASE}/api/user/login" | jq .
echo "(若上一步已消耗 proof，此处可能提示凭证过期，可忽略)"

echo ""
echo "== 4) 重新 request-proof + AES 加密密码后登录 =="
curl -sS -c "$COOKIEJAR" -b "$COOKIEJAR" \
  "${BASE}/api/user/login/request-proof" | tee "$PROOF_JSON" | jq -e '.success == true' >/dev/null

LOGIN_ID="$(jq -r '.data.login_request_id' "$PROOF_JSON")"
LOGIN_TS="$(jq -r '.data.login_request_ts' "$PROOF_JSON")"
LOGIN_SIG="$(jq -r '.data.login_request_sig' "$PROOF_JSON")"
LOGIN_ENC_KEY="$(jq -r '.data.login_enc_key' "$PROOF_JSON")"

if [[ -z "$PASS" ]]; then
  echo "跳过登录：请设置环境变量 PASS='你的密码'"
  echo "示例: BASE=${BASE} USER=${USER} PASS='***' $0"
  exit 0
fi

if [[ "$SECURE_LOGIN" == "true" ]]; then
  curl -sS -c "$COOKIEJAR" -b "$COOKIEJAR" \
    "${BASE}/api/user/login/request-proof" | tee "$PROOF_JSON" | jq -e '.success == true' >/dev/null
  LOGIN_ID="$(jq -r '.data.login_request_id' "$PROOF_JSON")"
  LOGIN_TS="$(jq -r '.data.login_request_ts' "$PROOF_JSON")"
  LOGIN_SIG="$(jq -r '.data.login_request_sig' "$PROOF_JSON")"
  LOGIN_ENC_KEY="$(jq -r '.data.login_enc_key' "$PROOF_JSON")"
  ENC_PASS="$(encrypt_aes "$LOGIN_ENC_KEY" "$PASS")"
  LOGIN_BODY="$(jq -n \
    --arg u "$USER" \
    --arg p "$ENC_PASS" \
    --arg id "$LOGIN_ID" \
    --argjson ts "$LOGIN_TS" \
    --arg sig "$LOGIN_SIG" \
    '{username:$u, password:$p, login_request_id:$id, login_request_ts:$ts, login_request_sig:$sig}')"
else
  echo "安全登录未开启，使用明文 password 登录"
  LOGIN_BODY="$(jq -n --arg u "$USER" --arg p "$PASS" '{username:$u, password:$p}')"
fi

echo ""
echo "== POST /api/user/login =="
curl -sS -b "$COOKIEJAR" -c "$COOKIEJAR" \
  -H 'Content-Type: application/json' \
  -d "$LOGIN_BODY" \
  "${BASE}/api/user/login" | jq .

echo ""
echo "== 5) 可选：GET /api/user/login/captcha（若启用图形验证码）=="
curl -sS -b "$COOKIEJAR" -c "$COOKIEJAR" \
  "${BASE}/api/user/login/captcha" | jq '{success, message, data: (.data | {login_request_id, login_enc_key, captcha_id, dot_num})}'

echo ""
echo "完成。Cookie 保存在: $COOKIEJAR"

#!/usr/bin/env bash
# 将 alibaba/nacos 克隆到构建上下文内默认路径，供 Dockerfile 挂载合并 legacy 控制台静态资源。
# 用法：在 docker build 前执行 ./scripts/ensure-nacos-upstream.sh
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DST="${REPO_ROOT}/third_party/nacos/nacos-upstream"
if [[ -d "${DST}/console/src/main/resources/static/console-ui/public" ]] &&
  [[ -n "$(ls -A "${DST}/console/src/main/resources/static/console-ui/public" 2>/dev/null)" ]]; then
  echo "OK: Nacos upstream already present: ${DST}"
  exit 0
fi
if [[ -e "$DST" ]]; then
  echo "ERROR: ${DST} exists but is not a usable Nacos tree (missing non-empty console-ui/public). Remove it and retry." >&2
  exit 1
fi
mkdir -p "$(dirname "$DST")"
git clone --depth 1 https://github.com/alibaba/nacos.git "$DST"
echo "Cloned Nacos to ${DST}"

#!/usr/bin/env bash
# 在 web/tob 目录构建 Docker 镜像，可选构建后启动容器。
#
# 用法:
#   ./build-image.sh
#   VITE_API_BASE=https://your-api.example.com ./build-image.sh
#   ./build-image.sh --run
#   ./build-image.sh --run -p 8886
#   IMAGE_NAME=my-tob IMAGE_TAG=v1 ./build-image.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

IMAGE_NAME="${IMAGE_NAME:-crpi-begxsocwym8a9lwq.cn-hangzhou.personal.cr.aliyuncs.com/hjbonc/one-api-tob}"
IMAGE_TAG="${IMAGE_TAG:-v202605215_01}"
CONTAINER_NAME="${CONTAINER_NAME:-one-api-tob}"
DOCKER_PLATFORM="${DOCKER_PLATFORM:-linux/amd64}"
VITE_API_BASE="${VITE_API_BASE:-}"
RUN_AFTER_BUILD=0
HOST_PORT="${HOST_PORT:-8886}"

usage() {
  cat <<'EOF'
用法: ./build-image.sh [选项]

选项:
  --run              构建完成后启动容器（映射 HOST_PORT:80）
  -p, --port PORT    宿主机端口，默认 8886（需配合 --run）
  --api-base URL     构建时注入 VITE_API_BASE（前后端分离时 API 根地址）
  -h, --help         显示帮助

环境变量:
  IMAGE_NAME         镜像仓库路径，默认阿里云 personal 仓库 hjbonc/one-api-tob
  IMAGE_TAG          标签，默认 v202605215_01
  CONTAINER_NAME     --run 时容器名，默认 one-api-tob
  DOCKER_PLATFORM    目标架构，默认 linux/amd64
  VITE_API_BASE      同 --api-base
  HOST_PORT          同 -p

示例:
  ./build-image.sh
  VITE_API_BASE=https://api.example.com ./build-image.sh
  ./build-image.sh --run -p 8886
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --run)
      RUN_AFTER_BUILD=1
      shift
      ;;
    -p|--port)
      HOST_PORT="${2:?缺少端口号}"
      shift 2
      ;;
    --api-base)
      VITE_API_BASE="${2:?缺少 API 地址}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "未知参数: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

echo ">> docker build --platform ${DOCKER_PLATFORM} -t ${FULL_IMAGE} ${SCRIPT_DIR}"
if [[ -n "$VITE_API_BASE" ]]; then
  echo ">> VITE_API_BASE=${VITE_API_BASE}"
  docker build --platform "$DOCKER_PLATFORM" \
    --build-arg "VITE_API_BASE=${VITE_API_BASE}" \
    -t "$FULL_IMAGE" .
else
  docker build --platform "$DOCKER_PLATFORM" -t "$FULL_IMAGE" .
fi

echo ">> 完成: ${FULL_IMAGE}"

if [[ "$RUN_AFTER_BUILD" -eq 1 ]]; then
  echo ">> docker run -d --rm -p ${HOST_PORT}:80 --name ${CONTAINER_NAME} ${FULL_IMAGE}"
  docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
  docker run -d --rm --platform "$DOCKER_PLATFORM" -p "${HOST_PORT}:80" --name "$CONTAINER_NAME" "$FULL_IMAGE"
  echo ">> 已启动: http://127.0.0.1:${HOST_PORT}"
  echo ">> API 反代见 nginx.conf（默认 /api/ -> 后端）"
fi

# syntax=docker/dockerfile:1.6
#
# 三阶段前端/后端分离，便于 BuildKit 按变更层重建：
#   1) build-nacos-console — Nacos 新版 Vite +（可选）Legacy 控制台，产出 /web/nacos-console/dist
#   2) build-one-api-web    — vue 主题（Ant Design Vue），产出 /web/build
#   3) builder-backend      — Go 编译 + embed 上述产物，产出 one-api 二进制
#
# 单独验证某一前端层（不产出最终运行镜像，常用于预热缓存）：
#   docker buildx build --target build-nacos-console .
#   docker buildx build --target build-one-api-web .
#
# BuildKit 缓存示例：
#   docker buildx build --load \
#     --cache-to type=local,dest=.docker-buildcache,mode=max \
#     --cache-from type=local,src=.docker-buildcache \
#     -t one-api:local .

FROM --platform=$BUILDPLATFORM node:24 AS web-toolchain-base

SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry ${NPM_REGISTRY}

# -----------------------------------------------------------------------------
# Stage 1 — Nacos 内嵌控制台（与 one-api web 源码解耦，避免改控制台前端时连带失效）
# -----------------------------------------------------------------------------
FROM web-toolchain-base AS build-nacos-console

WORKDIR /web
COPY ./third_party/nacos/console-ui-next ./nacos-console-src
COPY ./third_party/nacos/console-ui ./nacos-legacy-console-src

RUN --mount=type=cache,target=/root/.npm \
    cd /web/nacos-console-src && \
    npm ci && \
    npx tsc -b && \
    npx vite build && \
    mkdir -p /web/nacos-console && \
    rm -rf /web/nacos-console/dist && \
    cp -a /web/nacos-console-src/dist /web/nacos-console/dist

ARG NACOS_LEGACY_CONSOLE=1
ARG NACOS_UPSTREAM_BIND=third_party/nacos/nacos-upstream
RUN --mount=type=bind,source=${NACOS_UPSTREAM_BIND},target=/tmp/nacos-upstream-bind,ro \
    --mount=type=cache,target=/root/.npm \
    set -eu; \
    [ "$NACOS_LEGACY_CONSOLE" = "1" ] || exit 0; \
    cp /web/nacos-legacy-console-src/public/index.ejs /tmp/nacos-legacy-index.ejs && \
    cp -a /tmp/nacos-upstream-bind/console/src/main/resources/static/console-ui/public/. /web/nacos-legacy-console-src/public/ && \
    cp /tmp/nacos-legacy-index.ejs /web/nacos-legacy-console-src/public/index.ejs && \
    cd /web/nacos-legacy-console-src && \
    npm ci && \
    npm run build:embed && \
    mkdir -p /web/nacos-console/dist/legacy && \
    cp -a dist/. /web/nacos-console/dist/legacy/ && \
    rm -rf /web/nacos-legacy-console-src/node_modules /web/nacos-legacy-console-src/dist

# -----------------------------------------------------------------------------
# Stage 2 — One API 自带 Web UI（多主题）
# -----------------------------------------------------------------------------
FROM web-toolchain-base AS build-one-api-web

WORKDIR /web
COPY ./VERSION .
COPY ./web .

# 可选构建期代理（仅本步引用，避免污染其它层缓存）；DNS 不通的内网环境用它拉 npm 依赖。
ARG BUILD_PROXY=
RUN --mount=type=cache,target=/root/.npm \
    HTTPS_PROXY="${BUILD_PROXY}" HTTP_PROXY="${BUILD_PROXY}" npm install --prefix /web/vue

RUN --mount=type=cache,target=/root/.npm \
    V="$(cat ./VERSION)" && \
    export DISABLE_ESLINT_PLUGIN=true VITE_APP_VERSION="$V" && \
    npm run build --prefix /web/vue

RUN shopt -s nullglob && \
    for theme in vue; do \
      [[ -f "/web/build/${theme}/index.html" ]] || { echo "missing /web/build/${theme}/index.html"; exit 1; }; \
      bundles=(/web/build/"${theme}"/static/js/*.js); \
      ((${#bundles[@]} >= 1)) || { echo "missing JS bundles for ${theme}"; ls -la "/web/build/${theme}/static" || true; exit 1; }; \
    done

# -----------------------------------------------------------------------------
# Stage 3 — Go 后端（合并 embed 产物）
# -----------------------------------------------------------------------------
FROM golang:alpine AS builder-backend

# 纯 Go 构建（CGO_ENABLED=0）：sqlite 走 glebarez（纯 Go），生产二进制无需 cgo，
# 故无需 gcc/musl-dev 等工具链，避免受限网络下 apk 联网失败。
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOMODCACHE=/go/pkg/mod \
    GOCACHE=/root/.cache/go-build

WORKDIR /build

# 依赖通过 vendor/ 固化，构建期不拉取 Go 模块（适配受限网络）；go 命令统一走 -mod=vendor。
ARG BUILD_PROXY=
ENV GOFLAGS=-mod=vendor

COPY . .
COPY --from=build-one-api-web /web/build ./web/build
COPY --from=build-nacos-console /web/nacos-console/dist ./web/nacos-console/dist

# tiktoken 编码表仍需联网下载（Azure blob），经构建期代理；模块本身来自 vendor。
ENV TIKTOKEN_CACHE_DIR=/build/tiktoken-cache
RUN --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /build/tiktoken-cache && \
    HTTPS_PROXY="${BUILD_PROXY}" HTTP_PROXY="${BUILD_PROXY}" go run ./cmd/prefetch-tiktoken

RUN --mount=type=cache,target=/root/.cache/go-build \
    BUILD_TS=$(date -u +%Y%m%d%H%M%S) && \
    go build -trimpath -tags timetzdata -ldflags "-s -w -X github.com/songquanpeng/one-api/common.Version=$(cat VERSION) -X github.com/songquanpeng/one-api/common.BuildID=${BUILD_TS}" -o one-api

FROM debian:13 AS runtime

# 用 Debian 原生 apt 安装运行期所需：ca-certificates（上游 TLS）、tzdata（时区）、
# wget（docker-compose 健康检查）。apt 遵循 http(s)_proxy，故受限网络下经构建期代理安装。
ARG BUILD_PROXY=
RUN http_proxy="${BUILD_PROXY}" https_proxy="${BUILD_PROXY}" apt-get update && \
    http_proxy="${BUILD_PROXY}" https_proxy="${BUILD_PROXY}" apt-get install -y --no-install-recommends \
        ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder-backend /build/one-api /
COPY --from=builder-backend /build/tiktoken-cache /tiktoken-cache

ENV TIKTOKEN_CACHE_DIR=/tiktoken-cache

EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-api"]

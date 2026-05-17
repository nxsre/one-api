# syntax=docker/dockerfile:1.6
# BuildKit 缓存：npm(/root/.npm)、Go module(/go/pkg/mod)、Go build(/root/.cache/go-build)
# 本地导出/导入缓存示例：
#   docker buildx build --load \
#     --cache-to type=local,dest=.docker-buildcache,mode=max \
#     --cache-from type=local,src=.docker-buildcache \
#     -t one-api:local .

FROM --platform=$BUILDPLATFORM docker.m.daocloud.io/library/node:24 AS builder

# 使用 bash：并行 npm build + POSIX wait 可能吞掉失败退出码，镜像仍「构建成功」但 embed 里是半成品或旧产物。
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry ${NPM_REGISTRY}

WORKDIR /web
COPY ./VERSION .
COPY ./web .
COPY ./third_party/nacos/console-ui-next ./nacos-console-src
COPY ./third_party/nacos/console-ui ./nacos-legacy-console-src

# 新版 Nacos 控制台（Vite）
RUN --mount=type=cache,target=/root/.npm \
    cd /web/nacos-console-src && \
    npm ci && \
    npx tsc -b && \
    npx vite build && \
    mkdir -p /web/nacos-console && \
    rm -rf /web/nacos-console/dist && \
    cp -a /web/nacos-console-src/dist /web/nacos-console/dist

# Legacy console: merge third_party/nacos/console-ui with Nacos repo static assets from bind mount.
# Populate third_party/nacos/nacos-upstream (e.g. ./scripts/ensure-nacos-upstream.sh) before build, or:
#   docker build --build-arg NACOS_UPSTREAM_BIND=path/under/context .
# Skip: docker build --build-arg NACOS_LEGACY_CONSOLE=0 .
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

RUN --mount=type=cache,target=/root/.npm \
    npm install --prefix /web/default && \
    npm install --prefix /web/berry && \
    npm install --prefix /web/air

RUN --mount=type=cache,target=/root/.npm \
    V="$(cat ./VERSION)" && \
    export DISABLE_ESLINT_PLUGIN=true REACT_APP_VERSION="$V" && \
    npm run build --prefix /web/default && \
    npm run build --prefix /web/berry && \
    npm run build --prefix /web/air

# 任一主题构建失败或未产出 bundle 时立刻失败，避免 silent bad embed
RUN shopt -s nullglob && \
    for theme in default berry air; do \
      [[ -f "/web/build/${theme}/index.html" ]] || { echo "missing /web/build/${theme}/index.html"; exit 1; }; \
      bundles=(/web/build/"${theme}"/static/js/*.js); \
      ((${#bundles[@]} >= 1)) || { echo "missing JS bundles for ${theme}"; ls -la "/web/build/${theme}/static" || true; exit 1; }; \
    done

FROM golang:alpine AS builder2

RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite-dev \
    build-base

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux \
    GOMODCACHE=/go/pkg/mod \
    GOCACHE=/root/.cache/go-build

WORKDIR /build

ADD go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .
COPY --from=builder /web/build ./web/build
COPY --from=builder /web/nacos-console/dist ./web/nacos-console/dist

ENV TIKTOKEN_CACHE_DIR=/build/tiktoken-cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /build/tiktoken-cache && go run ./cmd/prefetch-tiktoken

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    BUILD_TS=$(date -u +%Y%m%d%H%M%S) && \
    go build -trimpath -ldflags "-s -w -X github.com/songquanpeng/one-api/common.Version=$(cat VERSION) -X github.com/songquanpeng/one-api/common.BuildID=${BUILD_TS} -linkmode external -extldflags '-static'" -o one-api

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder2 /build/one-api /
COPY --from=builder2 /build/tiktoken-cache /tiktoken-cache

ENV TIKTOKEN_CACHE_DIR=/tiktoken-cache

EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-api"]

# syntax=docker/dockerfile:1.6
# BuildKit 缓存：npm(/root/.npm)、Go module(/go/pkg/mod)、Go build(/root/.cache/go-build)
# 本地导出/导入缓存示例：
#   docker buildx build --load \
#     --cache-to type=local,dest=.docker-buildcache,mode=max \
#     --cache-from type=local,src=.docker-buildcache \
#     -t one-api:local .

FROM --platform=$BUILDPLATFORM docker.m.daocloud.io/library/node:24 AS builder

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

# 旧版控制台（Vite，与 console-ui-next 一致）：合并 Nacos 官方 public 资源后打入 dist/legacy（供 /nacos-ui/legacy/）
# 需网络拉取 Nacos 仓库；内网可构建前 ARG NACOS_GIT_REPO 指向镜像；不需要旧版时 ARG NACOS_LEGACY_CONSOLE=0
ARG NACOS_LEGACY_CONSOLE=1
ARG NACOS_GIT_REPO=https://github.com/alibaba/nacos.git
RUN --mount=type=cache,target=/root/.npm \
    if [ "$NACOS_LEGACY_CONSOLE" != "1" ]; then \
      echo "Skipping Nacos legacy console (NACOS_LEGACY_CONSOLE!=1)"; \
    else \
      apt-get update && apt-get install -y --no-install-recommends git ca-certificates && \
      rm -rf /var/lib/apt/lists/* && \
      git clone --depth 1 "$NACOS_GIT_REPO" /tmp/nacos-legacy && \
      cp /web/nacos-legacy-console-src/public/index.ejs /tmp/nacos-legacy-index.ejs && \
      cp -a /tmp/nacos-legacy/console/src/main/resources/static/console-ui/public/. /web/nacos-legacy-console-src/public/ && \
      cp /tmp/nacos-legacy-index.ejs /web/nacos-legacy-console-src/public/index.ejs && \
      cd /web/nacos-legacy-console-src && \
      npm ci && \
      npm run build:embed && \
      mkdir -p /web/nacos-console/dist/legacy && \
      cp -a dist/. /web/nacos-console/dist/legacy/ && \
      rm -rf /tmp/nacos-legacy /web/nacos-legacy-console-src/node_modules /web/nacos-legacy-console-src/dist; \
    fi

RUN --mount=type=cache,target=/root/.npm \
    npm install --prefix /web/default & \
    npm install --prefix /web/berry & \
    npm install --prefix /web/air & \
    wait

RUN --mount=type=cache,target=/root/.npm \
    DISABLE_ESLINT_PLUGIN='true' REACT_APP_VERSION=$(cat ./VERSION) npm run build --prefix /web/default & \
    DISABLE_ESLINT_PLUGIN='true' REACT_APP_VERSION=$(cat ./VERSION) npm run build --prefix /web/berry & \
    DISABLE_ESLINT_PLUGIN='true' REACT_APP_VERSION=$(cat ./VERSION) npm run build --prefix /web/air & \
    wait

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
    go build -trimpath -ldflags "-s -w -X 'github.com/songquanpeng/one-api/common.Version=$(cat VERSION)' -linkmode external -extldflags '-static'" -o one-api

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder2 /build/one-api /
COPY --from=builder2 /build/tiktoken-cache /tiktoken-cache

ENV TIKTOKEN_CACHE_DIR=/tiktoken-cache

EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-api"]

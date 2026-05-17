# 端到端测试（E2E）

依赖 **Docker Compose** 拉起 Mock 上游、MySQL、Redis 与 one-api，见仓库根目录：

- `docker-compose.e2e.yml` — 单机 one-api（宿主机 `127.0.0.1:13000`）
- `docker-compose.e2e.cluster.yml` — 双 one-api + Nginx（`127.0.0.1:28080`）

## 运行

```bash
docker compose -f docker-compose.e2e.yml build
docker compose -f docker-compose.e2e.yml up -d

go test -tags=e2e -v ./tests/e2e/ -timeout 15m
```

集群模式下：

```bash
docker compose -f docker-compose.e2e.cluster.yml up -d
E2E_ONE_API_URL=http://127.0.0.1:28080 go test -tags=e2e -v ./tests/e2e/ -timeout 15m
```

## 说明

- 测试文件使用构建约束 **`//go:build e2e`**，默认 `go test ./...` **不会**编译/执行，避免未起栈时失败。
- 环境变量（可选）：
  - `E2E_ONE_API_URL` — one-api 根地址
  - `E2E_MOCK_STATS_URL` — Mock 统计接口（默认 `http://127.0.0.1:18080/__debug/stats`）
  - `E2E_MOCK_UPSTREAM_URL` — **容器内**渠道 Base URL（默认 `http://mock-upstream:8080`，与 compose 服务名一致）
  - `E2E_OPENAI_MODEL` — OpenAI 渠道模型名（默认 `gpt-4o-mini`）
  - `E2E_ROOT_ACCESS_TOKEN` — 根用户系统访问令牌（默认与 `tests/e2e/config.e2e.toml` 里 `initial_root_access_token` 相同）。Bootstrap 时优先用该令牌调管理 API（请求头未带 `Authorization` 时由 HTTP 客户端自动注入）；无效时回退 `root` / `123456` 会话登录。

**注意：** `initial_root_access_token` 只在**首次**创建 root 用户时写入数据库。若复用已有 MySQL 数据卷且历史上 root 的 `access_token` 与当前配置不一致，访问令牌校验会失败，测试会尝试密码登录；若密码也不对，需清空卷（`docker compose ... down -v`）或在库中手动更新 `access_token`。

Mock 服务源码在 `mock_upstream/`。

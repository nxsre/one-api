# One API · ToB 控制台

面向 toB 客户的独立前端：仅改 UI/交互，业务 API 与 [one-api](https://github.com/songquanpeng/one-api) 后端一致。

样式参考：`/Users/judith/workspace/DFGX/token/index.html`（TokenHub 控制台 mockup）。

## 功能菜单

| 菜单 | 路由 | default 主题对照 |
|------|------|------------------|
| 登录 (MFA) | `/login` | `LoginForm` |
| 概览 | `/overview` | `Dashboard` |
| 模型广场 | `/models` | `Channel` |
| 用量统计 | `/usage` | `PlatformReports` |
| 日志 | `/logs` | `Operation` |
| API KEY | `/api-keys` | `Token` |
| 个人设置 | `/settings` | `Setting` |

## 本地开发

```bash
cd web/tob
npm install

# 先启动 one-api 后端（默认 3000）
# 仓库根目录: ./one-api --port 3000

npm run dev
```

浏览器打开 http://localhost:5173 。Vite 将 `/api` 代理到 `.env` 里的 `VITE_API_PROXY`。

**若出现 `http proxy error: ECONNREFUSED`**：说明代理目标没有服务在监听。请任选其一：

1. **本机启动后端**（仓库根目录）
   `./one-api --port 3000`
   `.env` 设为：`VITE_API_PROXY=http://127.0.0.1:3000`

2. **Docker Compose**（`docker-compose.yml` 映射 `13000:3000`）
   `docker compose up -d one-api`
   `.env` 设为：`VITE_API_PROXY=http://127.0.0.1:13000`

3. **远程后端**
   `.env` 设为可达地址，例如 `VITE_API_PROXY=https://101.254.166.7:13443`（需 VPN/防火墙放行）

修改 `.env` 后必须**重启** `npm run dev`。

生产构建若前后端分离，设置：

```bash
VITE_API_BASE=https://your-one-api.example.com npm run build
```

## Docker

默认镜像（`build-image.sh` 内置）：

```text
crpi-begxsocwym8a9lwq.cn-hangzhou.personal.cr.aliyuncs.com/hjbonc/one-api-tob:v202605215_01
```

```bash
cd web/tob
./build-image.sh              # 仅构建上述镜像
./build-image.sh --run        # 构建并启动，http://127.0.0.1:8886
./build-image.sh --run -p 9000

# 自定义标签
IMAGE_TAG=v20260527_01 ./build-image.sh

# 前后端分离：构建时写入 API 根地址
VITE_API_BASE=https://your-one-api.example.com ./build-image.sh
```

推送阿里云（需先 `docker login` 对应 registry）：

```bash
docker push crpi-begxsocwym8a9lwq.cn-hangzhou.personal.cr.aliyuncs.com/hjbonc/one-api-tob:v20260527_01
```

等价命令：

```bash
docker build -t crpi-begxsocwym8a9lwq.cn-hangzhou.personal.cr.aliyuncs.com/hjbonc/one-api-tob:v202605215_01 .
docker run -d --rm -p 13442:80 --name one-api-tob \
  crpi-begxsocwym8a9lwq.cn-hangzhou.personal.cr.aliyuncs.com/hjbonc/one-api-tob:v20260527_01
```

nginx 将 `/api/` 反代到后端，见 `nginx.conf`。与 compose 联调时，把 `proxy_pass` 改成你的服务名或地址。

## 技术栈

- React 18 + Vite 6 + React Router 6
- Axios（与 default 相同的 `/api/*` 会话）
- Docker + nginx 静态托管

## 后续迭代

1. 按页迁移 `web/default` 中对应页面的数据逻辑（或抽共享 hooks）
2. 按 mockup 补全模型卡片、图表、表格样式
3. 可选：在 `common/config/config.go` 注册 `tob` 主题并纳入 `web/build.sh`

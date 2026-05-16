# One API — Kubernetes 部署指南

本文描述在 Kubernetes 上**生产可用**的推荐部署方式：配置以 **`config.toml`（viper）** 为准，采用 **「主实例 1 + 从实例 N」**，避免多副本同时跑数据库迁移，并给出清单示例目录 `k8s/example/`。

## 1. 架构与角色

| 组件 | 作用 |
|------|------|
| **主实例（primary）** | `node_type` **留空**或 **`master`**。启动时执行 **GORM AutoMigrate**（主库结构变更只应发生在这里）。 replicas **固定为 1**。 |
| **从实例（worker）** | `node_type = "slave"`。**不执行** `migrateDB` / 日志库迁移逻辑，仅做 API 与业务；可按负载 **水平扩容**。 |
| **数据库** | 多实例 **共用同一 `sql_dsn`**（推荐托管 MySQL/PostgreSQL 或集群内高可用中间件）。 |
| **Redis** | **共用同一 `redis_conn_string`**，用于会话、限流、缓存等一致性。 |
| **配置** | 进程参数：`/one-api --config /etc/one-api/config.toml`（镜像 `ENTRYPOINT` 已为 `/one-api`）。 |

**不推荐**：一个 Deployment 多副本且全部为「主」语义——每个 Pod 启动都会尝试迁移，数据库上可能出现锁竞争或升级抖动。若暂时只能单 Deployment，请接受该风险，或改为单副本 + HPA 谨慎扩容。

**不要**把所有副本都设为 `slave`：否则**无人执行迁移**，除非你另有单独的迁移 Job/流程。

### 从实例与管理后台（可选）

从节点可配置 `frontend_base_url`，将浏览器侧请求重定向到**单独托管的前端**；主节点在代码中会忽略该项并始终使用内置 Web 控制台。多机时常见：** Ingress 仅把 `/v1` 等 API 指向 worker Service，控制台只暴露到主实例 Service**。

---

## 2. 前置条件

- 已构建并推送到镜像仓库的 **one-api** 镜像（可参考仓库根目录 `Dockerfile`）。
- 可用的 **MySQL 8+** 或 **PostgreSQL**（与 `sql_dsn` 一致）。
- 可用的 **Redis**（与 `redis_conn_string` 一致）。
- 集群可拉取镜像；若使用私有仓库需配置 `imagePullSecrets`。

---

## 3. 配置与安全

### 3.1 `config.toml` 要点

- 键名均为 **snake_case**，完整说明见仓库根目录 **`config.example.toml`**。
- 主/从两份配置内容绝大部分相同，**唯一差异**为：
  - 主：`node_type = ""` 或省略；  
  - 从：`node_type = "slave"`。
- 容器内建议统一：
  - `port = 3000`（与 Service、`httpGet` 探针一致）；
  - `log_dir = "/data/logs"`（见下方挂载卷，避免只读文件系统写失败）。

### 3.2 敏感信息

- **`sql_dsn`、`session_secret`、Redis 密码、`login_password_rsa_private_key`** 等须走 **Secret**（或 SealedSecrets / External Secrets / SOPS），**不要**写进 ConfigMap 明文。
- 示例清单采用 **一个 Secret 内两个键**：`primary.toml`、`worker.toml`，由 Deployment 以 `subPath` 挂成同一路径 `/etc/one-api/config.toml`。你也可以拆成两个 Secret，由 CI 注入。

### 3.3 时区

容器时区建议使用环境变量（**不属于** viper 配置项）：

```yaml
env:
  - name: TZ
    value: Asia/Shanghai
```

---

## 4. 健康检查

公开发行接口 **`GET /api/status`** 返回 JSON 且 `success: true`，适合作为 **就绪 / 存活探针**（无需鉴权）。

示例：

```yaml
readinessProbe:
  httpGet:
    path: /api/status
    port: http
  initialDelaySeconds: 15
  periodSeconds: 10
livenessProbe:
  httpGet:
    path: /api/status
    port: http
  initialDelaySeconds: 60
  periodSeconds: 30
```

根据冷启动与数据库延迟调整 `initialDelaySeconds`。

---

## 5. 资源与调度

- **requests/limits**：按并发与模型路由开销设置；无通用值，建议压测后设定。至少为 Go 运行时与连接池预留内存。
- **PodDisruptionBudget**：对 **worker** Deployment 设置 `minAvailable` 或 `maxUnavailable`，避免节点维护时清空所有从实例。
- **亲和性**：主实例与工作实例可拆到不同池子；非必须。

---

## 6. 网络与 Ingress

- **Service**：`ClusterIP`，端口例如 **3000 → 3000**，名称区分 `one-api-primary`、`one-api-worker` 或合并为统一 Service（selector 只选 worker + 主则各一套更清晰）。
- **Ingress**：
  - **仅 API**：把 `/v1`、`/v1beta` 等前缀指向 **worker** Service；
  - **控制台**：将管理路径指向 **primary** Service，或只开内网 `ClusterIP` + `kubectl port-forward`。
- **TLS**：Ingress 终止 TLS 或使用 **cert-manager**；应用内 `tls_cert_file` 多用于裸 Pod/边缘无 Ingress 的场景。

### 6.1 `secret-config.yaml` 中的应用内 TLS 示例

示例清单 **`k8s/example/secret-config.yaml`** 在 `primary.toml` / `worker.toml` 内已包含 **`tls_cert_file`、`tls_key_file`、`https_only`、`https_port`** 的默认空值与注释示例（与仓库根目录 `config.example.toml` 语义一致）。若启用进程内 HTTPS：

1. 将证书放入 Kubernetes **Secret**（如 `kubernetes.io/tls`），在 Deployment 中 **volumeMount** 到与配置一致的路径（示例中为 `/etc/one-api/tls/`）。
2. 编辑该 Secret 中的 TOML，取消注释或填入正确的 `tls_cert_file` / `tls_key_file`。
3. 若 **`https_only = true`** 且 **`port`** 改为 **443**（或仅暴露 HTTPS），需同步修改 Deployment 的 **端口、Service 与探针**（例如 `readinessProbe.httpGet.scheme: HTTPS`）。

主从两份 TOML 的 TLS 段通常相同；证书卷可同时挂到 primary 与 worker。

---

## 7. 滚动发布与数据库升级

1. 先完成**主库备份**。  
2. 发布带新迁移的镜像时，建议 **先让 primary 上线成功**（迁移跑完）再扩大 worker 副本，或短时间将 worker 副本数将为 0 再恢复（根据业务容忍度）。  
3. 若需 **严格单次迁移**，长期应引入 **单独 `migrate` 子命令 + K8s Job**；当前上游主线为启动时迁移，以 **单主实例** 为准最稳妥。

---

## 8. 示例清单（仓库内）

目录 **`k8s/example/`** 提供：

- `namespace.yaml`
- `secret-config.yaml` — **务必替换**占位 DSN、密钥后再应用；内含可选的 **应用内 TLS**（`tls_cert_file` / `tls_key_file` / `https_only` / `https_port`）示例与注释（切勿提交真实秘密到 Git）。
- `deployment-primary.yaml`
- `deployment-worker.yaml`
- `service.yaml`
- `ingress.yaml` — 可选，按集群 Ingress Class 修改。
- `poddisruptionbudget-worker.yaml` — 可选，避免维护窗口内 worker 全部被驱逐。

应用顺序示例：

```bash
kubectl apply -f k8s/example/namespace.yaml
# 编辑 secret-config.yaml 后：
kubectl apply -f k8s/example/secret-config.yaml
kubectl apply -f k8s/example/deployment-primary.yaml
kubectl apply -f k8s/example/deployment-worker.yaml
kubectl apply -f k8s/example/service.yaml
kubectl apply -f k8s/example/poddisruptionbudget-worker.yaml
# 可选：
kubectl apply -f k8s/example/ingress.yaml
```

将各文件中的 **`YOUR_REGISTRY/one-api:YOUR_TAG`** 换成实际镜像。

---

## 9. 校验

```bash
# Primary / Worker Pod 就绪
kubectl get pods -n one-api

# 探针与健康（替换 Pod 名与端口转发方式）
kubectl port-forward -n one-api pod/one-api-primary-xxxx 3000:3000
curl -s http://127.0.0.1:3000/api/status | head
```

期望响应中含 `"success": true`。

---

## 10. 常见问题

**Q：能否只用一个 Deployment、多个副本？**  
可以，但不「最合理」：多实例会重复执行迁移逻辑。更稳妥是 **primary Deployment(replicas=1) + worker Deployment(replicas=N)**。

**Q：从实例能否不配 `frontend_base_url`？**  
可以。不配置时从实例也会挂载内置前端；多机时常用 **统一外链前端** 减少重复静态资源。

**Q：SQLite 能在 K8s 里用吗？**  
不推荐多副本共享 SQLite；多实例请使用 **网络数据库**（MySQL/PostgreSQL 等）。

---

## 11. 与 docker-compose 的对照

本地 `docker-compose.yml` 使用 `config.docker.toml` 挂载为 `/app/config.toml`。K8s 中路径可自定义，只要与 **`--config` 路径一致**；示例使用 **`/etc/one-api/config.toml`**。

本文与仓库 **`config.example.toml`**、`config.docker.toml` 及 **`common/cfg`** 行为一致；键名已全部为小写 snake_case。

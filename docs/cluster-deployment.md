# One API 集群部署说明与注意事项

本文说明在 **多台 one-api 进程** 同时对外提供服务时（前置 **Kubernetes Service / NGINX / LVS / 云负载均衡** 等）的架构要求、配置同步与 **智能路由 / 模型限速** 等行为，适用于裸机、虚拟机与容器集群。

更具体的 **Kubernetes 清单与主从实例划分** 见 **[kubernetes-deployment.md](./kubernetes-deployment.md)**。

---

## 1. 总体架构

| 要点 | 说明 |
|------|------|
| **数据库** | 多实例必须 **共用同一** `SQL_DSN`（MySQL / PostgreSQL 等）。**不要**在多副本场景使用 SQLite。 |
| **Redis** | 生产环境强烈建议 **所有实例使用同一 `REDIS_CONN_STRING`（同一逻辑 Redis）**，否则限流、熔断、会话与部分缓存行为在实例间 **不一致**。 |
| **入口** | 上一层 LB 只做转发即可；**无需**为「模型限速」等能力单独配置会话粘滞（见第 4 节）。 |
| **主从角色** | `NODE_TYPE`：`slave` 的节点不执行库结构迁移；至少保留一台主语义实例跑迁移。细节见 K8s 部署文档。 |

---

## 2. Redis 在集群中的角色（必读）

源码在启用 `REDIS_CONN_STRING` 时会打开 Redis，并会将 **`MEMORY_CACHE_ENABLED` 视为开启**（与单机文档中「可关闭内存缓存」的表述在「已连 Redis」前提下由启动逻辑统一）。

| 能力 | 是否依赖「多实例共用一个 Redis」 |
|------|----------------------------------|
| **模型限速（`ModelRateLimitPolicy`）** | **是**。计数在 Redis 中；**未配置 Redis 时，该策略在代码路径上不会按集群统计，相当于对模型维度的 QPS/并发/日配额不起作用。** |
| **Redis 熔断、自适应权重等路由观测状态** | **是**。 |
| **会话 / 部分限流键** | **是**（与官方 Redis 用法一致）。 |

因此：**若要在 LB 后对全集群统一限速、统一熔断语义，必须让所有 one-api 连到同一套 Redis**（单机、哨兵或集群模式均可，只要连接串指向同一数据视图）。

**不推荐**：每台应用机器各装一个 **独立** Redis 且互不相同——会导致计数与状态分裂。

---

## 3. 配置与数据在内存中的同步

### 3.1 系统选项（含路由相关 JSON）

下列内容来自数据库 `options` 表，载入各进程内存 **`OptionMap`**（并周期性刷新），包括但不限于：

- `RoutingPolicy`（智能路由核心参数）
- `RelayRetryPolicy`
- `ModelAliasPolicy`
- `ModelRateLimitPolicy`（限速 **规则 JSON**；实际计数仍见第 2 节 Redis）

**后台保存选项**时：请求由 **某一实例** 处理，该实例会 **写库并立即更新本机内存**；**其余实例**最长滞后约 **`SYNC_FREQUENCY`（秒）** 才从数据库拉齐。

建议生产将 **`SYNC_FREQUENCY`** 设为 **60～300**（在 DB 压力与「改配置后全集群一致延迟」之间权衡）。关键变更后如需立即对齐，可对 worker 做一次 **滚动重启**（可选）。

### 3.2 渠道列表与 `model_mapping` 等（内存渠道缓存）

在开启内存渠道缓存时，启用节点会按 **`SYNC_FREQUENCY`** 周期执行 **`InitChannelCache`**，从数据库重建 **分组→模型若干渠道** 的索引。

- **新增/编辑/删除渠道** 写入数据库后，其他实例上的列表 **不会瞬间更新**，通常最长等待 **一个同步周期**。
- 若使用「固定渠道 ID」类路径，选路会 **`GetChannelById` 直读数据库**，与周期缓存的延迟特性不同，以代码为准。

---

## 4. 负载均衡（NGINX / LVS / Ingress）

- **API 面向 OpenAI 兼容客户端时一般为无状态**：下一跳落到任意实例均可。
- **模型限速、熔断、Redis 中的路由状态**：依赖 **共享 Redis**，与 **NGINX/LVS 是否粘滞会话** 无关；通常 **不需要** 为限速单独开 sticky。
- **一致性哈希选路**：策略仍在各进程内解析；候选渠道列表依赖 **共享 DB + 同步周期**，请在变更渠道后留意短暂不一致窗口。

健康检查可使用 **`GET /api/status`**（`success: true`），与 K8s 部署文档中的探针一致。

---

## 5. 环境变量与配置汇总（集群相关）

| 变量 / 配置 | 说明 |
|-------------|------|
| `SQL_DSN` | 全实例相同。 |
| `REDIS_CONN_STRING` | 全实例相同（推荐）。 |
| `SESSION_SECRET` | 多机必须相同，否则登录态无法共享（见 README）。 |
| `SYNC_FREQUENCY` | 建议显式设置（秒）；默认若未改代码层面可能较大，以运行环境为准。 |
| `NODE_TYPE` | 从节点设为 `slave`；至少一台主语义跑迁移。 |
| `MEMORY_CACHE_ENABLED` | 与 Redis 联用时以启动逻辑为准；集群请依赖 **Redis + SYNC_FREQUENCY**。 |
| `FRONTEND_BASE_URL` | 从节点可将页面指向统一前端（可选）。 |

完整键名参见仓库根目录 **`config.example.toml`**。

---

## 6. 运维与安全注意事项

1. **密钥**：`SQL_DSN`、`SESSION_SECRET`、Redis 密码等勿提交 Git；K8s 使用 Secret。  
2. **发布顺序**：带数据库迁移的版本建议 **先保证主实例成功启动** 再扩 worker（见 K8s 文档）。  
3. **监控**：LB 后端实例数变化时，关注 Redis 连接数、DB 连接池与 `SYNC_FREQUENCY` 是否满足 SLA。  
4. **排查「改策略不生效」**：确认是否命中「未落库 / 只更新了当前 Pod / 其他 Pod 未到同步周期」。

---

## 7. 相关文档

- [kubernetes-deployment.md](./kubernetes-deployment.md) — Kubernetes 示例清单、主从 Deployment、Ingress、PDB  
- 仓库根目录 [README.md](../README.md) — Docker、环境与常见问题  
- [API.md](./API.md) — 管理 API  

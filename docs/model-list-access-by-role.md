# 模型列表的获取入口与按角色的过滤规则

本文说明 **普通用户 / 租户下的普通用户 / 租户管理员 / 平台管理员** 分别从哪个接口
获取「可用模型列表」，以及各自的过滤条件。所有引用均标注了 `文件:行号`，便于核对。

> 适用范围：多租户（Tenant）部署。单租户/无租户场景只涉及「普通用户」与「平台管理员」。

## 1. 角色模型

用户角色常量（`model/user.go:22-30`）：

| 常量 | 值 | 含义 |
| --- | --- | --- |
| `RoleGuestUser` | 0 | 访客 |
| `RoleCommonUser` | 1 | 普通用户（平台用户或租户子账号都用这个角色） |
| `RoleAdminUser` | 10 | 平台管理员 |
| `RoleTenantAdmin` | 20 | 租户管理员（仅能用租户控制台 API） |
| `RoleSuperAdmin` / `RoleRootUser` | 50 / 100 | 超管 / root |

是否属于某租户由 `User.TenantID *int` 决定：`nil` = 平台级用户；非空 = 租户内用户。

关键判定函数：

- `IsAdmin(userId)`（`model/user.go:513-531`）：`role >= RoleAdminUser` **且** `TenantID == nil`。
  → **租户管理员（role=20 但 TenantID≠nil）不算 `IsAdmin`**，注释明确「租户管理员不走平台管理员能力」。
- `IsTenantConsoleAdmin(userId)`（`model/user.go:534-544`）：`role == RoleTenantAdmin && TenantID != nil`。

租户子账号还可带两份白名单（`model/user.go`，`LoadTenantSubRelayRestrictions`）：

- `AllowedModels []string`：模型白名单；
- `AllowedChannelIDs []int`：渠道 ID 白名单。

这两份由**租户管理员**为子账号配置。

## 2. 三类角色查「自己」的可用模型：同一接口，不同过滤

控制台里查可用模型，三类角色用的是**同一个**接口：

```
GET /api/user/available_models        (中间件: UserAuth)
```

路由 `router/api.go:57`，实现 `controller/model.go:269 GetUserAvailableModels`。核心逻辑：

```go
if IsAdmin(id) {                                  // 平台管理员
    return ListDistinctEnabledModels()            // 全部启用模型
}
userGroup := CacheGetUserGroup(id)
models := CacheGetGroupModels(userGroup)          // ① 按用户分组取 abilities(enabled) 的模型
models = filterGroupModelsByTenantSubUser(u, userGroup, models) // ② 仅租户子账号生效
return models
```

`filterGroupModelsByTenantSubUser`（`controller/model.go:137-170`）**只在 `TenantID != nil` 且
`Role == RoleCommonUser` 时生效**，否则原样返回。

由此三类角色的差异：

### 2.1 普通用户（平台：role=1, TenantID=nil）

- 只执行 ①：`CacheGetGroupModels(user.Group)`。
- ② 因 `TenantID == nil` 直接返回。
- **结果：该分组下全部启用模型，无额外限制。**

### 2.2 租户下的普通用户（role=1, TenantID≠nil）

- ① 之后 ② 生效，在分组模型基础上做**交集**：
  1. 若设了 `AllowedChannelIDs`：与「这些渠道（租户私有 + 平台）在该分组下能提供的模型」取交集
     （`GetDistinctModelsForGroupTenantChannelIDs`）；
  2. 若设了 `AllowedModels`：再与该白名单取交集。
- **结果：`分组模型 ∩ 渠道允许 ∩ 模型白名单`。** 两份白名单为空则等同 2.1。

### 2.3 租户管理员（role=20, TenantID≠nil）

- `IsAdmin` 为 false（TenantID≠nil），走分组路径；
- ② 因 `Role != RoleCommonUser` **早退**。
- **结果：与平台普通用户一样，只按自己分组取全部启用模型，不受子账号白名单约束。**

### 2.4 平台管理员（role≥10, TenantID=nil，对照）

- `IsAdmin` 为 true → `ListDistinctEnabledModels()`，返回**全部启用模型**。

## 3. 租户管理员额外的「管理用」接口

租户管理员在租户控制台管理子账号时，另有一组接口
（路由组 `/tenant_console`，中间件 `UserAuth + TenantConsoleMemberGate + TenantAdminConsoleOnly`，
`router/api.go:311-331`）：

| 接口 | 实现 | 返回 / 过滤 |
| --- | --- | --- |
| `GET /tenant_console/meta/all_models` | `ListAllModels` (`controller/model.go:199`) | 模型目录全部启用模型，**不按租户过滤**；用于配模型时的候选下拉 |
| `GET /tenant_console/meta/model_catalog/providers` | `GetModelCatalogProviders` | 目录 provider 只读视图（`status=current AND enabled=true`，无租户过滤） |
| `GET /tenant_console/meta/model_catalog/model_ids` | `GetModelCatalogModelIDs` | 目录 model_id 只读视图（同上过滤） |
| `GET /tenant_console/users/:id/available_models` | `TenantConsoleUserAvailableModels` (`controller/tenant_console_token.go:85`) | 某**子账号**实际可用模型（= 子账号分组模型 ∩ 该子账号白名单），用于核对/设置子账号范围 |
| `GET /tenant_console/meta/channels_for_acl` | `TenantConsoleChannelsForACL` | 租户私有渠道 + 平台渠道，用于配 `AllowedChannelIDs` |

> 模型目录管理接口 `GET /api/model_catalog/`（`router/api.go:145-156`）是**平台管理员**专属
> （`AdminAuth + PlatformConsoleOnly`），租户角色无权访问，故不在本表。

## 4. 用 API Key 调用时（对外，TokenAuth）

不论哪类用户，拿令牌（sk-…）请求 OpenAI 兼容的模型列表：

```
GET /v1/models            (中间件: TokenAuth)   实现: ListModels / collectRelayAvailableModels
GET /v1/models/:model                            实现: RetrieveModel
```

过滤链（`controller/model_availability.go:11-60`）：

1. 令牌自身 `Models` 限制（若设）→ 否则用 `CacheGetGroupModels(userGroup)`（分组模型）；
2. 再过 `filterGroupModelsByTenantSubUser`（仅租户子账号生效，同 §2.2 的白名单交集）；
3. 渠道范围（`model/ability.go:QueryEnabledChannelsForGroupModel`）：
   - 平台用户（`tenantID==0`）：仅 `channels.tenant_id IS NULL`（平台渠道）；
   - 租户用户：`channels.tenant_id IS NULL OR channels.tenant_id = <本租户>`（平台 + 本租户私有渠道）。

Claude / Gemini 兼容入口同理（`router/relay.go`：`/anthropic/v1/models`、`/v1beta/models` 等，均 `TokenAuth`）。

## 5. 注意：`/api/models` 不是按用户过滤的列表

`GET /api/models`（`DashboardListModels`，`controller/model.go:191`，`router/api.go:19`）返回的是
**未过滤**的 `channelId2Models` 静态映射（渠道类型 → 模型），仅供前端展示，不代表某用户实际可用的模型。
**按用户过滤的真实可用列表请用 `GET /api/user/available_models`。**

## 速查表

| 角色 | 查自己模型用的接口 | 鉴权 | 过滤 |
| --- | --- | --- | --- |
| 普通用户（平台） | `GET /api/user/available_models` | UserAuth | 仅分组模型 |
| 租户下普通用户 | `GET /api/user/available_models` | UserAuth | 分组模型 ∩ AllowedChannelIDs ∩ AllowedModels |
| 租户管理员 | `GET /api/user/available_models` | UserAuth | 仅分组模型（不受子账号白名单约束） |
| 平台管理员 | `GET /api/user/available_models` | UserAuth | 全部启用模型 |
| 任意用户（API Key） | `GET /v1/models` 等兼容接口 | TokenAuth | 令牌模型 / 分组模型 → 租户白名单 → 渠道范围 |
| 租户管理员（管理子账号） | `GET /tenant_console/...`（见 §3） | UserAuth + 租户管理员 | 目录全量（候选）/ 指定子账号实际可用 |
| 平台管理员（目录管理） | `GET /api/model_catalog/` | AdminAuth + PlatformConsoleOnly | `status=current AND enabled=true` |

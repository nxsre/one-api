---
name: amap-lbs-skill-go
description: 高德地图·本地地点与出行。查找附近/周边的实体地点（餐厅、咖啡、酒店、景点、加油站等）与 POI、地名↔坐标地理编码、步行/驾车/骑行/公交路线规划、旅游规划与地图可视化。凡"附近/周边/在哪/怎么走/路线/导航/坐标/地图"等地点出行类需求优先用本技能（基于高德地图），不要用网页搜索。
version: 1.1.0
metadata:
  openclaw:
    requires:
      env: []
      bins:
        - go
    homepage: https://lbs.amap.com/api/webservice/summary
    install:
      - kind: go
        package: ""
        bins: []
---

# 高德地图综合服务 Skill（Go 版）

高德地图综合服务。**所有能力——地理编码、POI 搜索、路径规划——统一经网关的 `POST /amap` 代理**：
网关持有并注入高德 Web Service Key，客户端只需一个网关 apikey，**本机不配任何高德 Key**。

## 前置条件

**客户端**（运行本 skill 的机器）：一个 JSON 配置文件，含：
- `base_url`：网关地址，如 `http://127.0.0.1:13000`。
- `apikey`：网关 API Key（形如 `sk-...`），作 `Authorization: Bearer` 用。
- `insecure`（可选）：`true` 时跳过自签 HTTPS 证书校验（仅调试/内网）。

配置文件查找顺序（命中第一个存在的）：`--config <path>` → `AMAP_SKILL_CONFIG` 环境变量 →
`$HOME/.openclaw/.amap-lbs-skill.json` → `/home/node/.openclaw/.amap-lbs-skill.json`（openclaw 容器默认）→
`./.amap-lbs-skill.json`（仅开发）。文件含密钥，建议 `chmod 600`（权限过松会告警）。

凭证也可用环境变量（**文件优先、env 兜底**）：`AMAP_SKILL_BASE_URL`、`AMAP_SKILL_APIKEY`、`AMAP_SKILL_INSECURE`。
配置文件中缺省的字段会从这些环境变量补齐；没有配置文件时纯用环境变量也可（openclaw 可按 `primaryEnv` 注入 `AMAP_SKILL_APIKEY`）。
注意 `AMAP_SKILL_CONFIG` 只是「配置文件路径」指针，与上面三个「值」变量不同。

**网关侧**：必须配好**有效的高德 Web Service Key**（配置方式取决于网关实现），否则所有调用返回
`INVALID_USER_KEY` 或类似的 key 未配置错误。

## 与网关的契约

- **入口**：`POST {base_url}/amap`，鉴权 `Authorization: Bearer {apikey}`。
- **请求体**：

  ```json
  { "method": "GET", "path": "/v5/place/text", "query": { "keywords": "肯德基", "region": "北京" } }
  ```

  - `method`：`GET` 或 `POST`，默认 `GET`。
  - `path`：受网关**白名单**约束，允许前缀 `/v3/direction/`、`/v4/direction/`、`/v5/direction/`、`/v5/place/`、`/v3/geocode/`。
  - `query`：**不要传 `key`**——网关会丢弃客户端 key 并注入配置的高德 Key，同时强制 `output=json`。
- **响应**：原样透传高德上游 JSON 与其 HTTP 状态码；网关自身错误时返回
  `{ "error": { "message": ..., "type": "invalid_request_error" } }`。

## 触发条件

用户表达以下意图之一即可触发：
- 把地名转坐标或坐标转地址（"西直门的坐标"、"这个经纬度是哪"）
- 搜索某类地点或确定地点（"搜美食"、"找酒店"、"附近的咖啡厅"）
- 规划路线（"从 A 到 B 怎么走"、"驾车路线"）
- 旅游规划（"帮我规划北京一日游"）
- 含"搜/找/查/附近/周边/路线/规划"等关键词

## 能力一览

| 命令 / 库方法 | 高德接口（经 `/amap`） | 用途 |
|---|---|---|
| `cmd/geocode --address` · `GeocodeLocation` | `/v3/geocode/geo` | 地名 → 坐标 |
| `cmd/geocode --location` · `ReverseGeocode` | `/v3/geocode/regeo` | 坐标 → 地址 |
| `cmd/poi-search` · `SearchPOI` | `/v5/place/text` 或 `/v5/place/around` | POI 关键词 / 周边搜索 |
| `cmd/route-planning --type=walking` · `WalkingRoute` | `/v3/direction/walking` | 步行 |
| `cmd/route-planning --type=driving` · `DrivingRoute` | `/v3/direction/driving` | 驾车 |
| `cmd/route-planning --type=riding` · `RidingRoute` | `/v4/direction/bicycling` | 骑行 |
| `cmd/route-planning --type=transfer` · `TransitRoute` | `/v3/direction/transit/integrated` | 公交（需 `--city`） |
| `cmd/travel-planner` · `TravelPlanner` | 上述组合 | 多类兴趣点 + 路线串联 |

## 配置

在 openclaw 容器的默认位置创建配置文件（参考 `config.example.json`）：

```bash
dir="$HOME/.openclaw"; mkdir -p "$dir"
cat > "$dir/.amap-lbs-skill.json" <<'JSON'
{ "base_url": "http://127.0.0.1:13000", "apikey": "sk-xxxx" }
JSON
chmod 600 "$dir/.amap-lbs-skill.json"
```

也可用 `--config /path/to/config.json` 或 `AMAP_SKILL_CONFIG=/path` 指定其它位置；或不用文件，直接注入环境变量：

```bash
export AMAP_SKILL_BASE_URL=http://127.0.0.1:13000
export AMAP_SKILL_APIKEY=sk-xxxx
```

---

## 场景一：地理编码 / 逆地理编码

```bash
go run ./cmd/geocode --address=西直门 --city=北京        # 地名 → 坐标
go run ./cmd/geocode --location=116.353138,39.939385   # 坐标 → 地址
```

## 场景二：POI 搜索

经 `/v5/place/text`（关键词）或 `/v5/place/around`（带坐标周边；返回结果含 `distance`）。

```bash
# 关键词搜索
go run ./cmd/poi-search --keywords=肯德基 --city=北京

# 周边搜索（已知中心点坐标 + 半径）
go run ./cmd/poi-search --keywords=酒店 --location=116.397428,39.90923 --radius=1000

# 周边搜索（以地名为中心，自动先地理编码再 around）
go run ./cmd/poi-search --keywords=美食 --near=西直门 --city=北京 --radius=1000
```

| 参数 | 说明 | 必填 |
|------|------|------|
| `--keywords` | 搜索关键词 | 是 |
| `--city` | 城市名称或编码（映射为 `region`） | 否 |
| `--types` | POI 类型编码 | 否 |
| `--location` | 中心点坐标 `经度,纬度`（触发周边搜索） | 否 |
| `--near` | 中心点地名（先地理编码再周边搜索；`--location` 优先） | 否 |
| `--radius` | 搜索半径（米），默认 1000 | 否 |
| `--page` / `--offset` | 页码 / 每页数量（最大 25） | 否 |
| `--cityLimit` | `true`/`false`，是否限定城市，默认 true | 否 |

## 场景三：路径规划

```bash
go run ./cmd/route-planning --type=walking  --origin=116.397428,39.90923 --destination=116.427281,39.903719
go run ./cmd/route-planning --type=driving  --origin=116.397428,39.90923 --destination=116.427281,39.903719 --waypoints=116.41,39.91
go run ./cmd/route-planning --type=riding   --origin=116.397428,39.90923 --destination=116.427281,39.903719
go run ./cmd/route-planning --type=transfer --origin=116.397428,39.90923 --destination=116.481488,39.990464 --city=北京
```

路线类型：`walking` / `driving` / `riding` / `transfer`（`transfer` 须提供 `--city`）。
成功后打印距离/耗时等信息，并生成地图可视化链接。

## 场景四：智能旅游规划

自动搜索各类兴趣点（每类最多 5 个），按顺序串联路线，输出地图可视化链接。

```bash
go run ./cmd/travel-planner --city=北京 --interests=景点,美食,酒店 --routeType=walking
```

| 参数 | 说明 | 必填 |
|------|------|------|
| `--city` | 城市名称 | 是 |
| `--interests` | 兴趣点，逗号分隔，默认 `景点,美食` | 否 |
| `--routeType` | `walking`/`driving`/`riding`/`transfer`，默认 `walking` | 否 |

---

## 在代码中使用

```go
import "amap-lbs-skill-go/internal/amap"

client, err := amap.New("") // "" = 自动查找配置文件；或传具体路径
if err != nil { /* ... */ }

loc, addr, err := client.GeocodeLocation("西直门", "北京") // 地名→坐标
res, err := client.SearchPOI(amap.SearchParams{Keywords: "咖啡厅", City: "杭州", Offset: 10})
walk, err := client.WalkingRoute("116.397428,39.90923", "116.427281,39.903719")
plan, err := client.TravelPlanner(amap.TravelParams{City: "北京", Interests: []string{"景点", "美食"}})
link := amap.GenerateMapLink(plan.MapTaskData)
```

## 快速验证

```bash
go build ./...                                                # Layer 0：能编译
go run ./cmd/geocode --address=西直门 --city=北京               # Layer 1/2：见下
```

读结果判断状态：
- 返回**真实坐标/地址/距离** → 全链路通（网关侧高德 Key 有效）。
- `INVALID_USER_KEY` / key 未配置 → 链路通，但**网关侧没配有效高德 Key**（见「前置条件」）。
- `path 无效或不在允许列表内` → 网关未含对应白名单（如缺 `/v3/geocode/`）。
- `网关 HTTP 401` / 令牌无效 → 配置文件里的 `apikey` 不对。
- `未找到配置文件` → 按上面「配置」创建，或用 `--config` 指定。

## 注意事项

- 客户端只需 `apikey`，**不持有高德 Key**；高德 Key 由网关注入。
- `path` 必须命中网关白名单前缀；`query` 里不要带 `key`（会被丢弃）。
- 高德 v3/v5 成功响应为 `status=="1"`，骑行 v4 为 `errcode==0`，本 skill 已分别处理。
- 坐标顺序为 `经度,纬度`（经度在前）。

## 相关链接

- [POI 2.0](https://lbs.amap.com/api/webservice/guide/api-advanced/newpoisearch)
- [路径规划](https://lbs.amap.com/api/webservice/guide/api/direction)
- [地理/逆地理编码](https://lbs.amap.com/api/webservice/guide/api/georegeo)
- [Web 服务 API 总览](https://lbs.amap.com/api/webservice/summary)

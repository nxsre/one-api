# 高德地图综合服务 Skill（Go 版）

高德地图综合服务。**所有能力统一经网关的 `POST /amap` 代理**——网关注入高德 Web Service Key，
客户端只需一个网关 apikey，本机无需任何高德 Key。

## 功能特性

- ✅ 地理编码 / 逆地理编码（`/v3/geocode/geo`、`/v3/geocode/regeo`）
- ✅ POI 搜索（`/v5/place/text`、`/v5/place/around`）
- ✅ 路径规划（步行 v3、驾车 v3、骑行 v4、公交 v3）
- ✅ 智能旅游规划助手
- ✅ 地图可视化链接生成
- ✅ 纯标准库实现，无第三方依赖

## 配置

在 openclaw 容器的默认位置创建配置文件（参考 `config.example.json`）：

```bash
dir="$HOME/.openclaw"; mkdir -p "$dir"
cat > "$dir/.amap-lbs-skill.json" <<'JSON'
{ "base_url": "http://127.0.0.1:13000", "apikey": "sk-xxxx" }
JSON
chmod 600 "$dir/.amap-lbs-skill.json"
```

查找顺序：`--config <path>` → `AMAP_SKILL_CONFIG` → `$HOME/.openclaw/.amap-lbs-skill.json` →
`/home/node/.openclaw/.amap-lbs-skill.json`（openclaw 容器默认）→ `./.amap-lbs-skill.json`（开发用，已 gitignore）。
字段：`base_url`、`apikey`、`insecure`（可选，自签 HTTPS）。

也支持环境变量（**文件优先、env 兜底**）：`AMAP_SKILL_BASE_URL`、`AMAP_SKILL_APIKEY`、`AMAP_SKILL_INSECURE`——
文件缺省字段从 env 补齐；无文件时纯 env 亦可。

## 使用方法

### 1. 地理编码 / 逆地理编码

```bash
go run ./cmd/geocode --address=西直门 --city=北京        # 地名 → 坐标
go run ./cmd/geocode --location=116.353138,39.939385   # 坐标 → 地址
```

### 2. POI 搜索

```bash
go run ./cmd/poi-search --keywords=肯德基 --city=北京 --page=1 --offset=20
# 周边搜索（坐标为中心）
go run ./cmd/poi-search --keywords=酒店 --location=116.397428,39.90923 --radius=1000
# 周边搜索（地名为中心，自动地理编码）
go run ./cmd/poi-search --keywords=美食 --near=西直门 --city=北京 --radius=1000
```

### 3. 路径规划

```bash
go run ./cmd/route-planning --type=walking  --origin=116.397428,39.90923 --destination=116.427281,39.903719
go run ./cmd/route-planning --type=driving  --origin=116.397428,39.90923 --destination=116.427281,39.903719 --waypoints=116.41,39.91
go run ./cmd/route-planning --type=riding   --origin=116.397428,39.90923 --destination=116.427281,39.903719
go run ./cmd/route-planning --type=transfer --origin=116.397428,39.90923 --destination=116.481488,39.990464 --city=北京
```

### 4. 智能旅游规划

```bash
go run ./cmd/travel-planner --city=北京 --interests=景点,美食,酒店 --routeType=walking
```

### 5. 编译为二进制

```bash
go build -o bin/geocode         ./cmd/geocode
go build -o bin/poi-search      ./cmd/poi-search
go build -o bin/route-planning  ./cmd/route-planning
go build -o bin/travel-planner  ./cmd/travel-planner
```

### 6. 作为库使用

```go
import "amap-lbs-skill-go/internal/amap"

client, err := amap.New("") // "" = 自动查找配置文件；或传具体路径
loc, addr, err := client.GeocodeLocation("西直门", "北京")
res, err := client.SearchPOI(amap.SearchParams{Keywords: "咖啡厅", City: "杭州", Offset: 10})
walk, err := client.WalkingRoute("116.397428,39.90923", "116.427281,39.903719")
plan, err := client.TravelPlanner(amap.TravelParams{City: "北京", Interests: []string{"景点", "美食"}})
link := amap.GenerateMapLink(plan.MapTaskData)
```

## 网关契约

`POST {base_url}/amap`，`Authorization: Bearer {apikey}`，请求体：

```json
{ "method": "GET", "path": "/v5/place/text", "query": { "keywords": "肯德基", "region": "北京" } }
```

- `path` 白名单前缀：`/v3/direction/`、`/v4/direction/`、`/v5/direction/`、`/v5/place/`、`/v3/geocode/`。
- 不要在 `query` 里传 `key`——网关会丢弃并注入自己的高德 Key，同时强制 `output=json`。
- 响应原样透传高德上游 JSON 与状态码。

## 项目结构

```
amap-lbs-skill-go/
├── go.mod
├── internal/amap/              # 核心库（标准库实现）
│   ├── client.go               # 网关 POST /amap 客户端
│   ├── config.go               # 配置文件查找与读取（base_url/apikey/insecure）
│   ├── poi.go                  # POI 搜索
│   ├── geocode.go              # 地理编码 / 逆地理编码
│   ├── route.go                # 步行/驾车/骑行/公交
│   ├── travel.go               # 旅游规划
│   ├── maplink.go              # 地图可视化链接
│   └── util.go                 # flexString 等工具
├── cmd/
│   ├── geocode/                # 地理编码 / 逆地理编码 CLI
│   ├── poi-search/             # POI 搜索 CLI（支持 --near 地名周边搜索）
│   ├── route-planning/         # 路径规划 CLI
│   └── travel-planner/         # 旅游规划 CLI
├── config.example.json
├── SKILL.md
├── README.md
└── LICENSE
```

## License

MIT

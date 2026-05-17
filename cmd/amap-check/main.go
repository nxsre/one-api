// amap-check：通过 one-api POST /amap 检测高德代理是否与 amap-lbs-skill 所用接口一致可用。
// Skill：POI 经 POST /amap；路径规划在 skill 内直连高德，本工具统一走 /amap 复现同等 path。
//
// 服务端高德 Key：环境变量 AMAP_KEY → 选项 AmapWebServiceSecret → 首个启用的「高德」渠道密钥。
// 用户令牌：-token（Bearer）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/songquanpeng/one-api/cmd/internal/apitest"
)

const prog = "amap-check"

// 与 skills/amap-lbs-skill/index.js 中对外能力对应的检测项（经 POST /amap）。
var skillAlignedOps = []string{
	"place_text",   // searchPOIViaOneApi → /v5/place/text
	"place_around", // searchPOIViaOneApi → /v5/place/around
	"walking",      // walkingRoute → /v3/direction/walking
	"driving",      // drivingRoute → /v3/direction/driving
	"bicycling",    // ridingRoute → /v4/direction/bicycling
	"transit",      // transitRoute → /v3/direction/transit/integrated
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage: %s -token sk-... [options]

通过 one-api POST /amap 检测高德代理；检测项与 skills/amap-lbs-skill/index.js 中
searchPOI（text/around）、walkingRoute、drivingRoute、ridingRoute、transitRoute 所用路径对齐。

`, prog)
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), `
Skill 对齐单测类型（可用 -operation）：
  place_text    /v5/place/text
  place_around  /v5/place/around
  walking       /v3/direction/walking
  driving       /v3/direction/driving
  bicycling     /v4/direction/bicycling
  transit       /v3/direction/transit/integrated

扩展（非 -all 默认集）：
  place_polygon /v5/place/polygon  （需 -polygon）
  place_detail  /v5/place/detail   （需 -id）

兼容别名：text_search|text -> place_text；around -> place_around；polygon；detail
`)
	}

	base := flag.String("base", getenvDefault("ONE_API_BASE", "http://127.0.0.1:3000"), "one-api 根地址")
	token := flag.String("token", "", "用户 API 令牌 sk-…，必填")
	insecure := flag.Bool("insecure", false, "跳过 HTTPS 证书校验（调试/自签证书）")
	all := flag.Bool("all", false, "依次运行与 amap-lbs-skill 对齐的全部检测（place_text place_around walking driving bicycling transit）")
	op := flag.String("operation", "place_text", "单测类型（见 -h）；与 -all 互斥时以 -all 为准")

	keywords := flag.String("keywords", "咖啡厅", "place_text：关键字")
	region := flag.String("region", "北京", "place_text / place_around：区域或城市")
	location := flag.String("location", "116.397428,39.90923", "place_around：中心点 经度,纬度")
	polygon := flag.String("polygon", "", "place_polygon：多边形坐标串")
	id := flag.String("id", "", "place_detail：POI id（多个用 |）")

	origin := flag.String("origin", "116.397428,39.90923", "步行/驾车/骑行/公交：起点 经度,纬度")
	destination := flag.String("destination", "116.407526,39.904031", "步行/驾车/骑行/公交：终点 经度,纬度")
	transitCity := flag.String("transit_city", "北京", "transit：城市")

	flag.Parse()

	if strings.TrimSpace(*token) == "" {
		fmt.Fprintf(os.Stderr, "%s: 必须提供 -token\n", prog)
		os.Exit(2)
	}

	cli := apitest.New(*base, *token, *insecure)

	if *all {
		var failed []string
		for _, name := range skillAlignedOps {
			fmt.Printf("== %s ==\n", name)
			payload, err := buildPayload(name, payloadArgs{
				keywords: *keywords, region: *region, location: *location,
				polygon: *polygon, id: *id,
				origin: *origin, destination: *destination, transitCity: *transitCity,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: [%s] %v\n", prog, name, err)
				failed = append(failed, name)
				continue
			}
			if _, err := postAndVerify(cli, name, payload); err != nil {
				fmt.Fprintf(os.Stderr, "%s: [%s] %v\n", prog, name, err)
				failed = append(failed, name)
				continue
			}
			fmt.Printf("[%s] OK\n", name)
		}
		if len(failed) > 0 {
			fmt.Fprintf(os.Stderr, "%s: 失败项: %v\n", prog, failed)
			os.Exit(1)
		}
		fmt.Printf("%s: all OK (%d checks)\n", prog, len(skillAlignedOps))
		return
	}

	name := normalizeOp(*op)
	payload, err := buildPayload(name, payloadArgs{
		keywords: *keywords, region: *region, location: *location,
		polygon: *polygon, id: *id,
		origin: *origin, destination: *destination, transitCity: *transitCity,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
		os.Exit(2)
	}

	body, err := postAndVerify(cli, name, payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", prog, err)
		os.Exit(1)
	}

	fmt.Printf("%s: OK [%s]\n", prog, name)
	fmt.Println(truncate(string(body), 1200))
}

type payloadArgs struct {
	keywords, region, location, polygon, id string
	origin, destination, transitCity        string
}

func normalizeOp(op string) string {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "text_search", "text":
		return "place_text"
	case "around":
		return "place_around"
	case "polygon":
		return "place_polygon"
	case "detail":
		return "place_detail"
	default:
		return strings.ToLower(strings.TrimSpace(op))
	}
}

func buildPayload(op string, a payloadArgs) (map[string]any, error) {
	op = normalizeOp(op)
	q := map[string]string{}
	out := map[string]any{
		"method": "GET",
		"query":  q,
	}
	switch op {
	case "place_text":
		out["path"] = "/v5/place/text"
		kw := strings.TrimSpace(a.keywords)
		if kw == "" {
			return nil, fmt.Errorf("place_text 须非空 -keywords")
		}
		q["keywords"] = kw
		q["region"] = strings.TrimSpace(a.region)
		q["city_limit"] = "true"
		q["page_num"] = "1"
		q["page_size"] = "10"
	case "place_around":
		out["path"] = "/v5/place/around"
		if strings.TrimSpace(a.location) == "" {
			return nil, fmt.Errorf("place_around 须 -location=经度,纬度")
		}
		q["location"] = strings.TrimSpace(a.location)
		q["keywords"] = strings.TrimSpace(a.keywords)
		q["radius"] = "1000"
		if r := strings.TrimSpace(a.region); r != "" {
			q["region"] = r
		}
		q["city_limit"] = "true"
		q["page_num"] = "1"
		q["page_size"] = "10"
	case "place_polygon":
		out["path"] = "/v5/place/polygon"
		if strings.TrimSpace(a.polygon) == "" {
			return nil, fmt.Errorf("place_polygon 须 -polygon")
		}
		q["polygon"] = strings.TrimSpace(a.polygon)
		q["keywords"] = strings.TrimSpace(a.keywords)
		q["page_num"] = "1"
		q["page_size"] = "10"
	case "place_detail":
		out["path"] = "/v5/place/detail"
		if strings.TrimSpace(a.id) == "" {
			return nil, fmt.Errorf("place_detail 须 -id")
		}
		q["id"] = strings.TrimSpace(a.id)
	case "walking":
		out["path"] = "/v3/direction/walking"
		if err := fillOriginDest(q, a); err != nil {
			return nil, err
		}
	case "driving":
		out["path"] = "/v3/direction/driving"
		if err := fillOriginDest(q, a); err != nil {
			return nil, err
		}
		q["strategy"] = "10"
		q["extensions"] = "base"
	case "bicycling":
		out["path"] = "/v4/direction/bicycling"
		if err := fillOriginDest(q, a); err != nil {
			return nil, err
		}
	case "transit":
		out["path"] = "/v3/direction/transit/integrated"
		if err := fillOriginDest(q, a); err != nil {
			return nil, err
		}
		city := strings.TrimSpace(a.transitCity)
		if city == "" {
			return nil, fmt.Errorf("transit 须非空 -transit_city")
		}
		q["city"] = city
		q["strategy"] = "0"
		q["nightflag"] = "0"
	default:
		return nil, fmt.Errorf("未知 operation: %q（skill 对齐: place_text place_around walking driving bicycling transit；扩展: place_polygon place_detail；或用 -all）", op)
	}
	return out, nil
}

func fillOriginDest(q map[string]string, a payloadArgs) error {
	o := strings.TrimSpace(a.origin)
	d := strings.TrimSpace(a.destination)
	if o == "" || d == "" {
		return fmt.Errorf("须同时提供 -origin 与 -destination（经度,纬度）")
	}
	q["origin"] = o
	q["destination"] = d
	return nil
}

func postAndVerify(cli *apitest.Client, op string, payload map[string]any) ([]byte, error) {
	status, resp, err := cli.PostJSON("/amap", payload)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("HTTP %d\n%s", status, truncate(string(resp), 1200))
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(resp, &top); err != nil {
		return nil, fmt.Errorf("parse JSON: %w\nbody=%s", err, truncate(string(resp), 500))
	}
	if errRaw, ok := top["error"]; ok {
		return nil, fmt.Errorf("one-api error: %s", string(errRaw))
	}

	if err := verifyUpstreamJSON(normalizeOp(op), resp); err != nil {
		return nil, fmt.Errorf("%w\nbody=%s", err, truncate(string(resp), 800))
	}
	return resp, nil
}

// verifyUpstreamJSON：v3/v5 成功一般为 status=="1"；骑行 v4 成功为 errcode==0（与 index.js 一致）。
func verifyUpstreamJSON(op string, raw []byte) error {
	var anyJSON map[string]interface{}
	if err := json.Unmarshal(raw, &anyJSON); err != nil {
		return err
	}
	if op == "bicycling" {
		ec, ok := anyJSON["errcode"]
		if ok && errcodeIsZero(ec) {
			return nil
		}
		return fmt.Errorf("bicycling 期望 errcode=0，got %+v errmsg=%v", ec, anyJSON["errmsg"])
	}
	st, _ := anyJSON["status"].(string)
	if st == "1" {
		return nil
	}
	info, _ := anyJSON["info"].(string)
	return fmt.Errorf("高德 status=%s info=%s", st, info)
}

func errcodeIsZero(v interface{}) bool {
	switch n := v.(type) {
	case float64:
		return n == 0
	case int:
		return n == 0
	case int64:
		return n == 0
	case json.Number:
		i, err := n.Int64()
		return err == nil && i == 0
	case string:
		return strings.TrimSpace(n) == "0"
	default:
		return false
	}
}

func getenvDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// route-planning：经网关 POST /amap 做路径规划（步行/驾车/骑行/公交）。
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"amap-lbs-skill-go/internal/amap"
)

func main() {
	typ := flag.String("type", "", "路线类型: walking/driving/riding/transfer")
	origin := flag.String("origin", "", "起点坐标 经度,纬度")
	destination := flag.String("destination", "", "终点坐标 经度,纬度")
	waypoints := flag.String("waypoints", "", "途经点（驾车，; 分隔）")
	strategy := flag.Int("strategy", -1, "策略（驾车默认 10，公交默认 0）")
	city := flag.String("city", "", "城市（公交必填）")
	nightflag := flag.Bool("nightflag", false, "是否计算夜班车（公交）")
	configPath := flag.String("config", "", "配置文件路径（默认自动查找）")
	flag.Parse()

	if *typ == "" || *origin == "" || *destination == "" {
		fmt.Fprintln(os.Stderr, "❌ 缺少必需参数")
		fmt.Println("\n用法: route-planning --type=类型 --origin=起点 --destination=终点 [其他参数]")
		fmt.Println("  walking / driving / riding / transfer（transfer 需 --city）")
		os.Exit(1)
	}

	client, err := amap.New(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	oLng, oLat := splitCoord(*origin)
	dLng, dLat := splitCoord(*destination)

	var task amap.MapTask
	ok := false

	switch *typ {
	case "walking":
		r, err := client.WalkingRoute(*origin, *destination)
		exitOnErr(err)
		if len(r.Route.Paths) > 0 {
			p := r.Route.Paths[0]
			dist := atof(p.Distance.String())
			fmt.Println("📊 路线信息:")
			fmt.Printf("   距离: %.2f 公里\n", dist/1000)
			fmt.Printf("   预计时间: %d 分钟\n\n", minutes(p.Duration.String()))
			task = amap.MapTask{Type: "route", RouteType: "walking", Start: []float64{oLng, oLat}, End: []float64{dLng, dLat},
				Remark: fmt.Sprintf("步行路线，距离约 %.2f 公里", dist/1000)}
			ok = true
		}
	case "driving":
		params := amap.DrivingParams{Origin: *origin, Destination: *destination, Waypoints: *waypoints}
		if *strategy >= 0 {
			params.Strategy = *strategy
		}
		r, err := client.DrivingRoute(params)
		exitOnErr(err)
		if len(r.Route.Paths) > 0 {
			p := r.Route.Paths[0]
			dist := atof(p.Distance.String())
			fmt.Println("📊 路线信息:")
			fmt.Printf("   距离: %.2f 公里\n", dist/1000)
			fmt.Printf("   预计时间: %d 分钟\n", minutes(p.Duration.String()))
			fmt.Printf("   过路费: %s 元\n", orZero(p.Tolls.String()))
			fmt.Printf("   红绿灯: %s 个\n\n", orZero(p.TrafficLights.String()))
			task = amap.MapTask{Type: "route", RouteType: "driving", Start: []float64{oLng, oLat}, End: []float64{dLng, dLat},
				Remark: fmt.Sprintf("驾车路线，距离约 %.2f 公里", dist/1000)}
			ok = true
		}
	case "riding":
		r, err := client.RidingRoute(*origin, *destination)
		exitOnErr(err)
		if len(r.Data.Paths) > 0 {
			p := r.Data.Paths[0]
			dist := atof(p.Distance.String())
			fmt.Println("📊 路线信息:")
			fmt.Printf("   距离: %.2f 公里\n", dist/1000)
			fmt.Printf("   预计时间: %d 分钟\n\n", minutes(p.Duration.String()))
			task = amap.MapTask{Type: "route", RouteType: "riding", Start: []float64{oLng, oLat}, End: []float64{dLng, dLat},
				Remark: fmt.Sprintf("骑行路线，距离约 %.2f 公里", dist/1000)}
			ok = true
		}
	case "transfer":
		if *city == "" {
			fmt.Fprintln(os.Stderr, "❌ 公交路线规划需要提供 --city 参数")
			os.Exit(1)
		}
		strat := 0
		if *strategy >= 0 {
			strat = *strategy
		}
		r, err := client.TransitRoute(amap.TransitParams{Origin: *origin, Destination: *destination, City: *city, Strategy: strat, Nightflag: *nightflag})
		exitOnErr(err)
		fmt.Println("📊 路线信息:")
		fmt.Printf("   方案数量: %d 个\n", len(r.Route.Transits))
		if len(r.Route.Transits) > 0 {
			t := r.Route.Transits[0]
			fmt.Printf("   预计时间: %d 分钟\n", minutes(t.Duration.String()))
			fmt.Printf("   费用: %s 元\n", orZero(t.Cost.String()))
			fmt.Printf("   步行距离: %s 米\n\n", orZero(t.WalkingDistance.String()))
		}
		task = amap.MapTask{Type: "route", RouteType: "transfer", Start: []float64{oLng, oLat}, End: []float64{dLng, dLat}, City: *city,
			Remark: fmt.Sprintf("公交路线，共 %d 个方案", len(r.Route.Transits))}
		ok = true
	default:
		fmt.Fprintf(os.Stderr, "❌ 不支持的路线类型: %s\n", *typ)
		os.Exit(1)
	}

	if ok {
		fmt.Println("🗺️  地图可视化链接:")
		fmt.Println(amap.GenerateMapLink([]amap.MapTask{task}))
		fmt.Println("\n💡 提示: 复制链接到浏览器打开即可查看路线详情")
	} else {
		fmt.Println("\n❌ 路线规划失败，请检查参数是否正确")
	}
}

func splitCoord(s string) (float64, float64) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0
	}
	return atof(parts[0]), atof(parts[1])
}

func atof(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func minutes(durationSeconds string) int {
	return int(atof(durationSeconds)/60 + 0.5)
}

func orZero(s string) string {
	if strings.TrimSpace(s) == "" {
		return "0"
	}
	return s
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 执行失败: %v\n", err)
		os.Exit(1)
	}
}

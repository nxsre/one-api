// travel-planner：经网关 POST /amap 搜索兴趣点并规划游览路线。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"amap-lbs-skill-go/internal/amap"
)

func main() {
	city := flag.String("city", "", "城市名称（必填）")
	interestsArg := flag.String("interests", "", "兴趣点，逗号分隔，如 景点,美食,酒店")
	routeType := flag.String("routeType", "walking", "walking/driving/riding/transfer")
	configPath := flag.String("config", "", "配置文件路径（默认自动查找）")
	flag.Parse()

	if *city == "" {
		fmt.Fprintln(os.Stderr, "❌ 缺少必需参数: --city")
		fmt.Println("\n用法: travel-planner --city=城市名 [--interests=景点,美食] [--routeType=walking]")
		os.Exit(1)
	}

	valid := map[string]bool{"walking": true, "driving": true, "riding": true, "transfer": true}
	if !valid[*routeType] {
		fmt.Fprintf(os.Stderr, "❌ 无效的路线类型: %s\n", *routeType)
		fmt.Println("有效的路线类型: walking, driving, riding, transfer")
		os.Exit(1)
	}

	client, err := amap.New(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	var interests []string
	if strings.TrimSpace(*interestsArg) != "" {
		for _, s := range strings.Split(*interestsArg, ",") {
			if t := strings.TrimSpace(s); t != "" {
				interests = append(interests, t)
			}
		}
	}

	fmt.Printf("\n🗺️  开始为您规划 %s 的旅游行程...\n\n", *city)
	result, err := client.TravelPlanner(amap.TravelParams{City: *city, Interests: interests, RouteType: *routeType})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 执行失败: %v\n", err)
		os.Exit(1)
	}
	if len(result.Pois) == 0 {
		fmt.Println("\n❌ 未找到相关地点，请尝试更换关键词或城市。")
		return
	}

	fmt.Print("\n✅ 旅游规划完成！\n\n")
	fmt.Println("📍 推荐地点：")
	for i, poi := range result.Pois {
		fmt.Printf("%d. %s\n", i+1, poi.Name)
		fmt.Printf("   地址: %s\n", poi.Address.String())
		fmt.Printf("   类型: %s\n\n", poi.Type.String())
	}

	routeCount := 0
	for _, t := range result.MapTaskData {
		if t.Type == "route" {
			routeCount++
		}
	}
	label := map[string]string{"walking": "步行", "driving": "驾车", "riding": "骑行", "transfer": "公交"}[*routeType]
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println("\n📊 规划数据统计：")
	fmt.Printf("   兴趣点数量: %d 个\n", len(result.Pois))
	fmt.Printf("   路线数量: %d 条\n", routeCount)
	fmt.Printf("   出行方式: %s\n", label)
	fmt.Println("\n" + strings.Repeat("═", 80))

	fmt.Println("\n🗺️  地图可视化链接:")
	fmt.Println(amap.GenerateMapLink(result.MapTaskData))
}

// poi-search：经网关 POST /amap 搜索高德 POI。
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strings"

	"amap-lbs-skill-go/internal/amap"
)

func main() {
	keywords := flag.String("keywords", "", "查询关键词（必填）")
	city := flag.String("city", "", "城市名称或编码")
	types := flag.String("types", "", "POI 类型编码")
	location := flag.String("location", "", "中心点坐标 经度,纬度（提供则周边搜索）")
	near := flag.String("near", "", "中心点地名（先地理编码为坐标再周边搜索，如 西直门）")
	radius := flag.String("radius", "", "搜索半径（米，周边搜索）")
	page := flag.Int("page", 1, "页码")
	offset := flag.Int("offset", 10, "每页数量（最大 25）")
	cityLimit := flag.String("cityLimit", "", "是否限定城市 true/false")
	configPath := flag.String("config", "", "配置文件路径（默认自动查找）")
	flag.Parse()

	if *keywords == "" {
		fmt.Fprintln(os.Stderr, "❌ 缺少必需参数: --keywords")
		fmt.Println("\n用法: poi-search --keywords=关键词 [--city=城市] [--types=类型] [--page=页码] [--offset=每页数量]")
		fmt.Println("\n示例: poi-search --keywords=肯德基 --city=北京 --page=1 --offset=20")
		os.Exit(1)
	}

	client, err := amap.New(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// --near：先把地名地理编码为坐标，再走周边搜索
	resolvedLocation := *location
	if resolvedLocation == "" && strings.TrimSpace(*near) != "" {
		loc, formatted, err := client.GeocodeLocation(*near, *city)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ 地理编码失败: %v\n", err)
			os.Exit(1)
		}
		resolvedLocation = loc
		fmt.Printf("📍 已定位「%s」→ %s（%s）\n", *near, loc, formatted)
	}

	params := amap.SearchParams{
		Keywords: *keywords, City: *city, Types: *types,
		Location: resolvedLocation, Radius: *radius, Page: *page, Offset: *offset,
	}
	if *cityLimit != "" {
		b := *cityLimit == "true"
		params.CityLimit = &b
	}

	result, err := client.SearchPOI(params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 执行失败: %v\n", err)
		os.Exit(1)
	}
	if len(result.Pois) == 0 {
		fmt.Println("\n❌ 未找到相关结果")
		return
	}

	count := result.CountInt()
	fmt.Printf("\n📍 共找到 %d 条结果，当前显示第 %d 页:\n\n", count, *page)
	fmt.Println(strings.Repeat("=", 80))
	for i, poi := range result.Pois {
		num := (*page-1)**offset + i + 1
		fmt.Printf("\n%d. %s\n", num, poi.Name)
		fmt.Printf("   📍 地址: %s\n", orNone(poi.Address.String()))
		fmt.Printf("   🏷️  类型: %s\n", poi.Type.String())
		fmt.Printf("   📞 电话: %s\n", orNone(poi.Tel.String()))
		fmt.Printf("   🗺️  坐标: %s\n", poi.Location)
		if poi.Distance.String() != "" {
			fmt.Printf("   📏 距离: %s米\n", poi.Distance.String())
		}
	}
	fmt.Println("\n" + strings.Repeat("=", 80))

	if count > 0 && *offset > 0 {
		totalPages := int(math.Ceil(float64(count) / float64(*offset)))
		fmt.Printf("\n第 %d/%d 页\n", *page, totalPages)
		if *page < totalPages {
			fmt.Printf("\n💡 查看下一页: poi-search --keywords=%s --city=%s --page=%d\n", *keywords, *city, *page+1)
		}
	}
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "无"
	}
	return s
}

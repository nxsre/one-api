// geocode：经网关 POST /amap 做地理编码（地名→坐标）与逆地理编码（坐标→地址）。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"amap-lbs-skill-go/internal/amap"
)

func main() {
	address := flag.String("address", "", "地名/地址（地理编码）")
	city := flag.String("city", "", "城市名称或编码（地理编码，可选）")
	location := flag.String("location", "", "坐标 经度,纬度（提供则逆地理编码）")
	configPath := flag.String("config", "", "配置文件路径（默认自动查找）")
	flag.Parse()

	if *address == "" && *location == "" {
		fmt.Fprintln(os.Stderr, "❌ 须提供 --address（地名→坐标）或 --location（坐标→地址）")
		fmt.Println("\n示例:")
		fmt.Println("  geocode --address=西直门 --city=北京")
		fmt.Println("  geocode --location=116.353138,39.939385")
		os.Exit(1)
	}

	client, err := amap.New(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}

	// 逆地理编码优先（给了坐标就反查地址）
	if strings.TrimSpace(*location) != "" {
		r, err := client.ReverseGeocode(*location)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n❌ 执行失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("📍 坐标 %s 对应地址：\n%s\n", *location, r.Regeocode.FormattedAddress.String())
		return
	}

	r, err := client.Geocode(*address, *city)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ 执行失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n📍 「%s」共 %s 条地理编码结果：\n\n", *address, r.Count)
	for i, g := range r.Geocodes {
		fmt.Printf("%d. %s\n", i+1, g.FormattedAddress.String())
		fmt.Printf("   坐标: %s\n", g.Location)
		fmt.Printf("   省/市/区: %s / %s / %s\n\n", g.Province.String(), g.City.String(), g.District.String())
	}
}

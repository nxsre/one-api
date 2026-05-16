package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultEndpoint = "https://restapi.amap.com/v5/place/around"

type config struct {
	endpoint   string
	key        string
	location   string
	keywords   string
	types      string
	radius     string
	sortrule   string
	region     string
	cityLimit  string
	showFields string
	pageSize   string
	pageNum    string
	timeout    time.Duration
	raw        bool
}

type amapResponse struct {
	Status   string   `json:"status"`
	Info     string   `json:"info"`
	InfoCode string   `json:"infocode"`
	Count    string   `json:"count"`
	POIs     poiItems `json:"pois"`
}

type poiItems []poi

type poi struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Parent   string `json:"parent"`
	Location string `json:"location"`
	Distance string `json:"distance"`
	Type     string `json:"type"`
	TypeCode string `json:"typecode"`
	Province string `json:"pname"`
	City     string `json:"cityname"`
	District string `json:"adname"`
	Address  string `json:"address"`
	Business *struct {
		BusinessArea string `json:"business_area"`
		OpenToday    string `json:"opentime_today"`
		OpenWeek     string `json:"opentime_week"`
		Tel          string `json:"tel"`
		Rating       string `json:"rating"`
		Cost         string `json:"cost"`
		Alias        string `json:"alias"`
	} `json:"business"`
	Photos []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"photos"`
}

func (items *poiItems) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*items = nil
		return nil
	}

	var list []poi
	if err := json.Unmarshal(data, &list); err == nil {
		*items = list
		return nil
	}

	var wrapped struct {
		POI []poi `json:"poi"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	*items = wrapped.POI
	return nil
}

func main() {
	cfg := parseFlags()
	if err := cfg.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "poi-test-client: %v\n\n", err)
		flag.Usage()
		os.Exit(2)
	}

	reqURL, err := buildURL(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "poi-test-client: build url: %v\n", err)
		os.Exit(1)
	}

	body, err := request(reqURL, cfg.timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "poi-test-client: request failed: %v\n", err)
		os.Exit(1)
	}
	if cfg.raw {
		fmt.Println(string(body))
		return
	}

	if err := printSummary(body); err != nil {
		fmt.Fprintf(os.Stderr, "poi-test-client: parse response: %v\n", err)
		fmt.Println(string(body))
		os.Exit(1)
	}
}

func parseFlags() config {
	cfg := config{}
	flag.StringVar(&cfg.endpoint, "endpoint", defaultEndpoint, "高德 POI 周边搜索接口地址")
	flag.StringVar(&cfg.key, "key", strings.TrimSpace(os.Getenv("AMAP_KEY")), "高德 Web 服务 API Key，也可用 AMAP_KEY 环境变量")
	flag.StringVar(&cfg.location, "location", "", "中心点坐标，格式：经度,纬度，例如 116.473168,39.993015")
	flag.StringVar(&cfg.keywords, "keywords", "", "地点关键字，只支持一个，最长 80 字符")
	flag.StringVar(&cfg.types, "types", "", "POI 类型编码，多个用 | 分隔，例如 050000|070000")
	flag.StringVar(&cfg.radius, "radius", "5000", "搜索半径，单位米，范围 0-50000")
	flag.StringVar(&cfg.sortrule, "sortrule", "distance", "排序规则：distance 或 weight")
	flag.StringVar(&cfg.region, "region", "", "搜索区划，可传行政区名、citycode 或 adcode")
	flag.StringVar(&cfg.cityLimit, "city_limit", "", "是否仅召回 region 区域内数据：true 或 false")
	flag.StringVar(&cfg.showFields, "show_fields", "business,photos", "扩展返回字段，多个用 , 分隔；留空只返回基础字段")
	flag.StringVar(&cfg.pageSize, "page_size", "10", "每页条数，范围 1-25")
	flag.StringVar(&cfg.pageNum, "page_num", "1", "页码")
	flag.DurationVar(&cfg.timeout, "timeout", 10*time.Second, "请求超时时间")
	flag.BoolVar(&cfg.raw, "raw", false, "直接输出原始 JSON")
	flag.Parse()
	return cfg
}

func (cfg config) validate() error {
	if cfg.key == "" {
		return errors.New("缺少 key，请传 -key 或设置 AMAP_KEY")
	}
	if cfg.location == "" {
		return errors.New("缺少 location，例如 -location 116.473168,39.993015")
	}
	return nil
}

func buildURL(cfg config) (string, error) {
	u, err := url.Parse(cfg.endpoint)
	if err != nil {
		return "", err
	}

	q := u.Query()
	add := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			q.Set(key, value)
		}
	}
	add("key", cfg.key)
	add("location", cfg.location)
	add("keywords", cfg.keywords)
	add("types", cfg.types)
	add("radius", cfg.radius)
	add("sortrule", cfg.sortrule)
	add("region", cfg.region)
	add("city_limit", cfg.cityLimit)
	add("show_fields", cfg.showFields)
	add("page_size", cfg.pageSize)
	add("page_num", cfg.pageNum)
	q.Set("output", "json")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func request(reqURL string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func printSummary(body []byte) error {
	var resp amapResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}

	fmt.Printf("status=%s info=%s infocode=%s count=%s returned=%d\n", resp.Status, resp.Info, resp.InfoCode, resp.Count, len(resp.POIs))
	for i, item := range resp.POIs {
		fmt.Printf("%02d. %s [%s]\n", i+1, item.Name, item.ID)
		fmt.Printf("    type=%s(%s) distance=%sm location=%s\n", item.Type, item.TypeCode, item.Distance, item.Location)
		fmt.Printf("    address=%s%s%s %s\n", item.Province, item.City, item.District, item.Address)
		if item.Business != nil {
			fmt.Printf("    business_area=%s tel=%s rating=%s cost=%s\n", item.Business.BusinessArea, item.Business.Tel, item.Business.Rating, item.Business.Cost)
		}
		if len(item.Photos) > 0 {
			fmt.Printf("    photo=%s %s\n", item.Photos[0].Title, item.Photos[0].URL)
		}
	}
	return nil
}

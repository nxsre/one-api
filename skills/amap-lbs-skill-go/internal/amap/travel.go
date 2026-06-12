package amap

import (
	"fmt"
	"os"
)

// TravelParams 为旅游规划参数。
type TravelParams struct {
	City      string
	Interests []string
	RouteType string // walking/driving/riding/transfer，默认 walking
}

// TravelResult 为旅游规划结果。
type TravelResult struct {
	Pois        []POI
	MapTaskData []MapTask
}

// TravelPlanner 搜索各类兴趣点（每类最多 5 个）并按顺序规划路线，全程经网关。
func (c *Client) TravelPlanner(p TravelParams) (*TravelResult, error) {
	routeType := p.RouteType
	if routeType == "" {
		routeType = "walking"
	}
	interests := p.Interests
	if len(interests) == 0 {
		interests = []string{"景点", "美食"}
	}

	res := &TravelResult{}
	for _, interest := range interests {
		fmt.Printf("📍 搜索 %s...\n", interest)
		r, err := c.SearchPOI(SearchParams{Keywords: interest, City: p.City, Page: 1, Offset: 5})
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  搜索 %s 失败: %v\n", interest, err)
			continue
		}
		if r == nil || len(r.Pois) == 0 {
			continue
		}
		res.Pois = append(res.Pois, r.Pois...)
		for _, poi := range r.Pois {
			lng, lat := parseLngLat(poi.Location)
			sort := poi.Type.String()
			if sort == "" {
				sort = interest
			}
			remark := poi.Address.String()
			if remark == "" {
				remark = interest + "推荐"
			}
			res.MapTaskData = append(res.MapTaskData, MapTask{
				Type: "poi", Lnglat: []float64{lng, lat}, Sort: sort, Text: poi.Name, Remark: remark,
			})
		}
	}

	if len(res.Pois) >= 2 {
		for i := 0; i < len(res.Pois)-1; i++ {
			s := res.Pois[i]
			e := res.Pois[i+1]
			sLng, sLat := parseLngLat(s.Location)
			eLng, eLat := parseLngLat(e.Location)
			task := MapTask{
				Type: "route", RouteType: routeType,
				Start: []float64{sLng, sLat}, End: []float64{eLng, eLat},
				Remark: fmt.Sprintf("从 %s 到 %s", s.Name, e.Name),
			}
			if routeType == "transfer" {
				task.City = p.City
			}
			res.MapTaskData = append(res.MapTaskData, task)
		}
	}
	return res, nil
}

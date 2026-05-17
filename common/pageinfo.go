package common

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/songquanpeng/one-api/common/config"
)

// MaxPageNumber 单请求允许的最大页码，避免超大 offset。
const MaxPageNumber = 100000

// PageInfo 分页响应体（页码、每页条数、总数）。
type PageInfo struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Items    any `json:"items"`
}

func (p *PageInfo) GetStartIdx() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}

func (p *PageInfo) GetPageSize() int {
	return p.PageSize
}

func (p *PageInfo) GetPage() int {
	return p.Page
}

func (p *PageInfo) SetTotal(total int) {
	p.Total = total
}

func (p *PageInfo) SetItems(items any) {
	p.Items = items
}

// GetPageQuery 解析 p（1 基页码）、page_size（1..100）。
func GetPageQuery(c *gin.Context) *PageInfo {
	pageInfo := &PageInfo{}
	if page, err := strconv.Atoi(c.Query("p")); err == nil {
		pageInfo.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil {
		pageInfo.PageSize = pageSize
	}
	if pageInfo.Page < 1 {
		pageInfo.Page = 1
	}
	if pageInfo.PageSize == 0 {
		if ps, _ := strconv.Atoi(c.Query("ps")); ps != 0 {
			pageInfo.PageSize = ps
		}
	}
	if pageInfo.PageSize == 0 {
		pageInfo.PageSize = config.ItemsPerPage
	}
	if pageInfo.PageSize < 1 {
		pageInfo.PageSize = config.ItemsPerPage
	}
	if pageInfo.PageSize > 100 {
		pageInfo.PageSize = 100
	}
	if pageInfo.Page > MaxPageNumber {
		pageInfo.Page = MaxPageNumber
	}
	return pageInfo
}

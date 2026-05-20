package controller

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/model"
)

func syncFromAliapi(ctx context.Context, req *modelCatalogSyncRequest) ([]model.ModelCatalog, error) {
	return syncFromAliapiWithClient(ctx, req, client.NewOutboundHTTPClient(30*time.Second))
}

func syncFromAliapiWithClient(ctx context.Context, req *modelCatalogSyncRequest, hc *http.Client) ([]model.ModelCatalog, error) {
	baseURL := strings.TrimSpace(req.BaseURL)
	if baseURL == "" {
		baseURL = "https://aliapi.me/models.html"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("User-Agent", "one-api/aliapi-sync")

	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: unable to fetch models", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var rows []model.ModelCatalog

	reCost := regexp.MustCompile(`￥\s*(\d+(?:\.\d+)?)`)
	reContext := regexp.MustCompile(`上下文\s*(\d+(?:\.\d+)?)k`)

	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		nameCol := s.Find("td").Eq(0)
		providerCol := s.Find("td").Eq(1)
		tagsCol := s.Find("td").Eq(2)
		priceCol := s.Find("td").Eq(3)
		ratioCol := s.Find("td").Eq(4)

		modelID := strings.TrimSpace(nameCol.Text())
		if modelID == "" {
			return
		}

		provider := strings.TrimSpace(providerCol.Text())
		tagsText := strings.TrimSpace(tagsCol.Text())
		
		tagsArr := strings.Split(strings.ReplaceAll(tagsText, "，", "、"), "、")
		var cleanTags []string
		for _, tag := range tagsArr {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				cleanTags = append(cleanTags, tag)
			}
		}
		tagsStr := strings.Join(cleanTags, ",")

		priceText, _ := priceCol.Html()
		var inputPrice, outputPrice, cachePrice float64
		if priceText != "" {
			lines := strings.Split(priceText, "<br/>")
			if len(lines) == 1 {
				lines = strings.Split(priceText, "<br>")
			}
			for _, line := range lines {
				if strings.Contains(line, "输入") {
					matches := reCost.FindStringSubmatch(line)
					if len(matches) > 1 {
						inputPrice, _ = strconv.ParseFloat(matches[1], 64)
					}
				} else if strings.Contains(line, "输出") {
					matches := reCost.FindStringSubmatch(line)
					if len(matches) > 1 {
						outputPrice, _ = strconv.ParseFloat(matches[1], 64)
					}
				} else if strings.Contains(line, "缓存命中") {
					matches := reCost.FindStringSubmatch(line)
					if len(matches) > 1 {
						cachePrice, _ = strconv.ParseFloat(matches[1], 64)
					}
				}
			}
		}

		ratioText := strings.TrimSpace(ratioCol.Text())
		var contextLimit int
		if ratioText != "" {
			matches := reContext.FindStringSubmatch(ratioText)
			if len(matches) > 1 {
				limitK, _ := strconv.ParseFloat(matches[1], 64)
				contextLimit = int(limitK * 1024)
			}
		}

		rows = append(rows, model.ModelCatalog{
			ModelId:       modelID,
			OwnedBy:       provider,
			Enabled:       true,
			Source:        "aliapi",
			ModelName:     modelID,
			Tags:          tagsStr,
			ContextLimit:  contextLimit,
			CostInput:     inputPrice,
			CostOutput:    outputPrice,
			CostCacheRead: cachePrice,
		})
	})

	return rows, nil
}
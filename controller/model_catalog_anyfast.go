package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/client"
	"github.com/songquanpeng/one-api/model"
)

const (
	anyfastDefaultBaseURL = "https://www.anyfast.ai"
	anyfastPricingPath    = "/api/pricing"
)

type anyfastPricingResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Data    []anyfastPricingItem     `json:"data"`
	Vendors []anyfastVendor          `json:"vendors"`
}

type anyfastVendor struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type anyfastPricingItem struct {
	ModelName        string               `json:"model_name"`
	VendorID         int                  `json:"vendor_id"`
	QuotaType        int                  `json:"quota_type"`
	ModelRatio       float64              `json:"model_ratio"`
	ModelPrice       float64              `json:"model_price"`
	CompletionRatio  float64              `json:"completion_ratio"`
	CacheRatio       *float64             `json:"cache_ratio"`
	CreateCacheRatio *float64             `json:"create_cache_ratio"`
	OwnerBy          string               `json:"owner_by"`
	Tags             string               `json:"tags"`
	BillingConfig    *anyfastBillingConfig `json:"billing_config"`
}

type anyfastBillingConfig struct {
	UsePrice bool                `json:"usePrice"`
	Rules    []anyfastBillingRule `json:"rules"`
}

type anyfastBillingRule struct {
	Type           string   `json:"type"`
	InputPrice     float64  `json:"inputPrice"`
	OutputPrice    float64  `json:"outputPrice"`
	InputRatio     float64  `json:"inputRatio"`
	OutputRatio    float64  `json:"outputRatio"`
	CacheRatio     *float64 `json:"cacheRatio"`
	CreateCacheRatio *float64 `json:"createCacheRatio"`
	Price          float64  `json:"price"`
}

func anyfastPricingURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = anyfastDefaultBaseURL
	}
	if strings.HasSuffix(strings.ToLower(base), "/api/pricing") {
		return base
	}
	return base + anyfastPricingPath
}

func syncFromAnyfast(ctx context.Context, req *modelCatalogSyncRequest) ([]model.ModelCatalog, error) {
	return syncFromAnyfastWithClient(ctx, req, client.NewOutboundHTTPClient(30*time.Second))
}

func syncFromAnyfastWithClient(ctx context.Context, req *modelCatalogSyncRequest, hc *http.Client) ([]model.ModelCatalog, error) {
	fullURL := anyfastPricingURL(req.BaseURL)
	if err := common.ValidateOutboundURL(fullURL); err != nil {
		return nil, fmt.Errorf("URL 未通过安全校验: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("User-Agent", "one-api/anyfast-sync")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: unable to fetch pricing", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, err
	}

	return rowsFromAnyfastPricingPayload(body)
}

func rowsFromAnyfastPricingPayload(body []byte) ([]model.ModelCatalog, error) {
	var payload anyfastPricingResponse
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	if !payload.Success {
		msg := strings.TrimSpace(payload.Message)
		if msg == "" {
			msg = "upstream returned success=false"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("empty pricing list from upstream")
	}

	vendorByID := make(map[int]string, len(payload.Vendors))
	for _, v := range payload.Vendors {
		if v.ID > 0 && strings.TrimSpace(v.Name) != "" {
			vendorByID[v.ID] = strings.TrimSpace(v.Name)
		}
	}

	rows := make([]model.ModelCatalog, 0, len(payload.Data))
	for _, item := range payload.Data {
		modelID := strings.TrimSpace(item.ModelName)
		if modelID == "" {
			continue
		}

		provider := strings.TrimSpace(item.OwnerBy)
		if provider == "" {
			provider = vendorByID[item.VendorID]
		}

		costIn, costOut, costCacheRead, costCacheWrite := anyfastCatalogCosts(item)

		notes := ""
		if item.QuotaType == 0 && item.ModelRatio > 0 {
			notes = fmt.Sprintf("倍率: input=%g completion=%g", item.ModelRatio, item.CompletionRatio)
		}

		rows = append(rows, model.ModelCatalog{
			ModelId:         modelID,
			OwnedBy:         provider,
			Enabled:         true,
			Source:          "anyfast",
			ModelName:       modelID,
			ProviderDisplay: provider,
			Tags:            strings.TrimSpace(item.Tags),
			CostInput:       costIn,
			CostOutput:      costOut,
			CostCacheRead:   costCacheRead,
			CostCacheWrite:  costCacheWrite,
			Notes:           notes,
		})
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no models parsed from pricing response")
	}
	return rows, nil
}

func anyfastCatalogCosts(item anyfastPricingItem) (input, output, cacheRead, cacheWrite float64) {
	if item.BillingConfig != nil && len(item.BillingConfig.Rules) > 0 {
		rule := item.BillingConfig.Rules[0]
		switch rule.Type {
		case "fixed":
			if rule.Price > 0 {
				return rule.Price, 0, 0, 0
			}
		default:
			if rule.InputPrice > 0 {
				input = rule.InputPrice
			}
			if rule.OutputPrice > 0 {
				output = rule.OutputPrice
			}
			if rule.CacheRatio != nil && rule.InputRatio > 0 {
				cacheRead = rule.InputRatio * *rule.CacheRatio
			}
			if rule.CreateCacheRatio != nil && rule.InputRatio > 0 {
				cacheWrite = rule.InputRatio * *rule.CreateCacheRatio
			}
		}
	}

	if input == 0 && item.QuotaType == 1 && item.ModelPrice > 0 {
		input = item.ModelPrice
	}
	if output == 0 && input > 0 && item.CompletionRatio > 0 {
		output = input * item.CompletionRatio
	}
	if cacheRead == 0 && item.CacheRatio != nil && input > 0 {
		cacheRead = input * *item.CacheRatio
	}
	if cacheWrite == 0 && item.CreateCacheRatio != nil && input > 0 {
		cacheWrite = input * *item.CreateCacheRatio
	}
	return input, output, cacheRead, cacheWrite
}

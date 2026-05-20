package dto

type UpstreamDTO struct {
	ID       int    `json:"id,omitempty"`
	Name     string `json:"name" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required"`
	Endpoint string `json:"endpoint"`
}

type UpstreamRequest struct {
	ChannelIDs []int64       `json:"channel_ids"`
	Upstreams  []UpstreamDTO `json:"upstreams"`
	Timeout    int           `json:"timeout"`
}

type TestResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type DifferenceItem struct {
	Current    interface{}            `json:"current"`
	Upstreams  map[string]interface{} `json:"upstreams"`
	Confidence map[string]bool        `json:"confidence"`
}

type SyncableChannel struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Status  int    `json:"status"`
	Type    int    `json:"type"`
}

type PricingSyncApplyItem struct {
	Model string      `json:"model" binding:"required"`
	Field string      `json:"field" binding:"required"`
	Value interface{} `json:"value" binding:"required"`
}

type PricingSyncApplyRequest struct {
	Items    []PricingSyncApplyItem `json:"items" binding:"required"`
	Activate bool                   `json:"activate"`
	Label    string                 `json:"label"`
	Source   string                 `json:"source"`
	Note     string                 `json:"note"`
}

type PricingVersionActivateRequest struct {
	BlockID   string `json:"block_id" binding:"required"`
	VersionID int    `json:"version_id" binding:"required"`
}

type UpstreamSyncSelectionItem struct {
	Model        string `json:"model" binding:"required"`
	Field        string `json:"field" binding:"required"`
	UpstreamName string `json:"upstream_name"`
	Selected     bool   `json:"selected"`
}

type UpstreamSyncSaveSelectionsRequest struct {
	Items []UpstreamSyncSelectionItem `json:"items"`
}

type UpstreamSyncSelectAllRequest struct {
	UpstreamName string `json:"upstream_name"`
	Selected     bool   `json:"selected"`
}

type UpstreamSyncApplyBatchRequest struct {
	Activate bool   `json:"activate"`
	Label    string `json:"label"`
	Source   string `json:"source"`
	Note     string `json:"note"`
}

type UpstreamSyncCompareRequest struct {
	Left  string `form:"left" binding:"required"`
	Right string `form:"right" binding:"required"`
	Page  int    `form:"page"`
	Size  int    `form:"page_size"`
	Model string `form:"model"`
}

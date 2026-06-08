package model

import "strings"

// ModelSquareCard 模型广场卡片所需的精简元数据（取自 model_catalog 的当前启用行）。
type ModelSquareCard struct {
	ModelID         string `json:"model_id"`
	ModelName       string `json:"model_name"`
	ProviderKey     string `json:"provider_key"`
	ProviderDisplay string `json:"provider_display"`
	Family          string `json:"family"`
	ModalitiesIn    string `json:"modalities_in"`
	ModalitiesOut   string `json:"modalities_out"`
	ContextLimit    int    `json:"context_limit"`
	Reasoning       bool   `json:"reasoning"`
	ToolCall        bool   `json:"tool_call"`
	AttachmentOK    bool   `json:"attachment_ok"`
}

// GetModelSquareCards 返回给定 model_id 集合在模型目录中的卡片元数据。
//   - 仅取 status=current AND enabled=true 行；
//   - 同一 model_id 在多 provider 下可能有多行，按 provider_key 字典序去重保留首个（结果稳定）；
//   - category 走模型广场分类过滤；search 为模糊关键字（可空）。
//
// 注意：不含目录元数据的可用模型（无对应行）不会出现在此结果里——由调用方在「无分类」时补裸卡片。
func GetModelSquareCards(ids []string, category, search string) ([]ModelSquareCard, error) {
	if len(ids) == 0 {
		return []ModelSquareCard{}, nil
	}
	q := DB.Model(&ModelCatalog{}).
		Where("status = ?", "current").
		Where("enabled = ?", true).
		Where("model_id IN ?", ids)
	q = modelCatalogApplyCategoryFilter(q, category)
	if s := strings.TrimSpace(search); s != "" {
		q = modelCatalogSearchQuery(q, s)
	}
	var rows []ModelCatalog
	if err := q.Order("model_id ASC, provider_key ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(rows))
	out := make([]ModelSquareCard, 0, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.ModelId]; ok {
			continue
		}
		seen[r.ModelId] = struct{}{}
		out = append(out, ModelSquareCard{
			ModelID:         r.ModelId,
			ModelName:       r.ModelName,
			ProviderKey:     r.ProviderKey,
			ProviderDisplay: r.ProviderDisplay,
			Family:          r.Family,
			ModalitiesIn:    r.ModalitiesIn,
			ModalitiesOut:   r.ModalitiesOut,
			ContextLimit:    r.ContextLimit,
			Reasoning:       r.Reasoning,
			ToolCall:        r.ToolCall,
			AttachmentOK:    r.AttachmentOK,
		})
	}
	return out, nil
}

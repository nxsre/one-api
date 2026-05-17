package controller

import (
	"sort"
	"strings"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

func defaultModelPermissions() []OpenAIModelPermission {
	return []OpenAIModelPermission{{
		Id:                 "modelperm-LwHkVFn8AcMItP432fKKDIKJ",
		Object:             "model_permission",
		Created:            1626777600,
		AllowCreateEngine:  true,
		AllowSampling:      true,
		AllowLogprobs:      true,
		AllowSearchIndices: false,
		AllowView:          true,
		AllowFineTuning:    false,
		Organization:       "*",
		Group:              nil,
		IsBlocking:         false,
	}}
}

// mergedOpenAIModels 将进程内置列表与数据库 model_catalog 合并：catalog 中 enabled=false 会移除该项；enabled=true 可覆盖 owned_by 或新增模型 id。
func mergedOpenAIModels() ([]OpenAIModels, error) {
	rows, err := model.GetAllModelCatalog()
	if err != nil {
		return nil, err
	}

	byID := make(map[string]OpenAIModels, len(models)+len(rows))
	for _, m := range models {
		byID[m.Id] = m
	}

	disabled := make(map[string]bool)
	enabledRows := make(map[string]model.ModelCatalog)
	for _, r := range rows {
		mid := strings.TrimSpace(r.ModelId)
		if mid == "" {
			continue
		}
		if !r.Enabled {
			disabled[mid] = true
			continue
		}
		enabledRows[mid] = r
	}

	for mid := range disabled {
		delete(byID, mid)
	}

	for mid, r := range enabledRows {
		ob := strings.TrimSpace(r.OwnedBy)
		base := byID[mid]
		if ob == "" {
			if base.Id != "" {
				ob = base.OwnedBy
			} else {
				ob = "custom"
			}
		}
		created := 1626777600
		if base.Id != "" {
			created = base.Created
		}
		byID[mid] = OpenAIModels{
			Id:         mid,
			Object:     "model",
			Created:    created,
			OwnedBy:    ob,
			Permission: defaultModelPermissions(),
			Root:       mid,
			Parent:     nil,
		}
	}

	out := make([]OpenAIModels, 0, len(byID))
	for _, v := range byID {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out, nil
}

// mergedOpenAIModelsBestEffort 在数据库异常时退回到内置列表，避免用户侧 /v1/models 全挂。
func mergedOpenAIModelsBestEffort() []OpenAIModels {
	m, err := mergedOpenAIModels()
	if err != nil {
		logger.SysError("mergedOpenAIModels: " + err.Error())
		return models
	}
	return m
}

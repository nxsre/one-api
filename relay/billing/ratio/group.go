package ratio

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/songquanpeng/one-api/common/logger"
)

var groupRatioLock sync.RWMutex
var GroupRatio = map[string]float64{
	"default": 1,
	"vip":     1,
	"svip":    1,
}

// GroupGroupRatio 用户所属分组 → 实际使用分组 的特殊倍率（new-api 兼容）。
var GroupGroupRatio = map[string]map[string]float64{}

// TopupGroupRatio 充值金额换算倍率。
var TopupGroupRatio = map[string]float64{
	"default": 1,
}

func GroupRatio2JSONString() string {
	jsonBytes, err := json.Marshal(GroupRatio)
	if err != nil {
		logger.SysError("error marshalling model ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateGroupRatioByJSONString(jsonStr string) error {
	groupRatioLock.Lock()
	defer groupRatioLock.Unlock()
	GroupRatio = make(map[string]float64)
	return json.Unmarshal([]byte(jsonStr), &GroupRatio)
}

func GetGroupRatio(name string) float64 {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	ratio, ok := GroupRatio[name]
	if !ok {
		logger.SysError("group ratio not found: " + name)
		return 1
	}
	return ratio
}

func GroupGroupRatio2JSONString() string {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	b, err := json.Marshal(GroupGroupRatio)
	if err != nil {
		logger.SysError("error marshalling group group ratio: " + err.Error())
		return "{}"
	}
	return string(b)
}

func UpdateGroupGroupRatioByJSONString(jsonStr string) error {
	groupRatioLock.Lock()
	defer groupRatioLock.Unlock()
	GroupGroupRatio = make(map[string]map[string]float64)
	if strings.TrimSpace(jsonStr) == "" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &GroupGroupRatio)
}

func GetGroupGroupRatio(userGroup, usingGroup string) (float64, bool) {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	gp, ok := GroupGroupRatio[userGroup]
	if !ok {
		return -1, false
	}
	ratio, ok := gp[usingGroup]
	if !ok {
		return -1, false
	}
	return ratio, true
}

// GetEffectiveGroupRatio 返回实际用于计费的分组倍率（优先 GroupGroupRatio）。
func GetEffectiveGroupRatio(userGroup, usingGroup string) float64 {
	if ratio, ok := GetGroupGroupRatio(userGroup, usingGroup); ok {
		return ratio
	}
	if usingGroup != "" {
		return GetGroupRatio(usingGroup)
	}
	return GetGroupRatio(userGroup)
}

func TopupGroupRatio2JSONString() string {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	b, err := json.Marshal(TopupGroupRatio)
	if err != nil {
		logger.SysError("error marshalling topup group ratio: " + err.Error())
		return "{}"
	}
	return string(b)
}

func UpdateTopupGroupRatioByJSONString(jsonStr string) error {
	groupRatioLock.Lock()
	defer groupRatioLock.Unlock()
	TopupGroupRatio = make(map[string]float64)
	if strings.TrimSpace(jsonStr) == "" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), &TopupGroupRatio)
}

func GetGroupRatioCopy() map[string]float64 {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	return copyRatioMap(GroupRatio)
}

func GetGroupGroupRatioCopy() map[string]map[string]float64 {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	out := make(map[string]map[string]float64, len(GroupGroupRatio))
	for outer, inner := range GroupGroupRatio {
		innerCopy := make(map[string]float64, len(inner))
		for k, v := range inner {
			innerCopy[k] = v
		}
		out[outer] = innerCopy
	}
	return out
}

func GetTopupGroupRatioCopy() map[string]float64 {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	return copyRatioMap(TopupGroupRatio)
}

func GetTopupGroupRatio(name string) float64 {
	groupRatioLock.RLock()
	defer groupRatioLock.RUnlock()
	if ratio, ok := TopupGroupRatio[name]; ok {
		return ratio
	}
	return 1
}

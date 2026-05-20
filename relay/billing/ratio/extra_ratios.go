package ratio

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/songquanpeng/one-api/common/logger"
)

var extraRatioLock sync.RWMutex

// ModelPrice 按次/按请求固定价（美元）；配置后优先于 ModelRatio 的 token 计费。
var ModelPrice = map[string]float64{}

// CacheRatio 缓存命中（读缓存）token 相对输入 prompt 的倍率。
var CacheRatio = map[string]float64{}

// CreateCacheRatio 写入缓存 token 的倍率（未配置时计费默认 1.25）。
var CreateCacheRatio = map[string]float64{}

// ImageRatio 图片输入 token 相对普通输入的倍率。
var ImageRatio = map[string]float64{}

// AudioRatio 音频输入 token 倍率。
var AudioRatio = map[string]float64{}

// AudioCompletionRatio 音频输出 token 相对输入的倍率。
var AudioCompletionRatio = map[string]float64{}

func normalizeRatioModelName(originModel, mappedModel string) string {
	name := BillingModelName(originModel, mappedModel)
	if strings.HasPrefix(name, "qwen-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}
	if strings.HasPrefix(name, "command-") && strings.HasSuffix(name, "-internet") {
		name = strings.TrimSuffix(name, "-internet")
	}
	return FormatMatchingModelName(name)
}

func lookupRatioMap(m map[string]float64, originModel, mappedModel string, channelType int) (float64, bool) {
	name := normalizeRatioModelName(originModel, mappedModel)
	if key, ok := channelRatioSlugKey(name, channelType); ok {
		if ratio, ok := m[key]; ok {
			return ratio, true
		}
	}
	if ratio, ok := m[name]; ok {
		return ratio, true
	}
	return 0, false
}

func updateExtraRatioMap(jsonStr string, target *map[string]float64) error {
	extraRatioLock.Lock()
	defer extraRatioLock.Unlock()
	return unmarshalRatioMap(jsonStr, target)
}

func ModelPrice2JSONString() string {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return ratioMapJSON(ModelPrice)
}

func UpdateModelPriceByJSONString(jsonStr string) error {
	return updateExtraRatioMap(jsonStr, &ModelPrice)
}

// GetModelPrice 返回模型固定价（美元）及是否配置。
func GetModelPrice(originModel, mappedModel string, channelType int) (float64, bool) {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	if price, ok := lookupRatioMap(ModelPrice, originModel, mappedModel, channelType); ok {
		return price, true
	}
	return -1, false
}

func CacheRatio2JSONString() string {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return ratioMapJSON(CacheRatio)
}

func UpdateCacheRatioByJSONString(jsonStr string) error {
	return updateExtraRatioMap(jsonStr, &CacheRatio)
}

func GetCacheRatio(originModel, mappedModel string, channelType int) (float64, bool) {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	if ratio, ok := lookupRatioMap(CacheRatio, originModel, mappedModel, channelType); ok {
		return ratio, true
	}
	return 1, false
}

func CreateCacheRatio2JSONString() string {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return ratioMapJSON(CreateCacheRatio)
}

func UpdateCreateCacheRatioByJSONString(jsonStr string) error {
	return updateExtraRatioMap(jsonStr, &CreateCacheRatio)
}

func GetCreateCacheRatio(originModel, mappedModel string, channelType int) (float64, bool) {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	if ratio, ok := lookupRatioMap(CreateCacheRatio, originModel, mappedModel, channelType); ok {
		return ratio, true
	}
	return 1.25, false
}

func ImageRatio2JSONString() string {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return ratioMapJSON(ImageRatio)
}

func UpdateImageRatioByJSONString(jsonStr string) error {
	return updateExtraRatioMap(jsonStr, &ImageRatio)
}

func GetImageRatio(originModel, mappedModel string, channelType int) (float64, bool) {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	if ratio, ok := lookupRatioMap(ImageRatio, originModel, mappedModel, channelType); ok {
		return ratio, true
	}
	return 1, false
}

func AudioRatio2JSONString() string {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return ratioMapJSON(AudioRatio)
}

func UpdateAudioRatioByJSONString(jsonStr string) error {
	return updateExtraRatioMap(jsonStr, &AudioRatio)
}

func GetAudioRatio(originModel, mappedModel string, channelType int) float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	if ratio, ok := lookupRatioMap(AudioRatio, originModel, mappedModel, channelType); ok {
		return ratio
	}
	return 1
}

func AudioCompletionRatio2JSONString() string {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return ratioMapJSON(AudioCompletionRatio)
}

func UpdateAudioCompletionRatioByJSONString(jsonStr string) error {
	return updateExtraRatioMap(jsonStr, &AudioCompletionRatio)
}

func GetAudioCompletionRatio(originModel, mappedModel string, channelType int) float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	if ratio, ok := lookupRatioMap(AudioCompletionRatio, originModel, mappedModel, channelType); ok {
		return ratio
	}
	return 1
}

func GetCacheRatioCopy() map[string]float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return copyRatioMap(CacheRatio)
}

func GetCreateCacheRatioCopy() map[string]float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return copyRatioMap(CreateCacheRatio)
}

func GetImageRatioCopy() map[string]float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return copyRatioMap(ImageRatio)
}

func GetAudioRatioCopy() map[string]float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return copyRatioMap(AudioRatio)
}

func GetAudioCompletionRatioCopy() map[string]float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return copyRatioMap(AudioCompletionRatio)
}

func GetModelPriceCopy() map[string]float64 {
	extraRatioLock.RLock()
	defer extraRatioLock.RUnlock()
	return copyRatioMap(ModelPrice)
}

func ratioMapJSON(m map[string]float64) string {
	b, err := json.Marshal(m)
	if err != nil {
		logger.SysError("error marshalling ratio map: " + err.Error())
		return "{}"
	}
	return string(b)
}

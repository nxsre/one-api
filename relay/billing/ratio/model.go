package ratio

import (
	"encoding/json"
	"regexp"
	"strconv"
	"sync"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/metrics"
	"github.com/songquanpeng/one-api/relay/channeltype"
)

// 按渠道覆盖倍率时 key 写法（查找顺序：渠道 slug → 裸名）：
//
//   1. "<modelName>@<channelSlug>"，如 "llama3-8b-8192@aws-claude"。
//   2. "<modelName>"，作为所有渠道的兜底单价。
//
// channelSlug 来自 channeltype.SlugByID，详见 relay/channeltype/slug.go。
func channelRatioSlugKey(name string, channelType int) (string, bool) {
	slug := channeltype.SlugByID(channelType)
	if slug == "" {
		return "", false
	}
	return name + "@" + slug, true
}

// legacyChannelRatioKeyRe 匹配已废弃的 "<modelName>(<channelType>)" 键（仅用于加载时迁移）。
var legacyChannelRatioKeyRe = regexp.MustCompile(`^(.+)\((\d+)\)$`)

// migrateLegacyParenRatioKeys 将 map 中旧的 (channelType) 键改写为 @channelSlug，并删除旧键。
func migrateLegacyParenRatioKeys(m map[string]float64) {
	for key, ratio := range m {
		ms := legacyChannelRatioKeyRe.FindStringSubmatch(key)
		if len(ms) != 3 {
			continue
		}
		channelID, err := strconv.Atoi(ms[2])
		if err != nil {
			continue
		}
		slug := channeltype.SlugByID(channelID)
		if slug == "" {
			delete(m, key)
			continue
		}
		newKey := ms[1] + "@" + slug
		if _, exists := m[newKey]; !exists {
			m[newKey] = ratio
		}
		delete(m, key)
	}
}

func unmarshalRatioMap(jsonStr string, target *map[string]float64) error {
	m := make(map[string]float64)
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return err
	}
	migrateLegacyParenRatioKeys(m)
	*target = m
	return nil
}

const (
	USD2RMB   = 7
	USD       = 500 // $0.002 = 1 -> $1 = 500
	MILLI_USD = 1.0 / 1000 * USD
	RMB       = USD / USD2RMB
)

var modelRatioLock sync.RWMutex

// ModelRatio 输入 token 单价倍率（1 === $0.002 / 1K tokens）。
// 价目由管理端「运营设置」中的 ModelRatio JSON 维护；查表请用官方模型名（见 BillingModelName）。
var ModelRatio = map[string]float64{}

// CompletionRatio 输出 token 相对输入的倍率（outputTokens × CompletionRatio 计入计费）。
// 由管理端「运营设置」中的 CompletionRatio JSON 维护。
var CompletionRatio = map[string]float64{}

var (
	DefaultModelRatio      map[string]float64
	DefaultCompletionRatio map[string]float64
)

func init() {
	DefaultModelRatio = make(map[string]float64)
	for k, v := range ModelRatio {
		DefaultModelRatio[k] = v
	}
	DefaultCompletionRatio = make(map[string]float64)
	for k, v := range CompletionRatio {
		DefaultCompletionRatio[k] = v
	}
}

func AddNewMissingRatio(oldRatio string) string {
	newRatio := make(map[string]float64)
	err := json.Unmarshal([]byte(oldRatio), &newRatio)
	if err != nil {
		logger.SysError("error unmarshalling old ratio: " + err.Error())
		return oldRatio
	}
	migrateLegacyParenRatioKeys(newRatio)
	for k, v := range DefaultModelRatio {
		if _, ok := newRatio[k]; !ok {
			newRatio[k] = v
		}
	}
	jsonBytes, err := json.Marshal(newRatio)
	if err != nil {
		logger.SysError("error marshalling new ratio: " + err.Error())
		return oldRatio
	}
	return string(jsonBytes)
}

func ModelRatio2JSONString() string {
	jsonBytes, err := json.Marshal(ModelRatio)
	if err != nil {
		logger.SysError("error marshalling model ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateModelRatioByJSONString(jsonStr string) error {
	modelRatioLock.Lock()
	defer modelRatioLock.Unlock()
	return unmarshalRatioMap(jsonStr, &ModelRatio)
}

func GetModelRatio(originModel, mappedModel string, channelType int) float64 {
	name := normalizeRatioModelName(originModel, mappedModel)
	modelRatioLock.RLock()
	defer modelRatioLock.RUnlock()
	if key, ok := channelRatioSlugKey(name, channelType); ok {
		if ratio, ok := ModelRatio[key]; ok {
			return ratio
		}
		if ratio, ok := DefaultModelRatio[key]; ok {
			return ratio
		}
	}
	if ratio, ok := ModelRatio[name]; ok {
		return ratio
	}
	if ratio, ok := DefaultModelRatio[name]; ok {
		return ratio
	}
	logger.SysError("model ratio not found: " + name + " (请在管理端运营设置中配置 ModelRatio)")
	metrics.RatioFallback.WithLabelValues(name, "model").Inc()
	return 1
}

func CompletionRatio2JSONString() string {
	jsonBytes, err := json.Marshal(CompletionRatio)
	if err != nil {
		logger.SysError("error marshalling completion ratio: " + err.Error())
	}
	return string(jsonBytes)
}

func UpdateCompletionRatioByJSONString(jsonStr string) error {
	return unmarshalRatioMap(jsonStr, &CompletionRatio)
}

func GetModelRatioCopy() map[string]float64 {
	modelRatioLock.RLock()
	defer modelRatioLock.RUnlock()
	return copyRatioMap(ModelRatio)
}

func GetCompletionRatioCopy() map[string]float64 {
	return copyRatioMap(CompletionRatio)
}

func copyRatioMap(src map[string]float64) map[string]float64 {
	dst := make(map[string]float64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func GetCompletionRatio(originModel, mappedModel string, channelType int) float64 {
	name := normalizeRatioModelName(originModel, mappedModel)
	if key, ok := channelRatioSlugKey(name, channelType); ok {
		if ratio, ok := CompletionRatio[key]; ok {
			return ratio
		}
		if ratio, ok := DefaultCompletionRatio[key]; ok {
			return ratio
		}
	}
	if ratio, ok := CompletionRatio[name]; ok {
		return ratio
	}
	if ratio, ok := DefaultCompletionRatio[name]; ok {
		return ratio
	}
	metrics.RatioFallback.WithLabelValues(name, "completion").Inc()
	return 1
}

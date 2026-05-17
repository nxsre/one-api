package jsonmerge

import (
	"fmt"
	"strings"
)

// ApplyParamOverride 应用渠道 param_override：含 `operations` 时走 DSL；否则对根对象做深度合并。
// ctx 可为 nil；含 header 类操作时需传入 BuildParamOverrideContext 的结果，执行后头域会写回同一 map。
func ApplyParamOverride(jsonData []byte, paramOverride map[string]interface{}, ctx map[string]interface{}) ([]byte, error) {
	if len(paramOverride) == 0 {
		return jsonData, nil
	}
	if ops, ok := tryParseOperations(paramOverride); ok {
		working := jsonData
		var err error
		legacy := buildLegacyParamOverride(paramOverride)
		if len(legacy) > 0 {
			working, err = applyLegacyShallowAssign(working, legacy)
			if err != nil {
				return nil, err
			}
		}
		ctx = ensureContextMap(ctx)
		s, err := applyOperations(string(working), ops, ctx)
		if err != nil {
			return nil, err
		}
		return []byte(s), nil
	}
	return mergeJSONRootDeep(jsonData, paramOverride)
}

// BuildParamOverrideContext 构造条件/头域操作可用的上下文（不含 gin 依赖）。
func BuildParamOverrideContext(requestPath string, incomingHeaders map[string]string, channelHdr map[string]string, originModel, actualModel string) map[string]interface{} {
	ctx := make(map[string]interface{})
	rh := make(map[string]interface{})
	for k, v := range incomingHeaders {
		nk := normalizeHeaderKey(k)
		if nk != "" && strings.TrimSpace(v) != "" {
			rh[nk] = v
		}
	}
	ctx[CtxRequestHeaders] = rh
	ho := make(map[string]interface{})
	for k, v := range channelHdr {
		nk := normalizeHeaderKey(k)
		if nk != "" {
			ho[nk] = v
		}
	}
	ctx[CtxHeaderOverride] = ho
	ctx["request_path"] = requestPath
	ctx["original_model"] = originModel
	am := strings.TrimSpace(actualModel)
	om := strings.TrimSpace(originModel)
	ctx["upstream_model"] = am
	ctx["model"] = am
	if am == "" {
		ctx["model"] = om
		ctx["upstream_model"] = om
	}
	return ctx
}

// RuntimeHeaderOverrideFromContext 提取 header 覆盖结果（小写键），供合并进上游请求。
func RuntimeHeaderOverrideFromContext(ctx map[string]interface{}) map[string]string {
	if ctx == nil {
		return nil
	}
	raw, ok := ctx[CtxHeaderOverride].(map[string]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		key := normalizeHeaderKey(k)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return out
}

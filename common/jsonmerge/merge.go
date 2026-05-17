package jsonmerge

import (
	"encoding/json"
)

// MergeJSONRoot 与 ApplyParamOverride 等价（ctx 为空）。保留函数名以兼容旧调用。
func MergeJSONRoot(base []byte, patch map[string]interface{}) ([]byte, error) {
	return ApplyParamOverride(base, patch, nil)
}

func mergeJSONRootDeep(base []byte, patch map[string]interface{}) ([]byte, error) {
	if len(patch) == 0 {
		return base, nil
	}
	var root map[string]interface{}
	if err := json.Unmarshal(base, &root); err != nil {
		return nil, err
	}
	merged := deepMergeMap(root, patch)
	return json.Marshal(merged)
}

func deepMergeMap(dst, src map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(dst)+len(src))
	for k, v := range dst {
		out[k] = v
	}
	for k, v := range src {
		if v == nil {
			continue
		}
		if existing, ok := out[k]; ok {
			em, ok1 := existing.(map[string]interface{})
			vm, ok2 := v.(map[string]interface{})
			if ok1 && ok2 {
				out[k] = deepMergeMap(em, vm)
				continue
			}
		}
		out[k] = v
	}
	return out
}

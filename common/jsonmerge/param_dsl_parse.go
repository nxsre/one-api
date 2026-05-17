package jsonmerge

import (
	"encoding/json"
	"fmt"
	"strings"
)

func tryParseOperations(paramOverride map[string]interface{}) ([]ParamOperation, bool) {
	opsValue, exists := paramOverride["operations"]
	if !exists {
		return nil, false
	}

	var opMaps []map[string]interface{}
	switch ops := opsValue.(type) {
	case []interface{}:
		opMaps = make([]map[string]interface{}, 0, len(ops))
		for _, op := range ops {
			opMap, ok := op.(map[string]interface{})
			if !ok {
				return nil, false
			}
			opMaps = append(opMaps, opMap)
		}
	case []map[string]interface{}:
		opMaps = ops
	default:
		return nil, false
	}

	operations := make([]ParamOperation, 0, len(opMaps))
	for _, opMap := range opMaps {
		op := ParamOperation{}
		if path, ok := opMap["path"].(string); ok {
			op.Path = path
		}
		if mode, ok := opMap["mode"].(string); ok {
			op.Mode = mode
		} else {
			return nil, false
		}
		if value, exists := opMap["value"]; exists {
			op.Value = value
		}
		if keepOrigin, ok := opMap["keep_origin"].(bool); ok {
			op.KeepOrigin = keepOrigin
		}
		if from, ok := opMap["from"].(string); ok {
			op.From = from
		}
		if to, ok := opMap["to"].(string); ok {
			op.To = to
		}
		if logic, ok := opMap["logic"].(string); ok {
			op.Logic = logic
		} else {
			op.Logic = "OR"
		}
		if conditions, exists := opMap["conditions"]; exists {
			parsed, err := parseConditionOperations(conditions)
			if err != nil {
				return nil, false
			}
			op.Conditions = append(op.Conditions, parsed...)
		}
		operations = append(operations, op)
	}
	return operations, true
}

func buildLegacyParamOverride(paramOverride map[string]interface{}) map[string]interface{} {
	if len(paramOverride) == 0 {
		return nil
	}
	legacy := make(map[string]interface{}, len(paramOverride))
	for key, value := range paramOverride {
		if strings.EqualFold(strings.TrimSpace(key), "operations") {
			continue
		}
		legacy[key] = value
	}
	return legacy
}

func applyLegacyShallowAssign(jsonData []byte, paramOverride map[string]interface{}) ([]byte, error) {
	if len(paramOverride) == 0 {
		return jsonData, nil
	}
	var reqMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &reqMap); err != nil {
		return nil, err
	}
	for key, value := range paramOverride {
		reqMap[key] = value
	}
	return json.Marshal(reqMap)
}

func parseConditionOperations(raw interface{}) ([]ConditionOperation, error) {
	switch typed := raw.(type) {
	case map[string]interface{}:
		var conditions []ConditionOperation
		for path, val := range typed {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			conditions = append(conditions, ConditionOperation{
				Path:  path,
				Mode:  "full",
				Value: val,
			})
		}
		if len(conditions) == 0 {
			return nil, fmt.Errorf("conditions object must contain at least one key")
		}
		return conditions, nil
	case []interface{}:
		result := make([]ConditionOperation, 0, len(typed))
		for _, item := range typed {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("condition must be object")
			}
			path, _ := itemMap["path"].(string)
			mode, _ := itemMap["mode"].(string)
			if strings.TrimSpace(path) == "" || strings.TrimSpace(mode) == "" {
				return nil, fmt.Errorf("condition path/mode is required")
			}
			c := ConditionOperation{Path: path, Mode: mode}
			if value, exists := itemMap["value"]; exists {
				c.Value = value
			}
			if invert, ok := itemMap["invert"].(bool); ok {
				c.Invert = invert
			}
			if passMissingKey, ok := itemMap["pass_missing_key"].(bool); ok {
				c.PassMissingKey = passMissingKey
			}
			result = append(result, c)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("conditions must be an array or object")
	}
}

func parseReturnError(value interface{}) (*ReturnError, error) {
	re := &ReturnError{
		StatusCode: 400,
		Code:       "invalid_request",
		Type:       "invalid_request_error",
		SkipRetry:  true,
	}
	switch raw := value.(type) {
	case nil:
		return nil, fmt.Errorf("return_error value is required")
	case string:
		re.Message = strings.TrimSpace(raw)
	case map[string]interface{}:
		if message, ok := raw["message"].(string); ok {
			re.Message = strings.TrimSpace(message)
		}
		if re.Message == "" {
			if message, ok := raw["msg"].(string); ok {
				re.Message = strings.TrimSpace(message)
			}
		}
		if code, exists := raw["code"]; exists {
			re.Code = strings.TrimSpace(fmt.Sprintf("%v", code))
		}
		if errType, ok := raw["type"].(string); ok && strings.TrimSpace(errType) != "" {
			re.Type = strings.TrimSpace(errType)
		}
		if skipRetry, ok := raw["skip_retry"].(bool); ok {
			re.SkipRetry = skipRetry
		}
		if statusCodeRaw, exists := raw["status_code"]; exists {
			sc, ok := parseOverrideInt(statusCodeRaw)
			if !ok {
				return nil, fmt.Errorf("return_error status_code must be an integer")
			}
			re.StatusCode = sc
		} else if statusRaw, exists := raw["status"]; exists {
			sc, ok := parseOverrideInt(statusRaw)
			if !ok {
				return nil, fmt.Errorf("return_error status must be an integer")
			}
			re.StatusCode = sc
		}
	default:
		return nil, fmt.Errorf("return_error value must be string or object")
	}
	if re.Message == "" {
		return nil, fmt.Errorf("return_error message is required")
	}
	if re.StatusCode < 100 || re.StatusCode > 599 {
		return nil, fmt.Errorf("return_error status code out of range: %d", re.StatusCode)
	}
	return re, nil
}

func parseOverrideInt(v interface{}) (int, bool) {
	switch value := v.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		if value != float64(int(value)) {
			return 0, false
		}
		return int(value), true
	default:
		return 0, false
	}
}

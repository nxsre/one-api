package jsonmerge

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func deleteValue(jsonStr, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return jsonStr, nil
	}
	return sjson.Delete(jsonStr, path)
}

func moveValue(jsonStr, fromPath, toPath string) (string, error) {
	sourceValue := gjson.Get(jsonStr, fromPath)
	if !sourceValue.Exists() {
		return jsonStr, fmt.Errorf("source path does not exist: %s", fromPath)
	}
	result, err := sjson.Set(jsonStr, toPath, sourceValue.Value())
	if err != nil {
		return "", err
	}
	return sjson.Delete(result, fromPath)
}

func copyValue(jsonStr, fromPath, toPath string) (string, error) {
	sourceValue := gjson.Get(jsonStr, fromPath)
	if !sourceValue.Exists() {
		return jsonStr, fmt.Errorf("source path does not exist: %s", fromPath)
	}
	return sjson.Set(jsonStr, toPath, sourceValue.Value())
}

func modifyValue(jsonStr, path string, value interface{}, keepOrigin, isPrepend bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	switch {
	case current.IsArray():
		return modifyArray(jsonStr, path, value, isPrepend)
	case current.Type == gjson.String:
		return modifyString(jsonStr, path, value, isPrepend)
	case current.Type == gjson.JSON:
		return mergeObjects(jsonStr, path, value, keepOrigin)
	}
	return jsonStr, fmt.Errorf("operation not supported for type: %v", current.Type)
}

func modifyArray(jsonStr, path string, value interface{}, isPrepend bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	var newArray []interface{}
	addValue := func() {
		if arr, ok := value.([]interface{}); ok {
			newArray = append(newArray, arr...)
		} else {
			newArray = append(newArray, value)
		}
	}
	addOriginal := func() {
		current.ForEach(func(_, val gjson.Result) bool {
			newArray = append(newArray, val.Value())
			return true
		})
	}
	if isPrepend {
		addValue()
		addOriginal()
	} else {
		addOriginal()
		addValue()
	}
	return sjson.Set(jsonStr, path, newArray)
}

func modifyString(jsonStr, path string, value interface{}, isPrepend bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	valueStr := fmt.Sprintf("%v", value)
	var newStr string
	if isPrepend {
		newStr = valueStr + current.String()
	} else {
		newStr = current.String() + valueStr
	}
	return sjson.Set(jsonStr, path, newStr)
}

func trimStringValue(jsonStr, path string, value interface{}, isPrefix bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	if current.Type != gjson.String {
		return jsonStr, fmt.Errorf("operation not supported for type: %v", current.Type)
	}
	if value == nil {
		return jsonStr, fmt.Errorf("trim value is required")
	}
	valueStr := fmt.Sprintf("%v", value)
	var newStr string
	if isPrefix {
		newStr = strings.TrimPrefix(current.String(), valueStr)
	} else {
		newStr = strings.TrimSuffix(current.String(), valueStr)
	}
	return sjson.Set(jsonStr, path, newStr)
}

func ensureStringAffix(jsonStr, path string, value interface{}, isPrefix bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	if current.Type != gjson.String {
		return jsonStr, fmt.Errorf("operation not supported for type: %v", current.Type)
	}
	if value == nil {
		return jsonStr, fmt.Errorf("ensure value is required")
	}
	valueStr := fmt.Sprintf("%v", value)
	if valueStr == "" {
		return jsonStr, fmt.Errorf("ensure value is required")
	}
	currentStr := current.String()
	if isPrefix {
		if strings.HasPrefix(currentStr, valueStr) {
			return jsonStr, nil
		}
		return sjson.Set(jsonStr, path, valueStr+currentStr)
	}
	if strings.HasSuffix(currentStr, valueStr) {
		return jsonStr, nil
	}
	return sjson.Set(jsonStr, path, currentStr+valueStr)
}

func transformStringValue(jsonStr, path string, transform func(string) string) (string, error) {
	current := gjson.Get(jsonStr, path)
	if current.Type != gjson.String {
		return jsonStr, fmt.Errorf("operation not supported for type: %v", current.Type)
	}
	return sjson.Set(jsonStr, path, transform(current.String()))
}

func replaceStringValue(jsonStr, path, from, to string) (string, error) {
	current := gjson.Get(jsonStr, path)
	if current.Type != gjson.String {
		return jsonStr, fmt.Errorf("operation not supported for type: %v", current.Type)
	}
	if from == "" {
		return jsonStr, fmt.Errorf("replace from is required")
	}
	return sjson.Set(jsonStr, path, strings.ReplaceAll(current.String(), from, to))
}

func regexReplaceStringValue(jsonStr, path, pattern, replacement string) (string, error) {
	current := gjson.Get(jsonStr, path)
	if current.Type != gjson.String {
		return jsonStr, fmt.Errorf("operation not supported for type: %v", current.Type)
	}
	if pattern == "" {
		return jsonStr, fmt.Errorf("regex pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return jsonStr, err
	}
	return sjson.Set(jsonStr, path, re.ReplaceAllString(current.String(), replacement))
}

func mergeObjects(jsonStr, path string, value interface{}, keepOrigin bool) (string, error) {
	current := gjson.Get(jsonStr, path)
	var currentMap, newMap map[string]interface{}
	if err := json.Unmarshal([]byte(current.Raw), &currentMap); err != nil {
		return "", err
	}
	switch v := value.(type) {
	case map[string]interface{}:
		newMap = v
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal(jsonBytes, &newMap); err != nil {
			return "", err
		}
	}
	result := make(map[string]interface{})
	for k, v := range currentMap {
		result[k] = v
	}
	for k, v := range newMap {
		if !keepOrigin || result[k] == nil {
			result[k] = v
		}
	}
	return sjson.Set(jsonStr, path, result)
}

type pruneObjectsOptions struct {
	conditions []ConditionOperation
	logic      string
	recursive  bool
}

func parsePruneObjectsOptions(value interface{}) (pruneObjectsOptions, error) {
	opts := pruneObjectsOptions{logic: "AND", recursive: true}
	switch raw := value.(type) {
	case nil:
		return opts, fmt.Errorf("prune_objects value is required")
	case string:
		v := strings.TrimSpace(raw)
		if v == "" {
			return opts, fmt.Errorf("prune_objects value is required")
		}
		opts.conditions = []ConditionOperation{{Path: "type", Mode: "full", Value: v}}
	case map[string]interface{}:
		if logic, ok := raw["logic"].(string); ok && strings.TrimSpace(logic) != "" {
			opts.logic = logic
		}
		if recursive, ok := raw["recursive"].(bool); ok {
			opts.recursive = recursive
		}
		if condRaw, exists := raw["conditions"]; exists {
			conditions, err := parseConditionOperations(condRaw)
			if err != nil {
				return opts, err
			}
			opts.conditions = append(opts.conditions, conditions...)
		}
		if whereRaw, exists := raw["where"]; exists {
			whereMap, ok := whereRaw.(map[string]interface{})
			if !ok {
				return opts, fmt.Errorf("prune_objects where must be object")
			}
			for key, val := range whereMap {
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				opts.conditions = append(opts.conditions, ConditionOperation{Path: key, Mode: "full", Value: val})
			}
		}
		if matchType, exists := raw["type"]; exists {
			opts.conditions = append(opts.conditions, ConditionOperation{Path: "type", Mode: "full", Value: matchType})
		}
	default:
		return opts, fmt.Errorf("prune_objects value must be string or object")
	}
	if len(opts.conditions) == 0 {
		return opts, fmt.Errorf("prune_objects conditions are required")
	}
	return opts, nil
}

func pruneObjectsJSON(jsonStr, path, contextJSON string, value interface{}) (string, error) {
	options, err := parsePruneObjectsOptions(value)
	if err != nil {
		return "", err
	}
	if path == "" {
		var root interface{}
		if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
			return "", err
		}
		cleaned, _, err := pruneObjectsNode(root, options, contextJSON, true)
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(cleaned)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	target := gjson.Get(jsonStr, path)
	if !target.Exists() {
		return jsonStr, nil
	}
	var targetNode interface{}
	if target.Type == gjson.JSON {
		if err := json.Unmarshal([]byte(target.Raw), &targetNode); err != nil {
			return "", err
		}
	} else {
		targetNode = target.Value()
	}
	cleaned, _, err := pruneObjectsNode(targetNode, options, contextJSON, true)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(cleaned)
	if err != nil {
		return "", err
	}
	out, err := sjson.SetRaw(jsonStr, path, string(b))
	if err != nil {
		return "", err
	}
	return out, nil
}

func pruneObjectsNode(node interface{}, options pruneObjectsOptions, contextJSON string, isRoot bool) (interface{}, bool, error) {
	switch value := node.(type) {
	case []interface{}:
		result := make([]interface{}, 0, len(value))
		for _, item := range value {
			next, drop, err := pruneObjectsNode(item, options, contextJSON, false)
			if err != nil {
				return nil, false, err
			}
			if drop {
				continue
			}
			result = append(result, next)
		}
		return result, false, nil
	case map[string]interface{}:
		shouldDrop, err := shouldPruneObject(value, options, contextJSON)
		if err != nil {
			return nil, false, err
		}
		if shouldDrop && !isRoot {
			return nil, true, nil
		}
		if !options.recursive {
			return value, false, nil
		}
		for key, child := range value {
			next, drop, err := pruneObjectsNode(child, options, contextJSON, false)
			if err != nil {
				return nil, false, err
			}
			if drop {
				delete(value, key)
				continue
			}
			value[key] = next
		}
		return value, false, nil
	default:
		return node, false, nil
	}
}

func shouldPruneObject(node map[string]interface{}, options pruneObjectsOptions, contextJSON string) (bool, error) {
	nodeBytes, err := json.Marshal(node)
	if err != nil {
		return false, err
	}
	return checkConditions(string(nodeBytes), contextJSON, options.conditions, options.logic)
}

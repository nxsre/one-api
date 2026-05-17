package jsonmerge

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

var negativeIndexRegexp = regexp.MustCompile(`\.(-\d+)`)

func checkConditions(jsonStr, contextJSON string, conditions []ConditionOperation, logic string) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}
	results := make([]bool, len(conditions))
	for i, condition := range conditions {
		result, err := checkSingleCondition(jsonStr, contextJSON, condition)
		if err != nil {
			return false, err
		}
		results[i] = result
	}
	if strings.ToUpper(logic) == "AND" {
		for _, r := range results {
			if !r {
				return false, nil
			}
		}
		return true, nil
	}
	for _, r := range results {
		if r {
			return true, nil
		}
	}
	return false, nil
}

func checkSingleCondition(jsonStr, contextJSON string, condition ConditionOperation) (bool, error) {
	path := processNegativeIndex(jsonStr, condition.Path)
	value := gjson.Get(jsonStr, path)
	if !value.Exists() && contextJSON != "" {
		value = gjson.Get(contextJSON, path)
	}
	if !value.Exists() {
		if condition.PassMissingKey {
			return true, nil
		}
		return false, nil
	}
	targetBytes, err := json.Marshal(condition.Value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal condition value: %v", err)
	}
	targetValue := gjson.ParseBytes(targetBytes)
	result, err := compareGjsonValues(value, targetValue, strings.ToLower(condition.Mode))
	if err != nil {
		return false, fmt.Errorf("comparison failed for path %s: %v", condition.Path, err)
	}
	if condition.Invert {
		result = !result
	}
	return result, nil
}

func processNegativeIndex(jsonStr string, path string) string {
	matches := negativeIndexRegexp.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return path
	}
	result := path
	for _, match := range matches {
		negIndex := match[1]
		index, _ := strconv.Atoi(negIndex)
		idx := strings.Index(path, negIndex)
		if idx < 0 {
			continue
		}
		arrayPath := path[:idx]
		if strings.HasSuffix(arrayPath, ".") {
			arrayPath = arrayPath[:len(arrayPath)-1]
		}
		array := gjson.Get(jsonStr, arrayPath)
		if array.IsArray() {
			length := len(array.Array())
			actualIndex := length + index
			if actualIndex >= 0 && actualIndex < length {
				result = strings.Replace(result, match[0], "."+strconv.Itoa(actualIndex), 1)
			}
		}
	}
	return result
}

func compareGjsonValues(jsonValue, targetValue gjson.Result, mode string) (bool, error) {
	switch mode {
	case "full":
		return compareEqual(jsonValue, targetValue)
	case "prefix":
		return strings.HasPrefix(jsonValue.String(), targetValue.String()), nil
	case "suffix":
		return strings.HasSuffix(jsonValue.String(), targetValue.String()), nil
	case "contains":
		return strings.Contains(jsonValue.String(), targetValue.String()), nil
	case "gt":
		return compareNumeric(jsonValue, targetValue, "gt")
	case "gte":
		return compareNumeric(jsonValue, targetValue, "gte")
	case "lt":
		return compareNumeric(jsonValue, targetValue, "lt")
	case "lte":
		return compareNumeric(jsonValue, targetValue, "lte")
	default:
		return false, fmt.Errorf("unsupported comparison mode: %s", mode)
	}
}

func compareEqual(jsonValue, targetValue gjson.Result) (bool, error) {
	if jsonValue.Type == gjson.Null || targetValue.Type == gjson.Null {
		return jsonValue.Type == gjson.Null && targetValue.Type == gjson.Null, nil
	}
	if (jsonValue.Type == gjson.True || jsonValue.Type == gjson.False) &&
		(targetValue.Type == gjson.True || targetValue.Type == gjson.False) {
		return jsonValue.Bool() == targetValue.Bool(), nil
	}
	if jsonValue.Type != targetValue.Type {
		return false, fmt.Errorf("compare for different types, got %v and %v", jsonValue.Type, targetValue.Type)
	}
	switch jsonValue.Type {
	case gjson.True, gjson.False:
		return jsonValue.Bool() == targetValue.Bool(), nil
	case gjson.Number:
		return jsonValue.Num == targetValue.Num, nil
	case gjson.String:
		return jsonValue.String() == targetValue.String(), nil
	default:
		return jsonValue.String() == targetValue.String(), nil
	}
}

func compareNumeric(jsonValue, targetValue gjson.Result, operator string) (bool, error) {
	if jsonValue.Type != gjson.Number || targetValue.Type != gjson.Number {
		return false, fmt.Errorf("numeric comparison requires both values to be numbers, got %v and %v", jsonValue.Type, targetValue.Type)
	}
	jsonNum := jsonValue.Num
	targetNum := targetValue.Num
	switch operator {
	case "gt":
		return jsonNum > targetNum, nil
	case "gte":
		return jsonNum >= targetNum, nil
	case "lt":
		return jsonNum < targetNum, nil
	case "lte":
		return jsonNum <= targetNum, nil
	default:
		return false, fmt.Errorf("unsupported numeric operator: %s", operator)
	}
}

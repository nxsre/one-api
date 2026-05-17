package jsonmerge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func ensureContextMap(conditionContext map[string]interface{}) map[string]interface{} {
	if conditionContext != nil {
		return conditionContext
	}
	return make(map[string]interface{})
}

func marshalContextJSON(context map[string]interface{}) (string, error) {
	if context == nil || len(context) == 0 {
		return "", nil
	}
	b, err := json.Marshal(context)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ensureMapKeyInContext(context map[string]interface{}, key string) map[string]interface{} {
	if context == nil {
		return map[string]interface{}{}
	}
	if existing, ok := context[key]; ok {
		if mapVal, ok := existing.(map[string]interface{}); ok {
			return mapVal
		}
	}
	result := make(map[string]interface{})
	context[key] = result
	return result
}

func normalizeHeaderKey(key string) string {
	return strings.TrimSpace(strings.ToLower(key))
}

func getHeaderValueFromContext(context map[string]interface{}, headerName string) (string, bool) {
	headerName = normalizeHeaderKey(headerName)
	if headerName == "" {
		return "", false
	}
	for _, key := range []string{CtxHeaderOverride, CtxRequestHeaders} {
		source := ensureMapKeyInContext(context, key)
		raw, ok := source[headerName]
		if !ok {
			continue
		}
		value := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func setHeaderOverrideInContext(context map[string]interface{}, headerName string, value interface{}, keepOrigin bool) error {
	headerName = normalizeHeaderKey(headerName)
	if headerName == "" {
		return fmt.Errorf("header name is required")
	}
	rawHeaders := ensureMapKeyInContext(context, CtxHeaderOverride)
	if keepOrigin {
		if existing, ok := rawHeaders[headerName]; ok {
			if strings.TrimSpace(fmt.Sprintf("%v", existing)) != "" {
				return nil
			}
		}
	}
	headerValue, hasValue, err := resolveHeaderOverrideValue(context, headerName, value)
	if err != nil {
		return err
	}
	if !hasValue {
		delete(rawHeaders, headerName)
		return nil
	}
	rawHeaders[headerName] = headerValue
	return nil
}

func resolveHeaderOverrideValue(context map[string]interface{}, headerName string, value interface{}) (string, bool, error) {
	if value == nil {
		return "", false, fmt.Errorf("header value is required")
	}
	if mapping, ok := value.(map[string]interface{}); ok {
		return resolveHeaderOverrideValueByMapping(context, headerName, mapping)
	}
	if mapping, ok := value.(map[string]string); ok {
		converted := make(map[string]interface{}, len(mapping))
		for k, item := range mapping {
			converted[k] = item
		}
		return resolveHeaderOverrideValueByMapping(context, headerName, converted)
	}
	headerValue := strings.TrimSpace(fmt.Sprintf("%v", value))
	if headerValue == "" {
		return "", false, nil
	}
	return headerValue, true, nil
}

func headerUniqTokens(tokens []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func splitHeaderListValue(raw string) []string {
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		t := strings.TrimSpace(item)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func parseHeaderReplacementTokens(value interface{}) ([]string, error) {
	switch raw := value.(type) {
	case nil:
		return nil, nil
	case string:
		return splitHeaderListValue(raw), nil
	case []string:
		tokens := make([]string, 0, len(raw))
		for _, item := range raw {
			tokens = append(tokens, splitHeaderListValue(item)...)
		}
		return headerUniqTokens(tokens), nil
	case []interface{}:
		tokens := make([]string, 0, len(raw))
		for _, item := range raw {
			sub, err := parseHeaderReplacementTokens(item)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, sub...)
		}
		return headerUniqTokens(tokens), nil
	default:
		token := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if token == "" {
			return nil, nil
		}
		return []string{token}, nil
	}
}

func parseHeaderAppendTokens(mapping map[string]interface{}) ([]string, error) {
	appendRaw, ok := mapping["$append"]
	if !ok {
		return nil, nil
	}
	return parseHeaderReplacementTokens(appendRaw)
}

func parseHeaderKeepOnlyDeclared(mapping map[string]interface{}) bool {
	keepOnlyDeclaredRaw, ok := mapping["$keep_only_declared"]
	if !ok {
		return false
	}
	b, ok := keepOnlyDeclaredRaw.(bool)
	return ok && b
}

func resolveHeaderOverrideValueByMapping(context map[string]interface{}, headerName string, mapping map[string]interface{}) (string, bool, error) {
	if len(mapping) == 0 {
		return "", false, fmt.Errorf("header value mapping cannot be empty")
	}
	appendTokens, err := parseHeaderAppendTokens(mapping)
	if err != nil {
		return "", false, err
	}
	keepOnlyDeclared := parseHeaderKeepOnlyDeclared(mapping)
	sourceValue, exists := getHeaderValueFromContext(context, headerName)
	sourceTokens := make([]string, 0)
	if exists {
		sourceTokens = splitHeaderListValue(sourceValue)
	}
	wildcardValue, hasWildcard := mapping["*"]
	resultTokens := make([]string, 0, len(sourceTokens)+len(appendTokens))
	for _, token := range sourceTokens {
		replacementRaw, hasReplacement := mapping[token]
		if !hasReplacement && hasWildcard && !keepOnlyDeclared {
			replacementRaw = wildcardValue
			hasReplacement = true
		}
		if !hasReplacement {
			if keepOnlyDeclared {
				continue
			}
			resultTokens = append(resultTokens, token)
			continue
		}
		replacementTokens, err := parseHeaderReplacementTokens(replacementRaw)
		if err != nil {
			return "", false, err
		}
		resultTokens = append(resultTokens, replacementTokens...)
	}
	resultTokens = append(resultTokens, appendTokens...)
	resultTokens = headerUniqTokens(resultTokens)
	if len(resultTokens) == 0 {
		return "", false, nil
	}
	return strings.Join(resultTokens, ","), true, nil
}

func deleteHeaderOverrideInContext(context map[string]interface{}, headerName string) error {
	headerName = normalizeHeaderKey(headerName)
	if headerName == "" {
		return fmt.Errorf("header name is required")
	}
	rawHeaders := ensureMapKeyInContext(context, CtxHeaderOverride)
	delete(rawHeaders, headerName)
	return nil
}

func copyHeaderInContext(context map[string]interface{}, fromHeader, toHeader string, keepOrigin bool) error {
	fromHeader = normalizeHeaderKey(fromHeader)
	toHeader = normalizeHeaderKey(toHeader)
	if fromHeader == "" || toHeader == "" {
		return fmt.Errorf("copy_header from/to is required")
	}
	value, exists := getHeaderValueFromContext(context, fromHeader)
	if !exists {
		return fmt.Errorf("%w: %s", errSourceHeaderNotFound, fromHeader)
	}
	return setHeaderOverrideInContext(context, toHeader, value, keepOrigin)
}

func moveHeaderInContext(context map[string]interface{}, fromHeader, toHeader string, keepOrigin bool) error {
	fromHeader = normalizeHeaderKey(fromHeader)
	toHeader = normalizeHeaderKey(toHeader)
	if fromHeader == "" || toHeader == "" {
		return fmt.Errorf("move_header from/to is required")
	}
	if err := copyHeaderInContext(context, fromHeader, toHeader, keepOrigin); err != nil {
		return err
	}
	if strings.EqualFold(fromHeader, toHeader) {
		return nil
	}
	return deleteHeaderOverrideInContext(context, fromHeader)
}

func parseHeaderPassThroughNames(value interface{}) ([]string, error) {
	normalizeNames := func(values []string) []string {
		var names []string
		for _, item := range values {
			headerName := normalizeHeaderKey(item)
			if headerName != "" {
				names = append(names, headerName)
			}
		}
		return headerUniqTokens(names)
	}
	switch raw := value.(type) {
	case nil:
		return nil, fmt.Errorf("pass_headers value is required")
	case string:
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return nil, fmt.Errorf("pass_headers value is required")
		}
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
			var parsed interface{}
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				return parseHeaderPassThroughNames(parsed)
			}
		}
		names := normalizeNames(strings.Split(trimmed, ","))
		if len(names) == 0 {
			return nil, fmt.Errorf("pass_headers value is invalid")
		}
		return names, nil
	case []interface{}:
		var names []string
		for _, item := range raw {
			headerName := normalizeHeaderKey(fmt.Sprintf("%v", item))
			if headerName != "" {
				names = append(names, headerName)
			}
		}
		names = headerUniqTokens(names)
		if len(names) == 0 {
			return nil, fmt.Errorf("pass_headers value is invalid")
		}
		return names, nil
	case []string:
		names := normalizeNames(raw)
		if len(names) == 0 {
			return nil, fmt.Errorf("pass_headers value is invalid")
		}
		return names, nil
	case map[string]interface{}:
		var candidates []string
		if headersRaw, ok := raw["headers"]; ok {
			names, err := parseHeaderPassThroughNames(headersRaw)
			if err == nil {
				candidates = append(candidates, names...)
			}
		}
		if namesRaw, ok := raw["names"]; ok {
			names, err := parseHeaderPassThroughNames(namesRaw)
			if err == nil {
				candidates = append(candidates, names...)
			}
		}
		if headerRaw, ok := raw["header"]; ok {
			names, err := parseHeaderPassThroughNames(headerRaw)
			if err == nil {
				candidates = append(candidates, names...)
			}
		}
		names := normalizeNames(candidates)
		if len(names) == 0 {
			return nil, fmt.Errorf("pass_headers value is invalid")
		}
		return names, nil
	default:
		return nil, fmt.Errorf("pass_headers value must be string, array or object")
	}
}

type syncTarget struct {
	kind string
	key  string
}

func parseSyncTarget(spec string) (syncTarget, error) {
	raw := strings.TrimSpace(spec)
	if raw == "" {
		return syncTarget{}, fmt.Errorf("sync_fields target is required")
	}
	idx := strings.Index(raw, ":")
	if idx < 0 {
		return syncTarget{kind: "json", key: raw}, nil
	}
	kind := strings.ToLower(strings.TrimSpace(raw[:idx]))
	key := strings.TrimSpace(raw[idx+1:])
	if key == "" {
		return syncTarget{}, fmt.Errorf("sync_fields target key is required: %s", raw)
	}
	switch kind {
	case "json", "body":
		return syncTarget{kind: "json", key: key}, nil
	case "header":
		return syncTarget{kind: "header", key: key}, nil
	default:
		return syncTarget{}, fmt.Errorf("sync_fields target prefix is invalid: %s", raw)
	}
}

func readSyncTargetValue(jsonStr string, context map[string]interface{}, target syncTarget) (interface{}, bool, error) {
	switch target.kind {
	case "json":
		path := processNegativeIndex(jsonStr, target.key)
		value := gjson.Get(jsonStr, path)
		if !value.Exists() || value.Type == gjson.Null {
			return nil, false, nil
		}
		if value.Type == gjson.String && strings.TrimSpace(value.String()) == "" {
			return nil, false, nil
		}
		return value.Value(), true, nil
	case "header":
		value, ok := getHeaderValueFromContext(context, target.key)
		if !ok || strings.TrimSpace(value) == "" {
			return nil, false, nil
		}
		return value, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported sync_fields target kind: %s", target.kind)
	}
}

func writeSyncTargetValue(jsonStr string, context map[string]interface{}, target syncTarget, value interface{}) (string, error) {
	switch target.kind {
	case "json":
		path := processNegativeIndex(jsonStr, target.key)
		nextJSON, err := sjson.Set(jsonStr, path, value)
		if err != nil {
			return "", err
		}
		return nextJSON, nil
	case "header":
		if err := setHeaderOverrideInContext(context, target.key, value, false); err != nil {
			return "", err
		}
		return jsonStr, nil
	default:
		return "", fmt.Errorf("unsupported sync_fields target kind: %s", target.kind)
	}
}

func syncFieldsBetweenTargets(jsonStr string, context map[string]interface{}, fromSpec string, toSpec string) (string, error) {
	fromTarget, err := parseSyncTarget(fromSpec)
	if err != nil {
		return "", err
	}
	toTarget, err := parseSyncTarget(toSpec)
	if err != nil {
		return "", err
	}
	fromValue, fromExists, err := readSyncTargetValue(jsonStr, context, fromTarget)
	if err != nil {
		return "", err
	}
	toValue, toExists, err := readSyncTargetValue(jsonStr, context, toTarget)
	if err != nil {
		return "", err
	}
	if fromExists && !toExists {
		return writeSyncTargetValue(jsonStr, context, toTarget, fromValue)
	}
	if toExists && !fromExists {
		return writeSyncTargetValue(jsonStr, context, fromTarget, toValue)
	}
	return jsonStr, nil
}

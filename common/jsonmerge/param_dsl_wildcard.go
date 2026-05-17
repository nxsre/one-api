package jsonmerge

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

func isPathBasedOperation(mode string) bool {
	switch mode {
	case "delete", "set", "prepend", "append", "trim_prefix", "trim_suffix", "ensure_prefix", "ensure_suffix",
		"trim_space", "to_lower", "to_upper", "replace", "regex_replace", "prune_objects":
		return true
	default:
		return false
	}
}

func resolveOperationPaths(jsonStr, path string) ([]string, error) {
	if !strings.Contains(path, "*") {
		return []string{path}, nil
	}
	return expandWildcardPaths(jsonStr, path)
}

func expandWildcardPaths(jsonStr, path string) ([]string, error) {
	var root interface{}
	if err := json.Unmarshal([]byte(jsonStr), &root); err != nil {
		return nil, err
	}
	segments := strings.Split(path, ".")
	paths := collectWildcardPaths(root, segments, nil)
	return uniqStrings(paths), nil
}

func uniqStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func collectWildcardPaths(node interface{}, segments []string, prefix []string) []string {
	if len(segments) == 0 {
		return []string{strings.Join(prefix, ".")}
	}
	segment := strings.TrimSpace(segments[0])
	if segment == "" {
		return nil
	}
	isLast := len(segments) == 1

	if segment == "*" {
		switch typed := node.(type) {
		case map[string]interface{}:
			keys := make([]string, 0, len(typed))
			for k := range typed {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var out []string
			for _, key := range keys {
				out = append(out, collectWildcardPaths(typed[key], segments[1:], append(prefix, key))...)
			}
			return out
		case []interface{}:
			var out []string
			for i := range typed {
				out = append(out, collectWildcardPaths(typed[i], segments[1:], append(prefix, strconv.Itoa(i)))...)
			}
			return out
		default:
			return nil
		}
	}

	switch typed := node.(type) {
	case map[string]interface{}:
		if isLast {
			return []string{strings.Join(append(prefix, segment), ".")}
		}
		next, exists := typed[segment]
		if !exists {
			return nil
		}
		return collectWildcardPaths(next, segments[1:], append(prefix, segment))
	case []interface{}:
		index, err := strconv.Atoi(segment)
		if err != nil || index < 0 || index >= len(typed) {
			return nil
		}
		if isLast {
			return []string{strings.Join(append(prefix, segment), ".")}
		}
		return collectWildcardPaths(typed[index], segments[1:], append(prefix, segment))
	default:
		return nil
	}
}

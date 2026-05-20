package controller

import "strings"

const basellmDefaultBaseURL = "https://basellm.github.io/llm-metadata"

// basellmAPIURL 解析 BaseLLM LLM Metadata 根地址，默认 …/api/all.json。
func basellmAPIURL(baseFromReq string) string {
	s := strings.TrimSpace(baseFromReq)
	if s == "" {
		return basellmDefaultBaseURL + "/api/all.json"
	}
	s = strings.TrimRight(s, "/")
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "/api/all.json") || strings.HasSuffix(lower, "/all.json") {
		return s
	}
	if strings.HasSuffix(lower, "/api.json") {
		return strings.TrimSuffix(s, "/api.json") + "/api/all.json"
	}
	return s + "/api/all.json"
}

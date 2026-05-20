package anthropic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SystemPrompt is the Anthropic Messages API "system" field: a string or content block array.
type SystemPrompt string

func (s SystemPrompt) String() string {
	return string(s)
}

func (s *SystemPrompt) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*s = ""
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = SystemPrompt(str)
		return nil
	}
	var blocks []Content
	if err := json.Unmarshal(data, &blocks); err == nil {
		var sb strings.Builder
		for _, b := range blocks {
			if t := strings.TrimSpace(b.Text); t != "" {
				if sb.Len() > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString(t)
			}
		}
		*s = SystemPrompt(sb.String())
		return nil
	}
	return fmt.Errorf("system must be a string or array of content blocks")
}

func (s SystemPrompt) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

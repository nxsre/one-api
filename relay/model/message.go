package model

type Message struct {
	Role             string  `json:"role,omitempty"`
	Content          any     `json:"content,omitempty"`
	ReasoningContent any     `json:"reasoning_content,omitempty"`
	Name             *string `json:"name,omitempty"`
	ToolCalls        []Tool  `json:"tool_calls,omitempty"`
	ToolCallId       string  `json:"tool_call_id,omitempty"`
}

func (m Message) IsStringContent() bool {
	_, ok := m.Content.(string)
	return ok
}

func (m Message) StringContent() string {
	content, ok := m.Content.(string)
	if ok {
		return content
	}
	contentList, ok := m.Content.([]any)
	if ok {
		var contentStr string
		for _, contentItem := range contentList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] == ContentTypeText {
				if subStr, ok := contentMap["text"].(string); ok {
					contentStr += subStr
				}
			}
		}
		return contentStr
	}
	return ""
}

func (m Message) ParseContent() []MessageContent {
	var contentList []MessageContent
	content, ok := m.Content.(string)
	if ok {
		contentList = append(contentList, MessageContent{
			Type: ContentTypeText,
			Text: content,
		})
		return contentList
	}
	anyList, ok := m.Content.([]any)
	if ok {
		for _, contentItem := range anyList {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			switch contentMap["type"] {
			case ContentTypeText:
				if subStr, ok := contentMap["text"].(string); ok {
					contentList = append(contentList, MessageContent{
						Type: ContentTypeText,
						Text: subStr,
					})
				}
			case ContentTypeImageURL:
				if subObj, ok := contentMap["image_url"].(map[string]any); ok {
					contentList = append(contentList, MessageContent{
						Type: ContentTypeImageURL,
						ImageURL: &ImageURL{
							Url: subObj["url"].(string),
						},
					})
				}
			case ContentTypeInputAudio:
				if subObj, ok := contentMap["input_audio"].(map[string]any); ok {
					ia := &InputAudio{}
					if data, ok := subObj["data"].(string); ok {
						ia.Data = data
					}
					if format, ok := subObj["format"].(string); ok {
						ia.Format = format
					}
					contentList = append(contentList, MessageContent{
						Type:       ContentTypeInputAudio,
						InputAudio: ia,
					})
				}
			case ContentTypeInputFile:
				if subObj, ok := contentMap["input_file"].(map[string]any); ok {
					f := &InputFile{}
					if v, ok := subObj["filename"].(string); ok {
						f.Filename = v
					}
					if v, ok := subObj["file_data"].(string); ok {
						f.FileData = v
					}
					if v, ok := subObj["file_id"].(string); ok {
						f.FileID = v
					}
					contentList = append(contentList, MessageContent{
						Type: ContentTypeInputFile,
						File: f,
					})
				}
			}
		}
		return contentList
	}
	return nil
}

type ImageURL struct {
	Url    string `json:"url,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type InputAudio struct {
	Data   string `json:"data,omitempty"`
	Format string `json:"format,omitempty"`
}

type InputFile struct {
	Filename string `json:"filename,omitempty"`
	FileData string `json:"file_data,omitempty"`
	FileID   string `json:"file_id,omitempty"`
}

type MessageContent struct {
	Type       string      `json:"type,omitempty"`
	Text       string      `json:"text"`
	ImageURL   *ImageURL   `json:"image_url,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
	File       *InputFile  `json:"input_file,omitempty"`
}

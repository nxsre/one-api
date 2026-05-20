package model

import "strings"

// ContentPartsToOpenAIArray 将 MessageContent 列表转为 OpenAI chat content part 数组。
func ContentPartsToOpenAIArray(parts []MessageContent) []map[string]any {
	out := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case ContentTypeText:
			out = append(out, map[string]any{"type": "text", "text": p.Text})
		case ContentTypeImageURL:
			if p.ImageURL != nil {
				out = append(out, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": p.ImageURL.Url,
					},
				})
			}
		case ContentTypeInputAudio:
			if p.InputAudio != nil {
				out = append(out, map[string]any{
					"type": "input_audio",
					"input_audio": map[string]any{
						"data":   p.InputAudio.Data,
						"format": p.InputAudio.Format,
					},
				})
			}
		case ContentTypeInputFile:
			if p.File != nil {
				part := map[string]any{"type": "input_file"}
				fileObj := map[string]any{}
				if p.File.Filename != "" {
					fileObj["filename"] = p.File.Filename
				}
				if p.File.FileData != "" {
					fileObj["file_data"] = p.File.FileData
				}
				if p.File.FileID != "" {
					fileObj["file_id"] = p.File.FileID
				}
				part["input_file"] = fileObj
				out = append(out, part)
			}
		}
	}
	return out
}

// IsImageMimeType 判断 MIME 是否为图片类型。
func IsImageMimeType(mime string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/")
}

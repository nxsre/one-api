package model

import "testing"

func TestContentPartsToOpenAIArray(t *testing.T) {
	parts := []MessageContent{
		{Type: ContentTypeText, Text: "hi"},
		{Type: ContentTypeImageURL, ImageURL: &ImageURL{Url: "https://example.com/a.png"}},
		{Type: ContentTypeInputFile, File: &InputFile{Filename: "doc.pdf", FileData: "data:application/pdf;base64,abc"}},
	}
	arr := ContentPartsToOpenAIArray(parts)
	if len(arr) != 3 {
		t.Fatalf("len=%d want 3", len(arr))
	}
	if arr[0]["type"] != "text" {
		t.Fatalf("first type=%v", arr[0]["type"])
	}
	if arr[1]["type"] != "image_url" {
		t.Fatalf("second type=%v", arr[1]["type"])
	}
	if arr[2]["type"] != "input_file" {
		t.Fatalf("third type=%v", arr[2]["type"])
	}
}

func TestIsImageMimeType(t *testing.T) {
	if !IsImageMimeType("image/png") {
		t.Fatal("image/png should be image")
	}
	if IsImageMimeType("application/pdf") {
		t.Fatal("pdf should not be image")
	}
}

func TestParseContentInputAudioAndFile(t *testing.T) {
	msg := Message{
		Content: []any{
			map[string]any{
				"type": "input_audio",
				"input_audio": map[string]any{
					"data":   "abc",
					"format": "wav",
				},
			},
			map[string]any{
				"type": "input_file",
				"input_file": map[string]any{
					"filename":  "f.pdf",
					"file_data": "data:application/pdf;base64,xyz",
				},
			},
		},
	}
	parts := msg.ParseContent()
	if len(parts) != 2 {
		t.Fatalf("len=%d want 2", len(parts))
	}
	if parts[0].Type != ContentTypeInputAudio || parts[0].InputAudio.Format != "wav" {
		t.Fatalf("audio part=%+v", parts[0])
	}
	if parts[1].Type != ContentTypeInputFile || parts[1].File.Filename != "f.pdf" {
		t.Fatalf("file part=%+v", parts[1])
	}
}

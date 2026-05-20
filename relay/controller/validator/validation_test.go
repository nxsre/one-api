package validator

import (
	"testing"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func TestValidateEmbeddingsRequiresInput(t *testing.T) {
	req := &relaymodel.GeneralOpenAIRequest{Model: "text-embedding-3-small"}
	if err := ValidateTextRequest(req, relaymode.Embeddings); err == nil {
		t.Fatal("expected error for empty input")
	}
	req.Input = "hello"
	if err := ValidateTextRequest(req, relaymode.Embeddings); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmbeddingInputEmpty(t *testing.T) {
	if !embeddingInputEmpty(nil) {
		t.Fatal("nil should be empty")
	}
	if !embeddingInputEmpty("") {
		t.Fatal("blank string should be empty")
	}
	if embeddingInputEmpty("x") {
		t.Fatal("non-empty string should not be empty")
	}
}

package controller

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildGeminiModelsListURL(t *testing.T) {
	Convey("buildGeminiModelsListURL", t, func() {
		So(buildGeminiModelsListURL("", "", "test-key"), ShouldEqual,
			"https://generativelanguage.googleapis.com/v1beta/models?key=test-key")
		So(buildGeminiModelsListURL("https://www.anyfast.ai", "v1beta", "sk-abc"), ShouldEqual,
			"https://www.anyfast.ai/v1beta/models?key=sk-abc")
		So(buildGeminiModelsListURL("https://proxy.example/", "v1", "k"), ShouldEqual,
			"https://proxy.example/v1/models?key=k")
	})
}

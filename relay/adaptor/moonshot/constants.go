package moonshot

// ModelList 与 https://platform.kimi.com/docs/pricing/chat 公开模型保持一致。
// 倍率定义见 relay/billing/ratio/model.go 中的 ModelRatio / CompletionRatio。
var ModelList = []string{
	// Moonshot V1
	"moonshot-v1-8k",
	"moonshot-v1-32k",
	"moonshot-v1-128k",
	"moonshot-v1-auto",
	"moonshot-v1-8k-vision-preview",
	"moonshot-v1-32k-vision-preview",
	"moonshot-v1-128k-vision-preview",
	// Kimi K2
	"kimi-k2-0711-preview",
	"kimi-k2-0905-preview",
	"kimi-k2-turbo-preview",
	"kimi-k2-thinking",
	"kimi-k2-thinking-turbo",
	"kimi-k2.5",
	// Kimi K2.6
	"kimi-k2.6",
}

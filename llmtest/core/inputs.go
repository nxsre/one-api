package core

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math/rand"
)

// solidColors 用于视觉场景：纯色图片 + 期望模型说出的颜色词。
var solidColors = []struct {
	Name string
	RGBA color.RGBA
	Word string // 期望响应包含的颜色词（英文，跨模型更稳定）
}{
	{"red", color.RGBA{220, 20, 20, 255}, "red"},
	{"green", color.RGBA{20, 180, 60, 255}, "green"},
	{"blue", color.RGBA{30, 60, 220, 255}, "blue"},
}

// makeSolidImagePNG 生成 size×size 的纯色 PNG，返回 base64（不含 data: 前缀）。
// 用 image/png 现场生成，无需联网取图，保证视觉断言可离线、可复现。
func makeSolidImagePNG(c color.RGBA, size int) string {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// pick 从切片中按 rng 取一个元素。
func pick[T any](rng *rand.Rand, items []T) T {
	return items[rng.Intn(len(items))]
}

// factPrompts 简单事实问答：提示 + 期望包含的答案。
var factPrompts = []struct {
	Q string
	A string
}{
	{"What is the capital of France? Answer with only the city name.", "paris"},
	{"What is the capital of Japan? Answer with only the city name.", "tokyo"},
	{"What is the chemical symbol for water? Answer with only the formula.", "h2o"},
	{"How many continents are there on Earth? Answer with only the number.", "7"},
	{"What color do you get by mixing blue and yellow? Answer with one word.", "green"},
}

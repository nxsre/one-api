package nacosdist

import "embed"

// 由 main 注入，供 controller 判断原生控制台是否已随二进制/Docker 打入 dist。
var bound embed.FS

// Bind 在 router 初始化前调用，绑定 go:embed 的 Nacos 控制台 dist。
func Bind(f embed.FS) {
	bound = f
}

// UIBundled 为 true 表示 embed 中存在 console-ui-next 构建产物（含 js/main.js）。
func UIBundled() bool {
	return bundleReady(bound)
}

func bundleReady(f embed.FS) bool {
	_, err := f.ReadFile("web/nacos-console/dist/js/main.js")
	return err == nil
}

// BundleReady 与 UIBundled 相同，供 Mount 使用显式传入的 FS（避免 Bind 顺序问题）。
func BundleReady(f embed.FS) bool {
	return bundleReady(f)
}

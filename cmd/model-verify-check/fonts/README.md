# 内嵌字体

PDF 验真报告所用字体，随二进制一起分发（`report_pdf.go` 中用 `//go:embed` 打包），
运行时无需依赖系统字体。

| 文件 | 角色 | 字体 | 覆盖 | 来源 / 许可 |
|------|------|------|------|-------------|
| `FangSong.ttf` | 中日文正文 | 全 Unicode 仿宋 | 汉字 + 日文假名 + 全角标点 | https://github.com/dolbydu/font （`unicode/FangSong.ttf`） |
| `Times.ttf` | 拉丁/数字 | Tinos-Regular | 拉丁字母、数字、常用标点 | https://github.com/google/fonts （`ofl/tinos`，SIL OFL 1.1） |

Tinos 与 Times New Roman 字宽一致，是其开源替身。

仿宋不含韩文谚文等字形，由可选的系统兜底字体补齐（`-font` / `MVR_FONT` / 自动探测，
见 `report_pdf.go` 的 `reportFontCandidates`）；兜底缺失时这些字符会被优雅跳过，报告仍可正常生成。

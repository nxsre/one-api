package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/signintech/gopdf"
)

// 内嵌字体：仿宋（中日文正文）+ Times New Roman 等价体 Tinos（拉丁/数字）。
// 二者均随二进制一起分发，无需依赖运行环境的系统字体。
// FangSong.ttf：全 Unicode 仿宋，覆盖中文与日文假名。
// Times.ttf：Tinos-Regular（OFL，与 Times New Roman 字宽一致的开源替身）。
//
//go:embed fonts/FangSong.ttf
var fangSongTTF []byte

//go:embed fonts/Times.ttf
var timesTTF []byte

// PDF 模型质量验真报告：把 A–S 全套探测结果汇总为一份可对外展示的客观报告。
// 设计目标：官方、克制、如实呈现模型在检测时点的真实响应，不作营销化修饰。

type rgb struct{ r, g, b uint8 }

var (
	cNavy  = rgb{15, 23, 42}
	cInk   = rgb{31, 41, 55}
	cMuted = rgb{107, 114, 128}
	cGreen = rgb{22, 163, 74}
	cAmber = rgb{202, 138, 4}
	cRed   = rgb{220, 38, 38}
	cRule  = rgb{226, 232, 240}
	cChip  = rgb{241, 245, 249}
	cWhite = rgb{255, 255, 255}
	cSlate = rgb{203, 213, 225}
)

const (
	// 三个字体族：仿宋（中日文）、Times（拉丁/数字）、fallback（前两者都缺字时的系统兜底，如韩文谚文）。
	fFang     = "fangsong"
	fTimes    = "times"
	fFallback = "fallback"

	pageW    = 595.0
	marginL  = 42.0
	marginR  = 42.0
	contentW = pageW - marginL - marginR
	topY     = 56.0
	bottomY  = 788.0
	footerY  = 808.0
)

type pdfDoc struct {
	pdf         *gopdf.GoPdf
	y           float64
	page        int
	reportID    string
	hasFallback bool // 是否成功加载了系统兜底字体
}

// ───────────────────────── 报告维度与判定 ─────────────────────────

type reportDim struct {
	title string
	ids   []string
}

var reportDims = []reportDim{
	{"身份与来源验真", []string{probeFamilyDirect, probeFamilyCorrection, probeClaudeYesNo, probeModelNameConfirm, probeModelNameEN, probeModelNameZH, probeAgentProxy, probeIdentityConsistency}},
	{"推理与能力基准", []string{probeStructuredJSON, probeThreeLine, probeReasoningBench, probeToolCall}},
	{"知识真实性与时效", []string{probeKnowledgeCutoff, probeFactElection}},
	{"安全与合规", []string{probeRefuseForgery, probeInstructionOverride, probePromptExtraction}},
	{"响应完整性与注入", []string{probeTokenInjection}},
	{"多语言能力", []string{probeMultilingual}},
}

type modelStats struct {
	pass, warn, fail, total int
	httpFail                int
	passByID                map[string]bool
}

func computeStats(outs []probeOutcome) modelStats {
	s := modelStats{passByID: map[string]bool{}}
	for _, o := range outs {
		s.total++
		switch {
		case o.Success && o.Pass:
			s.pass++
			s.passByID[o.ProbeID] = true
		case o.Success && !o.Pass:
			s.warn++
		default:
			s.fail++
			s.httpFail++
		}
	}
	return s
}

func (s modelStats) ratio() float64 {
	if s.total == 0 {
		return 0
	}
	return float64(s.pass) / float64(s.total)
}

// computeTier 给出综合判定：标签、颜色、客观结论。
func computeTier(s modelStats) (string, rgb, string) {
	ratio := s.ratio()
	idOK := s.passByID[probeFamilyDirect] && s.passByID[probeClaudeYesNo] && s.passByID[probeModelNameConfirm]
	knOK := s.passByID[probeKnowledgeCutoff] && s.passByID[probeFactElection]
	switch {
	case s.httpFail > 0 && s.pass == 0:
		return "无法判定", cRed, "请求大多失败，未能取得有效响应，无法对该通道作出可信度判定。"
	case s.pass == s.total:
		return "高度可信", cGreen, "全部检测项通过：模型身份、推理能力与知识时效均与所声称型号高度一致，未见套壳、注入或降级迹象。"
	case ratio >= 0.85 && idOK && knOK:
		return "基本可信", cGreen, "绝大多数检测项通过，核心身份验真与知识时效无异常；少量未达标项多属语义边界或代理通道的预期表现。"
	case ratio >= 0.6:
		return "存在风险", cAmber, "部分检测项未达标，建议结合下方明细审慎评估该通道与所声称模型的一致性。"
	default:
		return "高风险", cRed, "多项关键检测未通过，存在套壳、隐藏指令注入或能力/知识不符的显著风险，不建议直接用于生产。"
	}
}

func groupByModel(outs []probeOutcome) ([]string, map[string][]probeOutcome) {
	order := []string{}
	m := map[string][]probeOutcome{}
	for _, o := range outs {
		if _, ok := m[o.Model]; !ok {
			order = append(order, o.Model)
		}
		m[o.Model] = append(m[o.Model], o)
	}
	return order, m
}

func outcomeVerdict(o probeOutcome) (string, rgb) {
	switch {
	case o.Success && o.Pass:
		return "PASS", cGreen
	case o.Success && !o.Pass:
		return "WARN", cAmber
	default:
		return "FAIL", cRed
	}
}

func profileLabel(p string) string {
	switch p {
	case profileOAuthProxy:
		return "oauth-proxy（H/I/J 的 WARN 视为预期，不计入失败）"
	default:
		return "strict（任一 WARN 或请求失败即判为未通过）"
	}
}

// ───────────────────────── 入口 ─────────────────────────

func generatePDFReport(path, fontPath, base, profile string, outcomes []probeOutcome, generatedAt time.Time) error {
	if len(outcomes) == 0 {
		return fmt.Errorf("没有可用于生成报告的探测结果")
	}
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if err := pdf.AddTTFFontData(fFang, fangSongTTF); err != nil {
		return fmt.Errorf("加载内嵌仿宋字体失败: %w", err)
	}
	if err := pdf.AddTTFFontData(fTimes, timesTTF); err != nil {
		return fmt.Errorf("加载内嵌 Times 字体失败: %w", err)
	}
	// 系统兜底字体可选：仅用于仿宋/Times 都不含的字形（如韩文谚文）。缺失时优雅降级。
	hasFallback := false
	fb, err := resolveFallbackFont(fontPath)
	if err != nil {
		return err
	}
	if fb != "" {
		if err := pdf.AddTTFFont(fFallback, fb); err != nil {
			return fmt.Errorf("加载兜底字体失败 %s: %w（需为 .ttf，.ttc/.otf 不受支持）", fb, err)
		}
		hasFallback = true
	}

	order, byModel := groupByModel(outcomes)
	d := &pdfDoc{pdf: pdf, hasFallback: hasFallback, reportID: reportID(base, order, generatedAt)}

	pdf.AddPage()
	d.page = 1
	d.footer()
	d.y = topY
	d.drawTitleBlock(base, profile, order, generatedAt, outcomes)

	for i, m := range order {
		if i > 0 {
			d.newPage()
		} else {
			d.ensure(60)
		}
		d.drawModelSection(m, byModel[m])
	}

	d.drawDisclaimer()

	if err := pdf.WritePdf(path); err != nil {
		return fmt.Errorf("写入 PDF 失败: %w", err)
	}
	return nil
}

func reportID(base string, models []string, ts time.Time) string {
	h := sha256.Sum256([]byte(base + "|" + strings.Join(models, ",") + "|" + ts.Format(time.RFC3339)))
	return "MVR-" + strings.ToUpper(hex.EncodeToString(h[:])[:8])
}

// ───────────────────────── 版面绘制 ─────────────────────────

func (d *pdfDoc) drawTitleBlock(base, profile string, models []string, ts time.Time, outcomes []probeOutcome) {
	d.rect(0, 0, pageW, 96, cNavy)
	d.at(marginL, 30, 21, cWhite, "模型质量与真实性验真报告")
	d.at(marginL, 60, 10, cSlate, "Model Quality & Authenticity Verification Report")
	d.y = 120

	overall := computeStats(outcomes)
	d.metaRow("目标网关", base)
	d.metaRow("受检模型", strings.Join(models, "，"))
	d.metaRow("检测方法", "modelknow 身份验真（A–K）+ api-check 能力/知识/安全（L–S），每模型 19 项探测")
	d.metaRow("判定模式", profileLabel(profile))
	d.metaRow("汇总结果", fmt.Sprintf("PASS %d · WARN %d · FAIL %d（共 %d 项 / %d 模型）", overall.pass, overall.warn, overall.fail, overall.total, len(models)))
	d.metaRow("生成时间", ts.Format("2006-01-02 15:04:05 -0700"))
	d.metaRow("报告编号", d.reportID)

	d.y += 6
	d.hrule(d.y, cRule)
	d.y += 12
	d.paragraph(marginL, contentW, 8.8, 12.5, cMuted,
		"方法说明：报告对每个目标模型依次发送身份、推理、知识、安全与多语言等探测请求，逐项比对模型真实回复与预期判据。"+
			"PASS 表示符合预期，WARN 表示 HTTP 成功但语义未达标，FAIL 表示请求或传输失败。下文如实记录检测时点的原始反馈。")
	d.y += 8
}

func (d *pdfDoc) drawModelSection(model string, outs []probeOutcome) {
	s := computeStats(outs)
	tierLabel, tierColor, tierNote := computeTier(s)

	d.ensure(40)
	d.at(marginL, d.y, 14, cInk, "受检模型："+model)
	d.y += 20
	d.hrule(d.y, cRule)
	d.y += 14

	d.ensure(22)
	w := d.chip(marginL, d.y-2, "综合判定 "+tierLabel, tierColor, cWhite)
	d.at(marginL+w+10, d.y, 10, cMuted,
		fmt.Sprintf("通过 %d / %d（%.0f%%）· WARN %d · FAIL %d", s.pass, s.total, s.ratio()*100, s.warn, s.fail))
	d.y += 20
	d.paragraph(marginL, contentW, 9.3, 13.5, cInk, tierNote)
	d.y += 10

	d.drawDimensionTable(outs)
	d.y += 10

	d.ensure(24)
	d.at(marginL, d.y, 11.5, cInk, "检测明细")
	d.y += 6
	d.hrule(d.y+2, cRule)
	d.y += 12
	for _, o := range outs {
		d.drawProbeDetail(o)
	}
}

func (d *pdfDoc) drawDimensionTable(outs []probeOutcome) {
	byID := map[string]probeOutcome{}
	for _, o := range outs {
		byID[o.ProbeID] = o
	}
	d.ensure(22)
	d.at(marginL, d.y, 11.5, cInk, "能力维度概览")
	d.y += 18
	for _, dim := range reportDims {
		var tot, pas int
		for _, id := range dim.ids {
			if o, ok := byID[id]; ok {
				tot++
				if o.Success && o.Pass {
					pas++
				}
			}
		}
		if tot == 0 {
			continue
		}
		d.ensure(18)
		d.at(marginL, d.y, 9.5, cInk, dim.title)
		barX := marginL + 150
		barW := 230.0
		barH := 7.0
		barY := d.y + 1.5
		d.rect(barX, barY, barW, barH, cChip)
		ratio := float64(pas) / float64(tot)
		col := cGreen
		if ratio < 0.6 {
			col = cRed
		} else if ratio < 1 {
			col = cAmber
		}
		if ratio > 0 {
			d.rect(barX, barY, barW*ratio, barH, col)
		}
		d.rightAt(barX+barW+8, d.y, (pageW-marginR)-(barX+barW+8), 9, cMuted, fmt.Sprintf("%d/%d", pas, tot))
		d.y += 16.5
	}
}

func (d *pdfDoc) drawProbeDetail(o probeOutcome) {
	mark, col := outcomeVerdict(o)

	d.ensure(30)
	d.chip(marginL, d.y, mark, col, cWhite)
	d.at(marginL+44, d.y+1, 10, cInk, probeTitle(o.ProbeID))
	d.y += 18

	if o.Expected != "" {
		d.labelPara("期望", o.Expected, cMuted)
	}
	reason := o.Reason
	if reason == "" && o.Error != "" {
		reason = o.Error
	}
	if reason != "" {
		d.labelPara("实测", reason, col)
	}
	if len(o.SubItems) > 0 {
		d.drawSubItems(o.SubItems)
	} else if o.Snippet != "" {
		reply := strings.ReplaceAll(o.Snippet, "\n", " / ")
		d.labelPara("模型回复", truncate(reply, 600), cInk)
	}
	d.ensure(13)
	d.at(marginL+58, d.y, 8.3, cMuted, fmt.Sprintf("HTTP %d · 延迟 %d ms · max_tokens %d", o.HTTPStatus, o.DurationMs, o.MaxTokens))
	d.y += 13
	d.hrule(d.y, cRule)
	d.y += 9
}

// subItemMark 把子题标记映射为报告中的短标签与配色。
func subItemMark(mark string) (string, rgb) {
	switch mark {
	case subPass:
		return "OK", cGreen
	case subWarn:
		return "WARN", cAmber
	case subFail:
		return "X", cRed
	default:
		return "·", cMuted
	}
}

// drawSubItems 把「单项探测的多个子题」逐行排版：每题一枚状态标签 + 标签名，回复另起一行缩进显示，
// 避免多个问答在「模型回复」里挤成一团。
func (d *pdfDoc) drawSubItems(items []probeSubItem) {
	d.ensure(13)
	d.at(marginL+8, d.y, 8.3, cMuted, "模型回复")
	d.y += 14

	itemX := marginL + 58           // 与上方期望/实测的取值列对齐
	replyX := itemX + 12            // 回复相对子题标签再缩进一档
	replyW := contentW - (replyX - marginL)
	for _, it := range items {
		label, col := subItemMark(it.Mark)
		d.ensure(14)
		cw := d.chip(itemX, d.y-1, label, col, cWhite)
		d.at(itemX+cw+6, d.y+1, 9, cInk, it.Label)
		d.y += 14
		for _, ln := range d.wrap(it.Reply, replyW, 8.7) {
			d.ensure(11.5)
			d.at(replyX, d.y, 8.7, cMuted, ln)
			d.y += 11.5
		}
		d.y += 4
	}
}

func (d *pdfDoc) drawDisclaimer() {
	d.ensure(70)
	d.y += 4
	d.hrule(d.y, cRule)
	d.y += 12
	d.at(marginL, d.y, 9.5, cInk, "说明与免责")
	d.y += 15
	notes := []string{
		"本报告由 model-verify-check 自动化探测生成，如实记录检测时点目标网关对各探测请求的真实响应，未作人工修饰或删改。",
		"判定结论基于公开的模型行为基准与启发式规则，反映被检通道与所声称模型的一致性，不构成任何官方认证或法律意见。",
		"模型回复可能随时间、温度与上游路由策略波动；对外引用时建议结合多次检测与具体业务场景综合评估。",
	}
	for i, n := range notes {
		d.paragraph(marginL, contentW, 8.5, 12, cMuted, fmt.Sprintf("%d. %s", i+1, n))
		d.y += 1
	}
}

// ───────────────────────── 低层绘制原语 ─────────────────────────

func (d *pdfDoc) setColor(c rgb) { d.pdf.SetTextColor(c.r, c.g, c.b) }
func (d *pdfDoc) fill(c rgb)      { d.pdf.SetFillColor(c.r, c.g, c.b) }

// fontForRune 按 Unicode 区段把字符分派到对应字体族。
func fontForRune(r rune) string {
	switch {
	case r <= 0x024F: // 基本拉丁 + 拉丁补充 + 拉丁扩展A（含 °×÷ 等）
		return fTimes
	case r >= 0x2000 && r <= 0x206F: // 常用标点（– — “ ” … • 等，Tinos 覆盖）
		return fTimes
	case (r >= 0x2E80 && r <= 0x30FF) || // CJK 部首 + 假名 + CJK 标点
		(r >= 0x3400 && r <= 0x9FFF) || // CJK 扩展A + 基本汉字
		(r >= 0xF900 && r <= 0xFAFF) || // CJK 兼容汉字
		(r >= 0xFF00 && r <= 0xFFEF): // 全角/半角形式（全角逗号、括号等）
		return fFang
	default: // 其余（如韩文谚文）走系统兜底
		return fFallback
	}
}

// resolveFont 返回该字符实际可用的字体族；当目标为 fallback 而未加载兜底字体时返回 false（跳过该字符）。
func (d *pdfDoc) resolveFont(r rune) (string, bool) {
	f := fontForRune(r)
	if f == fFallback && !d.hasFallback {
		return "", false
	}
	return f, true
}

type textRun struct{ font, s string }

// splitRuns 把一段文本按字体族切成连续片段，无法渲染的字符被丢弃。
func (d *pdfDoc) splitRuns(s string) []textRun {
	var runs []textRun
	var b strings.Builder
	cur := ""
	flush := func() {
		if b.Len() > 0 {
			runs = append(runs, textRun{cur, b.String()})
			b.Reset()
		}
	}
	for _, r := range s {
		f, ok := d.resolveFont(r)
		if !ok {
			continue
		}
		if cur == "" {
			cur = f
		}
		if f != cur {
			flush()
			cur = f
		}
		b.WriteRune(r)
	}
	flush()
	return runs
}

// measure 按各片段对应字体逐段累加文本像素宽度。
func (d *pdfDoc) measure(s string, size float64) float64 {
	w := 0.0
	for _, run := range d.splitRuns(s) {
		_ = d.pdf.SetFont(run.font, "", size)
		rw, _ := d.pdf.MeasureTextWidth(run.s)
		w += rw
	}
	return w
}

func (d *pdfDoc) runeWidth(r rune, size float64) float64 {
	f, ok := d.resolveFont(r)
	if !ok {
		return 0
	}
	_ = d.pdf.SetFont(f, "", size)
	rw, _ := d.pdf.MeasureTextWidth(string(r))
	return rw
}

func (d *pdfDoc) at(x, y, size float64, c rgb, s string) {
	d.setColor(c)
	cx := x
	for _, run := range d.splitRuns(sanitizeForPDF(s)) {
		_ = d.pdf.SetFont(run.font, "", size)
		d.pdf.SetXY(cx, y)
		_ = d.pdf.Cell(nil, run.s)
		rw, _ := d.pdf.MeasureTextWidth(run.s)
		cx += rw
	}
}

func (d *pdfDoc) rightAt(x, y, w, size float64, c rgb, s string) {
	s = sanitizeForPDF(s)
	tw := d.measure(s, size)
	d.at(x+w-tw, y, size, c, s)
}

func (d *pdfDoc) hrule(y float64, c rgb) {
	d.pdf.SetStrokeColor(c.r, c.g, c.b)
	d.pdf.SetLineWidth(0.6)
	d.pdf.Line(marginL, y, pageW-marginR, y)
}

func (d *pdfDoc) rect(x, y, w, h float64, c rgb) {
	d.fill(c)
	d.pdf.RectFromUpperLeftWithStyle(x, y, w, h, "F")
}

// chip 绘制一枚带背景的小标签，返回其宽度。
func (d *pdfDoc) chip(x, y float64, label string, bg, fg rgb) float64 {
	label = sanitizeForPDF(label)
	tw := d.measure(label, 8)
	w := tw + 12
	h := 13.0
	d.rect(x, y, w, h, bg)
	d.at(x+6, y+2.6, 8, fg, label)
	return w
}

func (d *pdfDoc) metaRow(label, value string) {
	valueX := marginL + 82
	valueW := contentW - 82
	lines := d.wrap(value, valueW, 9.5)
	d.ensure(float64(len(lines)) * 14)
	d.at(marginL, d.y, 9.5, cMuted, label)
	for _, ln := range lines {
		d.at(valueX, d.y, 9.5, cInk, ln)
		d.y += 14
	}
	d.y += 2
}

func (d *pdfDoc) labelPara(label, value string, c rgb) {
	valX := marginL + 58
	valW := contentW - 58
	lines := d.wrap(value, valW, 9)
	lh := 13.0
	d.ensure(lh)
	d.at(marginL+8, d.y, 8.3, cMuted, label)
	for _, ln := range lines {
		d.ensure(lh)
		d.at(valX, d.y, 9, c, ln)
		d.y += lh
	}
	d.y += 2
}

func (d *pdfDoc) paragraph(x, w, size, lineH float64, c rgb, s string) {
	for _, ln := range d.wrap(s, w, size) {
		d.ensure(lineH)
		d.at(x, d.y, size, c, ln)
		d.y += lineH
	}
}

// wrap 按像素宽度换行，兼容无空格的中日韩文本。
func (d *pdfDoc) wrap(text string, maxW, size float64) []string {
	text = sanitizeForPDF(text)
	var out []string
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r", ""), "\n") {
		out = append(out, d.wrapParagraph(para, maxW, size)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func (d *pdfDoc) wrapParagraph(s string, maxW, size float64) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	cur := make([]rune, 0, 64)
	curW := 0.0
	lastSpace := -1
	for _, r := range s {
		rw := d.runeWidth(r, size)
		if curW+rw > maxW && len(cur) > 0 {
			if lastSpace > 0 && lastSpace < len(cur) {
				head := strings.TrimRight(string(cur[:lastSpace]), " ")
				tail := append([]rune{}, cur[lastSpace:]...)
				lines = append(lines, head)
				cur = tail
				curW = d.measure(string(cur), size)
				lastSpace = -1
			} else {
				lines = append(lines, string(cur))
				cur = cur[:0]
				curW = 0
				lastSpace = -1
			}
		}
		if r == ' ' {
			lastSpace = len(cur)
		}
		cur = append(cur, r)
		curW += rw
	}
	if len(cur) > 0 {
		lines = append(lines, string(cur))
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

func (d *pdfDoc) ensure(h float64) {
	if d.y+h > bottomY {
		d.newPage()
	}
}

func (d *pdfDoc) newPage() {
	d.pdf.AddPage()
	d.page++
	d.footer()
	d.y = topY
}

func (d *pdfDoc) footer() {
	d.hrule(footerY-9, cRule)
	d.at(marginL, footerY, 7.5, cMuted, "model-verify-check · 模型验真自动化报告（modelknow A–K + api-check L–S）")
	d.rightAt(pageW-marginR-220, footerY, 220, 7.5, cMuted, fmt.Sprintf("报告编号 %s · 第 %d 页", d.reportID, d.page))
}

// ───────────────────────── 字体与文本清洗 ─────────────────────────

var reportFontCandidates = []string{
	// macOS
	"/Library/Fonts/Arial Unicode.ttf",
	"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
	// Linux（仅列出单文件 .ttf；Noto/文泉驿的 .ttc/.otf 不受 gopdf 支持）
	"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttf",
	"/usr/share/fonts/truetype/noto/NotoSansSC-Regular.ttf",
	"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttf",
	"/usr/share/fonts/truetype/wqy/wqy-microhei.ttf",
	// Windows
	`C:\Windows\Fonts\msyh.ttf`,
	`C:\Windows\Fonts\simhei.ttf`,
	`C:\Windows\Fonts\arialuni.ttf`,
}

// resolveFallbackFont 解析可选的系统兜底字体路径，仅用于内嵌仿宋/Times 都缺失的字形（如韩文谚文）。
// 返回空字符串表示无兜底字体（仍可正常出报告，仅个别字形被跳过）；仅当显式 -font 指定却找不到时报错。
func resolveFallbackFont(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		if fileExists(explicit) {
			return explicit, nil
		}
		return "", fmt.Errorf("指定的 -font 不存在: %s", explicit)
	}
	if env := strings.TrimSpace(os.Getenv("MVR_FONT")); env != "" && fileExists(env) {
		return env, nil
	}
	for _, c := range reportFontCandidates {
		if fileExists(c) {
			return c, nil
		}
	}
	return "", nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// emojiReplacer 把检测结论里的表情/符号替换为字体可渲染的纯文本标记。
var emojiReplacer = strings.NewReplacer(
	"✅", "[OK] ", "❌", "[X] ", "⚠️", "[!] ", "⚠", "[!] ", "🚨", "[!!] ",
	"🚩", "[!] ", "ℹ️", "", "ℹ", "", "⚡", "", "→", "->", "⇒", "=>",
	"🌐", "", "📝", "", "🪪", "", "📅", "", "📊", "", "📋", "", "🔍", "", "🔎", "",
	"🔢", "", "🗳️", "", "🗳", "", "🧠", "", "🧮", "", "🌉", "", "🐱", "", "📡", "",
)

// sanitizeForPDF 去除字体无法渲染的字符，保证 WritePdf 不因缺字形而丢内容。
func sanitizeForPDF(s string) string {
	s = emojiReplacer.Replace(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0xFE0F || r == 0x200D: // 变体选择符 / 零宽连接符
			continue
		case r >= 0x1F000: // 星形面表情/象形文字
			continue
		case r >= 0x2600 && r <= 0x27BF: // 杂项符号与装饰符号
			continue
		case r >= 0x2190 && r <= 0x21FF: // 箭头（常用的 → ⇒ 已在上面替换）
			continue
		case r == 0xFFFD: // 替换字符
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

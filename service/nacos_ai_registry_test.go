package service_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	gleSQLite "github.com/glebarez/sqlite"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/env"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/service"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func setupNacosAITestDB(t *testing.T) {
	t.Helper()
	v := viper.New()
	v.SetDefault("sql_dsn", "")
	env.BindViper(v)
	config.LoadRuntime()
	// modernc.org/sqlite（经 github.com/glebarez/sqlite）纯 Go，无需 CGO；与 mattn/go-sqlite3 不同。
	db, err := gorm.Open(gleSQLite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Skip("无法打开 SQLite:", err)
	}
	if err := db.Exec("SELECT 1").Error; err != nil {
		t.Skip("SQLite 不可用:", err)
	}
	model.DB = db
	common.UsingSQLite = true
	if err := db.AutoMigrate(&model.NacosAIArtifact{}, &model.NacosAIArtifactVersion{}); err != nil {
		t.Fatal(err)
	}
}

func zipSkill(name, md string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	p := name + "/SKILL.md"
	f, _ := w.Create(p)
	_, _ = f.Write([]byte(md))
	_ = w.Close()
	return buf.Bytes()
}

func zipSkillWithPkg(name, md, packageJSON string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	p := name + "/SKILL.md"
	f, _ := w.Create(p)
	_, _ = f.Write([]byte(md))
	pj, _ := w.Create(name + "/package.json")
	_, _ = pj.Write([]byte(packageJSON))
	_ = w.Close()
	return buf.Bytes()
}

func zipSkillWithMeta(name, md, metaJSON string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	p := name + "/SKILL.md"
	f, _ := w.Create(p)
	_, _ = f.Write([]byte(md))
	m, _ := w.Create(name + "/_meta.json")
	_, _ = m.Write([]byte(metaJSON))
	_ = w.Close()
	return buf.Bytes()
}

func TestNacosAIUploadSkillUsesMetaJSONVersion(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkillWithMeta("metaskill", "# M\n", `{"version":"4.5.6"}`)
	if err := service.NacosAIUploadSkill(ns, data, 0, ""); err != nil {
		t.Fatal(err)
	}
	d, err := service.NacosAIDescribeSkill(ns, "metaskill")
	if err != nil {
		t.Fatal(err)
	}
	ok := false
	for _, v := range d.Versions {
		if v.Version == "4.5.6" && v.Status == "draft" {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("expected version 4.5.6, got %#v", d.Versions)
	}
}

func TestNacosAICreateDraftFromVersionUsesTargetAndPatchesMeta(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkillWithMeta("fork", "# Hi\n", `{"version":"1.0.0"}`)
	if err := service.NacosAIUploadSkill(ns, data, 0, ""); err != nil {
		t.Fatal(err)
	}
	d, _ := service.NacosAIDescribeSkill(ns, "fork")
	if len(d.Versions) < 1 {
		t.Fatal("no versions")
	}
	base := d.Versions[0].Version
	nv, err := service.NacosAIArtifactCreateDraftFromVersion(ns, model.NacosAIKindSkill, "fork", base, "1.0.1", "init draft")
	if err != nil {
		t.Fatal(err)
	}
	if nv != "1.0.1" {
		t.Fatalf("expected new version 1.0.1, got %q", nv)
	}
	doc, err := service.NacosAIDocumentSkillVersion(ns, "fork", "1.0.1")
	if err != nil {
		t.Fatal(err)
	}
	var metaContent string
	for _, r := range doc.Resource {
		if strings.HasSuffix(r.Name, "_meta.json") || r.Type == "_meta.json" {
			metaContent = r.Content
			break
		}
	}
	if metaContent == "" {
		for rel, r := range doc.Resource {
			if strings.Contains(rel, "_meta.json") {
				metaContent = r.Content
				break
			}
		}
	}
	if !strings.Contains(metaContent, "1.0.1") {
		t.Fatalf("_meta.json should contain new version, resource keys=%v meta=%q", doc.Resource, metaContent)
	}
}

// 根目录先有杂文件（模拟常见错误打包顺序），再写 skill 目录。
func zipSkillWithRootNoise(name, md string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	n0, _ := w.Create("package-lock.json")
	_, _ = n0.Write([]byte("{}"))
	p := name + "/SKILL.md"
	f, _ := w.Create(p)
	_, _ = f.Write([]byte(md))
	_ = w.Close()
	return buf.Bytes()
}

func zipSkillFlat(md string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f0, _ := w.Create("SKILL.md")
	_, _ = f0.Write([]byte(md))
	j, _ := w.Create("package.json")
	_, _ = j.Write([]byte("{}"))
	s, _ := w.Create("src/x.js")
	_, _ = s.Write([]byte("//"))
	_ = w.Close()
	return buf.Bytes()
}

func zipSkillAmbiguousTwoRoots() []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	f1, _ := w.Create("SKILL.md")
	_, _ = f1.Write([]byte("# root"))
	f2, _ := w.Create("other/SKILL.md")
	_, _ = f2.Write([]byte("# other"))
	_ = w.Close()
	return buf.Bytes()
}

func TestNacosAIUploadSkillIgnoresLooseRootFiles(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkillWithRootNoise("noisy", "# Noisy\n\nok")
	if err := service.NacosAIUploadSkill(ns, data, 0, ""); err != nil {
		t.Fatal(err)
	}
}

func TestNacosAISkillLifecycle(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkill("demo", "# Demo\n\nHello skill")
	if err := service.NacosAIUploadSkill(ns, data, 0, ""); err != nil {
		t.Fatal(err)
	}
	if err := service.NacosAISubmit(ns, model.NacosAIKindSkill, "demo", ""); err != nil {
		t.Fatal(err)
	}
	ver := ""
	d, _ := service.NacosAIDescribeSkill(ns, "demo")
	for _, v := range d.Versions {
		if v.Status == model.NacosAIVersionReviewing {
			ver = v.Version
			break
		}
	}
	if ver == "" {
		t.Fatal("no reviewing version")
	}
	if err := service.NacosAIPublish(ns, model.NacosAIKindSkill, "demo", ver, true, false); err != nil {
		t.Fatal(err)
	}
	out, err := service.NacosAIGetSkillZIP(ns, "demo", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 10 {
		t.Fatalf("zip too small: %d", len(out))
	}
}

func TestNacosAIUploadSkillUsesPackageJSONVersion(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkillWithPkg("pkgdemo", "# P\n\nx", `{"name":"pkgdemo","version":"1.0.2"}`)
	if err := service.NacosAIUploadSkill(ns, data, 0, ""); err != nil {
		t.Fatal(err)
	}
	d, err := service.NacosAIDescribeSkill(ns, "pkgdemo")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range d.Versions {
		// Describe 接口将 DB 的 editing 映射为控制台用的 draft
		if v.Version == "1.0.2" && v.Status == "draft" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected version 1.0.2 in versions: %#v", d.Versions)
	}
}

func TestNacosAIUploadSkillFrontmatterVersionOverridesPackage(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	md := "---\nversion: 3.1.0\n---\n\n# Hi\n"
	data := zipSkillWithPkg("fm", md, `{"version":"1.0.0"}`)
	if err := service.NacosAIUploadSkill(ns, data, 0, ""); err != nil {
		t.Fatal(err)
	}
	d, _ := service.NacosAIDescribeSkill(ns, "fm")
	for _, v := range d.Versions {
		if v.Status == "draft" && v.Version == "3.1.0" {
			return
		}
	}
	t.Fatalf("expected frontmatter version 3.1.0, got %#v", d.Versions)
}

func TestNacosAIUploadSkillDuplicateVersionGetsSuffix(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	pkg := `{"version":"9.9.9"}`
	if err := service.NacosAIUploadSkill(ns, zipSkillWithPkg("dup", "# A\n", pkg), 0, ""); err != nil {
		t.Fatal(err)
	}
	if err := service.NacosAIUploadSkill(ns, zipSkillWithPkg("dup", "# B\n", pkg), 0, ""); err != nil {
		t.Fatal(err)
	}
	d, _ := service.NacosAIDescribeSkill(ns, "dup")
	if len(d.Versions) < 2 {
		t.Fatalf("expected 2 versions, got %d", len(d.Versions))
	}
	seen := map[string]bool{}
	for _, v := range d.Versions {
		seen[v.Version] = true
	}
	if !seen["9.9.9"] {
		t.Fatalf("missing 9.9.9: %#v", d.Versions)
	}
	suffixed := false
	for v := range seen {
		if strings.HasPrefix(v, "9.9.9-") {
			suffixed = true
			break
		}
	}
	if !suffixed {
		t.Fatalf("expected suffixed second version like 9.9.9-<millis>, got %v", seen)
	}
}

func zipAgentSpec(name, manifestJSON string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	p := name + "/manifest.json"
	f, _ := w.Create(p)
	_, _ = f.Write([]byte(manifestJSON))
	_ = w.Close()
	return buf.Bytes()
}

func TestNacosAIUploadAgentSpecUsesManifestVersion(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	man := `{"version":"2.4.1","description":"test agent spec"}`
	data := zipAgentSpec("aspecver", man)
	if err := service.NacosAIUploadAgentSpec(ns, data, 0, true, ""); err != nil {
		t.Fatal(err)
	}
	d, err := service.NacosAIDescribeAgentSpec(ns, "aspecver")
	if err != nil {
		t.Fatal(err)
	}
	ok := false
	for _, v := range d.Versions {
		if v.Version == "2.4.1" && v.Status == "draft" {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("expected version 2.4.1, versions=%#v", d.Versions)
	}
}

func TestNacosAIUploadSkillFlatRootZIP(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkillFlat("# Weibo\n\nskill")
	if err := service.NacosAIUploadSkill(ns, data, 0, "weibo-manager-1.0.2.zip"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.NacosAIDescribeSkill(ns, "weibo-manager-1.0.2"); err != nil {
		t.Fatalf("describe: %v", err)
	}
}

func TestNacosAIUploadSkillFlatRootZIPRequiresFilename(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkillFlat("# x\n")
	if err := service.NacosAIUploadSkill(ns, data, 0, ""); err == nil {
		t.Fatal("expected error when root-flat zip without upload filename")
	}
}

func TestNacosAIUploadSkillRejectsAmbiguousRoots(t *testing.T) {
	setupNacosAITestDB(t)
	ns := "public"
	data := zipSkillAmbiguousTwoRoots()
	if err := service.NacosAIUploadSkill(ns, data, 0, "x.zip"); err == nil {
		t.Fatal("expected error when ZIP has both root SKILL.md and other/SKILL.md")
	}
}

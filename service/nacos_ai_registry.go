package service

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/songquanpeng/one-api/model"
	"gorm.io/gorm"
)

const nacosCtxUserKey = "nacosRegistryUser"

// mapNacosAIVersionStatusForConsoleUI 将 DB 中的版本状态映射为 Nacos console-ui-next 使用的 SkillVersionStatus。
// 本地库使用 editing；新版控制台与官方契约使用 draft，否则生命周期按钮（下线、基于本版建草稿等）不会渲染。
func mapNacosAIVersionStatusForConsoleUI(db string) string {
	switch strings.TrimSpace(db) {
	case model.NacosAIVersionEditing:
		return "draft"
	case model.NacosAIVersionReviewing:
		return "reviewing"
	case model.NacosAIVersionOnline:
		return "online"
	case model.NacosAIVersionOffline:
		return "offline"
	default:
		return strings.TrimSpace(db)
	}
}

// NacosCtxUserKey Gin Context 中存放 *model.User（可选，匿名读时为空）。
func NacosCtxUserKey() string { return nacosCtxUserKey }

// --- ZIP ---

// skillArtifactNameFromZipFilename 从上传文件名推断技能名（去掉路径与 .zip 后缀）。
func skillArtifactNameFromZipFilename(filename string) string {
	s := strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	if s == "" {
		return ""
	}
	s = path.Base(s)
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, ".zip") {
		s = s[:len(s)-4]
	}
	return strings.TrimSpace(s)
}

func parseZipArchive(zipData []byte) (root string, files map[string][]byte, err error) {
	files = make(map[string][]byte)
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", nil, err
	}
	type zfile struct {
		z    *zip.File
		name string
	}
	var list []zfile
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		if strings.Contains(name, "..") {
			return "", nil, fmt.Errorf("非法路径: %s", f.Name)
		}
		if strings.HasSuffix(name, "/") {
			continue
		}
		list = append(list, zfile{z: f, name: name})
	}
	// 由「一级目录/SKILL.md」或「一级目录/manifest.json」推断唯一根目录；也支持压缩包根目录的 SKILL.md / manifest.json（root 记为 ""）。
	roots := map[string]struct{}{}
	for _, e := range list {
		dir := path.Dir(e.name)
		base := strings.ToLower(path.Base(e.name))
		if base != "skill.md" && base != "manifest.json" {
			continue
		}
		if dir == "." {
			roots[""] = struct{}{}
			continue
		}
		if strings.Contains(dir, "/") {
			continue
		}
		roots[dir] = struct{}{}
	}
	switch len(roots) {
	case 0:
		return "", nil, errors.New("ZIP 内未找到 SKILL.md 或 manifest.json（可为 <目录>/SKILL.md，或压缩包根目录 SKILL.md；根目录平铺时请使用「技能名.zip」上传以便命名）")
	case 1:
		for r := range roots {
			root = r
		}
	default:
		names := make([]string, 0, len(roots))
		for r := range roots {
			names = append(names, r)
		}
		sort.Strings(names)
		return "", nil, fmt.Errorf("ZIP 内含多个资源根目录（均含 SKILL.md 或 manifest.json）: %v", names)
	}
	for _, e := range list {
		if root == "" {
			if strings.HasPrefix(e.name, "__MACOSX/") {
				continue
			}
		} else {
			prefix := root + "/"
			if e.name != root && !strings.HasPrefix(e.name, prefix) {
				continue
			}
		}
		rc, err := e.z.Open()
		if err != nil {
			return "", nil, err
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return "", nil, err
		}
		files[e.name] = b
	}
	if len(files) == 0 {
		return "", nil, errors.New("ZIP 为空或根目录下无有效文件")
	}
	return root, files, nil
}

var nacosSkillVersionTokenRe = regexp.MustCompile(`^[A-Za-z0-9._~+\-]{1,64}$`)

// skillVersionFromZip 从包内推断 Skill 版本号（供上传时作为版本字符串）。
// 优先级：_meta.json > SKILL.md YAML frontmatter > package.json。
func skillVersionFromZip(root string, files map[string][]byte) string {
	metaPath := path.Join(root, "_meta.json")
	if b, ok := files[metaPath]; ok {
		if v := sanitizeNacosArtifactVersion(artifactVersionFromTopLevelJSON(b)); v != "" {
			return v
		}
	}
	mdPath := path.Join(root, "SKILL.md")
	if b, ok := files[mdPath]; ok {
		if v := skillVersionFromSkillMdFrontmatter(string(b)); v != "" {
			return sanitizeNacosArtifactVersion(v)
		}
	}
	pjPath := path.Join(root, "package.json")
	if b, ok := files[pjPath]; ok {
		if v := skillVersionFromPackageJSON(b); v != "" {
			return sanitizeNacosArtifactVersion(v)
		}
	}
	return ""
}

func skillVersionFromSkillMdFrontmatter(md string) string {
	md = strings.TrimSpace(md)
	if md == "" || !strings.HasPrefix(md, "---") {
		return ""
	}
	rest := strings.TrimPrefix(md, "---")
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	block := rest[:end]
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		low := strings.ToLower(line)
		if !strings.HasPrefix(low, "version:") {
			continue
		}
		// 支持 version: 1.0.0 / version: "1.0.0"
		val := strings.TrimSpace(line[strings.Index(line, ":")+1:])
		val = strings.Trim(val, `"'`)
		return strings.TrimSpace(val)
	}
	return ""
}

// artifactVersionFromTopLevelJSON 读取 JSON 文档顶层 version（package.json、manifest.json 等）。
func artifactVersionFromTopLevelJSON(raw []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	v, ok := m["version"]
	if !ok {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return ""
	}
}

func skillVersionFromPackageJSON(raw []byte) string {
	return artifactVersionFromTopLevelJSON(raw)
}

// sanitizeNacosArtifactVersion 将版本号限制为控制台与 DB 可接受的短字符串（varchar 64）；不合法则返回空以便回退时间戳版本。
func sanitizeNacosArtifactVersion(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 64 {
		return ""
	}
	if !nacosSkillVersionTokenRe.MatchString(s) {
		return ""
	}
	return s
}

// allocateArtifactVersion 在 artifact 下生成唯一 version 字符串；preferred 为空则用 v{毫秒时间戳}。
func allocateArtifactVersion(tx *gorm.DB, artifactID int64, preferred string) (string, error) {
	ver := strings.TrimSpace(preferred)
	if ver == "" {
		ver = fmt.Sprintf("v%d", time.Now().UnixMilli())
	}
	ver = truncateNacosVersionField(ver, 64)
	if ver == "" {
		ver = fmt.Sprintf("v%d", time.Now().UnixMilli())
	}
	for i := 0; i < 32; i++ {
		var n int64
		if err := tx.Model(&model.NacosAIArtifactVersion{}).
			Where("artifact_id = ? AND version = ?", artifactID, ver).
			Count(&n).Error; err != nil {
			return "", err
		}
		if n == 0 {
			return ver, nil
		}
		suffix := "-" + strconv.FormatInt(time.Now().UnixMilli(), 10)
		base := ver
		if len(base)+len(suffix) > 64 {
			base = base[:max(0, 64-len(suffix))]
		}
		ver = truncateNacosVersionField(base+suffix, 64)
		if ver == "" {
			ver = fmt.Sprintf("v%d", time.Now().UnixMilli())
		}
		time.Sleep(time.Millisecond)
	}
	return "", errors.New("无法生成唯一版本号")
}

func truncateNacosVersionField(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	s = strings.TrimSpace(s)
	if len(s) <= maxBytes {
		return s
	}
	for len(s) > 0 && len(s) > maxBytes {
		_, sz := utf8.DecodeLastRuneInString(s)
		if sz <= 0 {
			break
		}
		s = s[:len(s)-sz]
	}
	return strings.TrimSpace(s)
}

func jsonSetTopLevelStringField(raw []byte, field, value string) ([]byte, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	m[field] = value
	return json.Marshal(m)
}

// isInternalMillisVersion 形如 v1735123456789（仅 v+十进制），用于判断是否跳过改写包内 _meta.json 等。
func isInternalMillisVersion(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if s[0] != 'v' && s[0] != 'V' {
		return false
	}
	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func repackZipFromFileMap(files map[string][]byte) ([]byte, error) {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, k := range keys {
		fw, err := w.Create(k)
		if err != nil {
			_ = w.Close()
			return nil, err
		}
		if _, err := fw.Write(files[k]); err != nil {
			_ = w.Close()
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// patchZipArtifactVersionMetadata 将入库版本号写入 Skill 的 _meta.json / package.json，或 AgentSpec 的 manifest.json。
// Skill 若无 _meta.json 且 ver 非内部 v+数字 形式，则创建 `_meta.json`。
func patchZipArtifactVersionMetadata(zipData []byte, kind, newVersion string) ([]byte, error) {
	root, files, err := parseZipArchive(zipData)
	if err != nil {
		return nil, err
	}
	ver := sanitizeNacosArtifactVersion(newVersion)
	if ver == "" {
		return zipData, nil
	}
	changed := false
	switch kind {
	case model.NacosAIKindSkill:
		metaPath := path.Join(root, "_meta.json")
		if b, ok := files[metaPath]; ok {
			nb, err := jsonSetTopLevelStringField(b, "version", ver)
			if err != nil {
				return nil, fmt.Errorf("_meta.json: %w", err)
			}
			files[metaPath] = nb
			changed = true
		} else if !isInternalMillisVersion(ver) {
			raw, err := json.Marshal(map[string]string{"version": ver})
			if err != nil {
				return nil, err
			}
			files[metaPath] = raw
			changed = true
		}
		pj := path.Join(root, "package.json")
		if b, ok := files[pj]; ok {
			nb, err := jsonSetTopLevelStringField(b, "version", ver)
			if err != nil {
				return nil, fmt.Errorf("package.json: %w", err)
			}
			files[pj] = nb
			changed = true
		}
	case model.NacosAIKindAgentSpec:
		mp := path.Join(root, "manifest.json")
		b, ok := files[mp]
		if !ok {
			return nil, fmt.Errorf("缺少 %s", mp)
		}
		nb, err := jsonSetTopLevelStringField(b, "version", ver)
		if err != nil {
			return nil, fmt.Errorf("manifest.json: %w", err)
		}
		files[mp] = nb
		changed = true
	default:
		return zipData, nil
	}
	if !changed {
		return zipData, nil
	}
	return repackZipFromFileMap(files)
}

func skillDescriptionFromMd(md string) string {
	md = strings.TrimSpace(md)
	if md == "" {
		return ""
	}
	lines := strings.Split(md, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		if len(ln) > 200 {
			return ln[:200]
		}
		return ln
	}
	if len(lines[0]) > 200 {
		return lines[0][:200]
	}
	return lines[0]
}

func agentspecDescriptionFromManifest(raw []byte) string {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	if d, ok := m["description"].(string); ok {
		return strings.TrimSpace(d)
	}
	if w, ok := m["worker"].(map[string]interface{}); ok {
		if s, ok := w["suggested_name"].(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// --- Namespace ---

func NormalizeNacosNamespaceID(ns string) string {
	ns = strings.TrimSpace(ns)
	if ns == "" {
		return "public"
	}
	return ns
}

// --- Labels ---

func parseArtifactLabelsJSON(s string) map[string]string {
	out := map[string]string{}
	s = strings.TrimSpace(s)
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func marshalArtifactLabels(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// --- DB helpers ---

func findArtifact(ns, kind, name string) (*model.NacosAIArtifact, error) {
	ns = NormalizeNacosNamespaceID(ns)
	var a model.NacosAIArtifact
	err := model.DB.Where("namespace_id = ? AND kind = ? AND name = ?", ns, kind, name).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func listVersions(artifactID int64) ([]model.NacosAIArtifactVersion, error) {
	var vs []model.NacosAIArtifactVersion
	err := model.DB.Where("artifact_id = ?", artifactID).Order("created_at desc").Find(&vs).Error
	return vs, err
}

func pickLatestVersionByStatus(vs []model.NacosAIArtifactVersion, st string) string {
	for _, v := range vs {
		if v.Status == st {
			return v.Version
		}
	}
	return ""
}

func countOnline(vs []model.NacosAIArtifactVersion) int {
	n := 0
	for _, v := range vs {
		if v.Status == model.NacosAIVersionOnline {
			n++
		}
	}
	return n
}

func findVersion(vs []model.NacosAIArtifactVersion, ver string) *model.NacosAIArtifactVersion {
	for i := range vs {
		if vs[i].Version == ver {
			return &vs[i]
		}
	}
	return nil
}

// ResolveVersionForGet 优先级: label > version 参数 > labels["latest"] > 最新 online。
func ResolveVersionForGet(a *model.NacosAIArtifact, vs []model.NacosAIArtifactVersion, label, version string) (*model.NacosAIArtifactVersion, error) {
	labels := parseArtifactLabelsJSON(a.LabelsJSON)
	if strings.TrimSpace(label) != "" {
		if v, ok := labels[label]; ok {
			if pv := findVersion(vs, v); pv != nil {
				if pv.Status == model.NacosAIVersionOffline {
					return nil, fmt.Errorf("label %q 指向已下线版本", label)
				}
				return pv, nil
			}
		}
		return nil, fmt.Errorf("label %q 未找到或未指向有效版本", label)
	}
	if strings.TrimSpace(version) != "" {
		if pv := findVersion(vs, version); pv != nil {
			return pv, nil
		}
		return nil, fmt.Errorf("版本 %q 不存在", version)
	}
	if v, ok := labels["latest"]; ok {
		if pv := findVersion(vs, v); pv != nil && pv.Status == model.NacosAIVersionOnline {
			return pv, nil
		}
	}
	for _, x := range vs {
		if x.Status == model.NacosAIVersionOnline {
			return &x, nil
		}
	}
	return nil, errors.New("没有可下载的已发布版本")
}

// --- Upload ---

func NacosAIUploadSkill(namespace string, zipData []byte, ownerUserID int, uploadFilename string) error {
	root, files, err := parseZipArchive(zipData)
	if err != nil {
		return err
	}
	skillPath := path.Join(root, "SKILL.md")
	if _, ok := files[skillPath]; !ok {
		return fmt.Errorf("缺少 %s", skillPath)
	}
	md := string(files[skillPath])
	desc := skillDescriptionFromMd(md)

	artifactName := root
	if root == "" {
		artifactName = skillArtifactNameFromZipFilename(uploadFilename)
		if artifactName == "" {
			return errors.New("ZIP 为根目录平铺（SKILL.md 在压缩包根目录）时，请使用「技能名.zip」作为上传文件名，或改为「技能目录/SKILL.md」结构")
		}
	}

	hint := skillVersionFromZip(root, files)
	return nacosAIUploadArtifact(namespace, model.NacosAIKindSkill, artifactName, desc, "", hint, zipData, ownerUserID)
}

func NacosAIUploadAgentSpec(namespace string, zipData []byte, ownerUserID int, overwrite bool, uploadFilename string) error {
	root, files, err := parseZipArchive(zipData)
	if err != nil {
		return err
	}
	manPath := path.Join(root, "manifest.json")
	raw, ok := files[manPath]
	if !ok {
		return fmt.Errorf("缺少 %s", manPath)
	}
	desc := agentspecDescriptionFromManifest(raw)
	verHint := sanitizeNacosArtifactVersion(artifactVersionFromTopLevelJSON(raw))

	artifactName := root
	if root == "" {
		artifactName = skillArtifactNameFromZipFilename(uploadFilename)
		if artifactName == "" {
			return errors.New("ZIP 为根目录平铺（manifest.json 在压缩包根目录）时，请使用「资源名.zip」作为上传文件名，或改为「目录/manifest.json」结构")
		}
	}

	ns := NormalizeNacosNamespaceID(namespace)
	if !overwrite {
		if _, err := findArtifact(ns, model.NacosAIKindAgentSpec, artifactName); err == nil {
			return fmt.Errorf("AgentSpec %q 已存在（overwrite=false）", artifactName)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	return nacosAIUploadArtifact(ns, model.NacosAIKindAgentSpec, artifactName, desc, "", verHint, zipData, ownerUserID)
}

func nacosAIUploadArtifact(namespace, kind, name, description, bizTags, preferredVersion string, zipData []byte, ownerUserID int) error {
	var createdVersion string
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var a model.NacosAIArtifact
		err := tx.Where("namespace_id = ? AND kind = ? AND name = ?", namespace, kind, name).First(&a).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			a = model.NacosAIArtifact{
				NamespaceId:   namespace,
				Kind:          kind,
				Name:          name,
				Description:   description,
				BizTags:       bizTags,
				LabelsJSON:    "{}",
				Enable:        true,
				OwnerUserId:   ownerUserID,
				DownloadCount: 0,
			}
			if err := tx.Create(&a).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			a.Description = description
			if bizTags != "" {
				a.BizTags = bizTags
			}
			if err := tx.Save(&a).Error; err != nil {
				return err
			}
		}
		ver, err := allocateArtifactVersion(tx, a.Id, preferredVersion)
		if err != nil {
			return err
		}
		createdVersion = ver
		nv := &model.NacosAIArtifactVersion{
			ArtifactId: a.Id,
			Version:    ver,
			Status:     model.NacosAIVersionEditing,
		}
		if NacosZipStorageBackend() == "db" {
			nv.ZipBytes = zipData
			nv.ZipStorageKind = "db"
		}
		return tx.Create(nv).Error
	})
	if err != nil {
		return err
	}
	if NacosZipStorageBackend() == "db" {
		return nil
	}
	var ar model.NacosAIArtifact
	if err := model.DB.Where("namespace_id = ? AND kind = ? AND name = ?", namespace, kind, name).First(&ar).Error; err != nil {
		return err
	}
	var last model.NacosAIArtifactVersion
	if err := model.DB.Where("artifact_id = ? AND version = ?", ar.Id, createdVersion).Order("id desc").First(&last).Error; err != nil {
		return err
	}
	return NacosPersistVersionZIP(&last, namespace, kind, ar.Id, zipData)
}

// --- Submit / Publish ---

func NacosAISubmit(namespace, kind, resourceName, version string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, kind, resourceName)
	if err != nil {
		return err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return err
	}
	var target *model.NacosAIArtifactVersion
	if strings.TrimSpace(version) != "" {
		target = findVersion(vs, version)
	} else {
		for i := range vs {
			if vs[i].Status == model.NacosAIVersionEditing {
				target = &vs[i]
				break
			}
		}
	}
	if target == nil {
		return errors.New("没有可提交的草稿版本")
	}
	if target.Status != model.NacosAIVersionEditing {
		return fmt.Errorf("版本 %s 状态不是 editing", target.Version)
	}
	return model.DB.Model(target).Updates(map[string]interface{}{
		"status": model.NacosAIVersionReviewing,
	}).Error
}

func NacosAIPublish(namespace, kind, resourceName, version string, updateLatestLabel bool, force bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, kind, resourceName)
	if err != nil {
		return err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return err
	}
	target := findVersion(vs, version)
	if target == nil {
		return fmt.Errorf("版本 %q 不存在", version)
	}
	if !force && target.Status != model.NacosAIVersionReviewing {
		return fmt.Errorf("版本 %s 需先 submit 进入 reviewing 才能发布", version)
	}
	if force && target.Status != model.NacosAIVersionReviewing && target.Status != model.NacosAIVersionEditing {
		return fmt.Errorf("force 发布仅允许 editing 或 reviewing 状态，当前为 %s", target.Status)
	}
	if err := model.DB.Model(target).Updates(map[string]interface{}{
		"status": model.NacosAIVersionOnline,
	}).Error; err != nil {
		return err
	}
	if updateLatestLabel {
		labels := parseArtifactLabelsJSON(a.LabelsJSON)
		labels["latest"] = version
		a.LabelsJSON = marshalArtifactLabels(labels)
		return model.DB.Model(a).Update("labels_json", a.LabelsJSON).Error
	}
	return nil
}

// NacosAIUpdateSkillMetadata 更新 Skill 元数据（不含改名；name 与 ZIP 根目录一致）。
func NacosAIUpdateSkillMetadata(namespace, skillName string, description *string, bizTags *string, enable *bool, scope *string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(skillName)
	if name == "" {
		return errors.New("name 必填")
	}
	a, err := findArtifact(ns, model.NacosAIKindSkill, name)
	if err != nil {
		return err
	}
	up := map[string]interface{}{}
	if description != nil {
		up["description"] = *description
	}
	if bizTags != nil {
		up["biz_tags"] = *bizTags
	}
	if enable != nil {
		up["enable"] = *enable
	}
	if scope != nil {
		s := strings.TrimSpace(*scope)
		if s != "" {
			up["scope"] = s
		}
	}
	if len(up) == 0 {
		return errors.New("无更新字段")
	}
	return model.DB.Model(a).Updates(up).Error
}

// NacosAIUpdateAgentSpecMetadata 更新 AgentSpec 元数据。
func NacosAIUpdateAgentSpecMetadata(namespace, specName string, description *string, bizTags *string, enable *bool, scope *string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(specName)
	if name == "" {
		return errors.New("name 必填")
	}
	a, err := findArtifact(ns, model.NacosAIKindAgentSpec, name)
	if err != nil {
		return err
	}
	up := map[string]interface{}{}
	if description != nil {
		up["description"] = *description
	}
	if bizTags != nil {
		up["biz_tags"] = *bizTags
	}
	if enable != nil {
		up["enable"] = *enable
	}
	if scope != nil {
		s := strings.TrimSpace(*scope)
		if s != "" {
			up["scope"] = s
		}
	}
	if len(up) == 0 {
		return errors.New("无更新字段")
	}
	return model.DB.Model(a).Updates(up).Error
}

// NacosAIUpdateArtifactLabels 合并或替换 artifact 的 labels（label->version 字符串）。
func NacosAIUpdateArtifactLabels(namespace, kind, resourceName string, labels map[string]string, replace bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(resourceName)
	if name == "" {
		return errors.New("name 必填")
	}
	if kind != model.NacosAIKindSkill && kind != model.NacosAIKindAgentSpec {
		return errors.New("kind 无效")
	}
	a, err := findArtifact(ns, kind, name)
	if err != nil {
		return err
	}
	var merged map[string]string
	if replace {
		merged = map[string]string{}
	} else {
		merged = parseArtifactLabelsJSON(a.LabelsJSON)
	}
	for k, v := range labels {
		kk := strings.TrimSpace(k)
		if kk == "" {
			continue
		}
		merged[kk] = strings.TrimSpace(v)
	}
	a.LabelsJSON = marshalArtifactLabels(merged)
	return model.DB.Model(a).Update("labels_json", a.LabelsJSON).Error
}

// NacosAIDeleteAgentSpec 删除 AgentSpec 及其全部版本与 ZIP。
func NacosAIDeleteAgentSpec(namespace, specName string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(specName)
	if name == "" {
		return errors.New("name 必填")
	}
	a, err := findArtifact(ns, model.NacosAIKindAgentSpec, name)
	if err != nil {
		return err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return err
	}
	for i := range vs {
		if err := NacosRemoveStoredZIP(&vs[i]); err != nil {
			return err
		}
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("artifact_id = ?", a.Id).Delete(&model.NacosAIArtifactVersion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.NacosAIArtifact{}, a.Id).Error
	})
}

// NacosAIDeleteSkill 删除 Skill 及其全部版本（并尽力清理 local/s3 上的 ZIP）。
func NacosAIDeleteSkill(namespace, skillName string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	name := strings.TrimSpace(skillName)
	if name == "" {
		return errors.New("name 必填")
	}
	a, err := findArtifact(ns, model.NacosAIKindSkill, name)
	if err != nil {
		return err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return err
	}
	for i := range vs {
		if err := NacosRemoveStoredZIP(&vs[i]); err != nil {
			return err
		}
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("artifact_id = ?", a.Id).Delete(&model.NacosAIArtifactVersion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.NacosAIArtifact{}, a.Id).Error
	})
}

// --- List / Describe ---

type NacosSkillListItem struct {
	NamespaceId      string            `json:"namespaceId,omitempty"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Owner            string            `json:"owner,omitempty"`
	Enable           bool              `json:"enable"`
	Scope            string            `json:"scope,omitempty"`
	BizTags          string            `json:"bizTags,omitempty"`
	From             string            `json:"from,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	EditingVersion   string            `json:"editingVersion,omitempty"`
	ReviewingVersion string            `json:"reviewingVersion,omitempty"`
	OnlineCnt        *int              `json:"onlineCnt,omitempty"`
	DownloadCount    *int64            `json:"downloadCount,omitempty"`
	UpdateTime       *int64            `json:"updateTime,omitempty"`
}

type NacosSkillVersionSummary struct {
	Version             string `json:"version"`
	Status              string `json:"status"`
	Author              string `json:"author,omitempty"`
	CommitMsg           string `json:"commitMsg,omitempty"`
	CreateTime          *int64 `json:"createTime,omitempty"`
	UpdateTime          *int64 `json:"updateTime,omitempty"`
	PublishPipelineInfo string `json:"publishPipelineInfo,omitempty"`
	DownloadCount       *int64 `json:"downloadCount,omitempty"`
}

type NacosSkillDetail struct {
	NacosSkillListItem
	Versions []NacosSkillVersionSummary `json:"versions,omitempty"`
}

type NacosSkillListData struct {
	TotalCount     int                  `json:"totalCount"`
	PageNumber     int                  `json:"pageNumber"`
	PagesAvailable int                  `json:"pagesAvailable"`
	PageItems      []NacosSkillListItem `json:"pageItems"`
}

func ptrI64(ms int64) *int64 { return &ms }
func ptrI(n int) *int       { return &n }

func NacosAIListSkills(namespace, skillNameFilter string, pageNo, pageSize int) (*NacosSkillListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosAIArtifact{}).Where("namespace_id = ? AND kind = ?", ns, model.NacosAIKindSkill)
	if strings.TrimSpace(skillNameFilter) != "" {
		q = q.Where("name = ?", strings.TrimSpace(skillNameFilter))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var arts []model.NacosAIArtifact
	offset := (pageNo - 1) * pageSize
	if err := q.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&arts).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosSkillListItem, 0, len(arts))
	for _, a := range arts {
		vs, _ := listVersions(a.Id)
		on := countOnline(vs)
		ut := a.UpdatedAt.UnixMilli()
		item := NacosSkillListItem{
			NamespaceId:      ns,
			Name:             a.Name,
			Description:      a.Description,
			Enable:           a.Enable,
			Scope:            strings.TrimSpace(a.Scope),
			BizTags:          a.BizTags,
			Labels:           parseArtifactLabelsJSON(a.LabelsJSON),
			EditingVersion:   pickLatestVersionByStatus(vs, model.NacosAIVersionEditing),
			ReviewingVersion: pickLatestVersionByStatus(vs, model.NacosAIVersionReviewing),
			OnlineCnt:        ptrI(on),
			DownloadCount:    ptrI64(a.DownloadCount),
			UpdateTime:       ptrI64(ut),
		}
		items = append(items, item)
	}
	return &NacosSkillListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

func NacosAIDescribeSkill(namespace, skillName string) (*NacosSkillDetail, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindSkill, skillName)
	if err != nil {
		return nil, err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return nil, err
	}
	on := countOnline(vs)
	ut := a.UpdatedAt.UnixMilli()
	item := NacosSkillListItem{
		NamespaceId:      ns,
		Name:             a.Name,
		Description:      a.Description,
		Enable:           a.Enable,
		Scope:            strings.TrimSpace(a.Scope),
		BizTags:          a.BizTags,
		Labels:           parseArtifactLabelsJSON(a.LabelsJSON),
		EditingVersion:   pickLatestVersionByStatus(vs, model.NacosAIVersionEditing),
		ReviewingVersion: pickLatestVersionByStatus(vs, model.NacosAIVersionReviewing),
		OnlineCnt:        ptrI(on),
		DownloadCount:    ptrI64(a.DownloadCount),
		UpdateTime:       ptrI64(ut),
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].CreatedAt.After(vs[j].CreatedAt) })
	sums := make([]NacosSkillVersionSummary, 0, len(vs))
	for _, v := range vs {
		ct := v.CreatedAt.UnixMilli()
		utv := v.UpdatedAt.UnixMilli()
		dc := v.DownloadCount
		sums = append(sums, NacosSkillVersionSummary{
			Version:       v.Version,
			Status:        mapNacosAIVersionStatusForConsoleUI(v.Status),
			Author:        v.Author,
			CommitMsg:     v.CommitMsg,
			CreateTime:    ptrI64(ct),
			UpdateTime:    ptrI64(utv),
			DownloadCount: &dc,
		})
	}
	return &NacosSkillDetail{NacosSkillListItem: item, Versions: sums}, nil
}

// NacosAIGetSkillZIP 仅允许下载已发布(online)版本；label / version 与 nacos-cli 优先级一致。
func NacosAIGetSkillZIP(namespace, skillName, label, version string) ([]byte, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindSkill, skillName)
	if err != nil {
		return nil, err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return nil, err
	}
	var pv *model.NacosAIArtifactVersion
	if strings.TrimSpace(label) != "" {
		labels := parseArtifactLabelsJSON(a.LabelsJSON)
		if v, ok := labels[label]; ok {
			pv = findVersion(vs, v)
		}
		if pv == nil {
			return nil, fmt.Errorf("label %q 未找到", label)
		}
	} else if strings.TrimSpace(version) != "" {
		pv = findVersion(vs, version)
		if pv == nil {
			return nil, fmt.Errorf("版本 %q 不存在", version)
		}
	} else {
		pv, err = ResolveVersionForGet(a, vs, "", "")
		if err != nil {
			return nil, err
		}
	}
	if pv.Status != model.NacosAIVersionOnline {
		return nil, errors.New("仅已发布(online)版本可下载")
	}
	_ = model.DB.Model(pv).UpdateColumn("download_count", gorm.Expr("download_count + ?", 1))
	_ = model.DB.Model(a).UpdateColumn("download_count", gorm.Expr("download_count + ?", 1))
	zipB, err := NacosLoadVersionZIP(pv)
	if err != nil {
		return nil, err
	}
	return zipB, nil
}

// NacosAIGetSkillZIPAdmin 管理端下载任意状态版本的 ZIP（用于排障与导出）。
func NacosAIGetSkillZIPAdmin(namespace, skillName, version string) ([]byte, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindSkill, skillName)
	if err != nil {
		return nil, err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(version) == "" {
		return nil, errors.New("version 必填")
	}
	pv := findVersion(vs, version)
	if pv == nil {
		return nil, fmt.Errorf("版本 %q 不存在", version)
	}
	return NacosLoadVersionZIP(pv)
}

// NacosAIGetAgentSpecZIPAdmin 管理端下载任意状态版本的 ZIP（与 Skill 对称）。
func NacosAIGetAgentSpecZIPAdmin(namespace, specName, version string) ([]byte, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindAgentSpec, specName)
	if err != nil {
		return nil, err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(version) == "" {
		return nil, errors.New("version 必填")
	}
	pv := findVersion(vs, version)
	if pv == nil {
		return nil, fmt.Errorf("版本 %q 不存在", version)
	}
	return NacosLoadVersionZIP(pv)
}

// NacosAIArtifactCreateDraftFromVersion 从已有版本复制 ZIP，新建一条 editing 版本（控制台「基于此版本创建草稿」）。
// preferredVersion 非空时经 sanitize 与去重后作为新版本号；为空则使用 v{毫秒时间戳}。
// 新版本号若非内部 v+数字 形式，会同步写入 Skill 的 _meta.json / package.json 或 AgentSpec 的 manifest.json。
func NacosAIArtifactCreateDraftFromVersion(namespace, kind, resourceName, basedOnVersion, preferredVersion, commitMsg string) (newVersion string, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	nm := strings.TrimSpace(resourceName)
	if nm == "" {
		return "", errors.New("资源名必填")
	}
	bv := strings.TrimSpace(basedOnVersion)
	if bv == "" {
		return "", errors.New("basedOnVersion 必填")
	}
	a, err := findArtifact(ns, kind, nm)
	if err != nil {
		return "", err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return "", err
	}
	src := findVersion(vs, bv)
	if src == nil {
		return "", fmt.Errorf("源版本 %q 不存在", basedOnVersion)
	}
	zipData, err := NacosLoadVersionZIP(src)
	if err != nil {
		return "", err
	}
	hint := strings.TrimSpace(preferredVersion)
	if hint != "" {
		hint = sanitizeNacosArtifactVersion(hint)
		if hint == "" {
			return "", errors.New("targetVersion 格式不合法")
		}
	}
	var createdVersion string
	var zipOut []byte
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		ver, err := allocateArtifactVersion(tx, a.Id, hint)
		if err != nil {
			return err
		}
		z := zipData
		if !isInternalMillisVersion(ver) {
			patched, err := patchZipArtifactVersionMetadata(zipData, kind, ver)
			if err != nil {
				return err
			}
			z = patched
		}
		nv := &model.NacosAIArtifactVersion{
			ArtifactId: a.Id,
			Version:    ver,
			Status:     model.NacosAIVersionEditing,
			CommitMsg:  strings.TrimSpace(commitMsg),
		}
		if NacosZipStorageBackend() == "db" {
			nv.ZipBytes = z
			nv.ZipStorageKind = "db"
		}
		if err := tx.Create(nv).Error; err != nil {
			return err
		}
		createdVersion = ver
		zipOut = z
		return nil
	})
	if err != nil {
		return "", err
	}
	if NacosZipStorageBackend() == "db" {
		return createdVersion, nil
	}
	var ar model.NacosAIArtifact
	if err := model.DB.Where("namespace_id = ? AND kind = ? AND name = ?", ns, kind, nm).First(&ar).Error; err != nil {
		return "", err
	}
	var last model.NacosAIArtifactVersion
	if err := model.DB.Where("artifact_id = ? AND version = ?", ar.Id, createdVersion).Order("id desc").First(&last).Error; err != nil {
		return "", err
	}
	if err := NacosPersistVersionZIP(&last, ns, kind, ar.Id, zipOut); err != nil {
		return "", err
	}
	return createdVersion, nil
}

// BuildAgentSpecZipBytes 从 manifest JSON 生成 `<name>/manifest.json` 结构的 ZIP，供控制台草稿与上传复用。
func BuildAgentSpecZipBytes(artifactDirName string, manifestJSON []byte) ([]byte, error) {
	if !json.Valid(manifestJSON) {
		return nil, errors.New("manifest 须为合法 JSON")
	}
	name := strings.TrimSpace(artifactDirName)
	if name == "" {
		return nil, errors.New("资源名必填")
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	zpath := path.Join(name, "manifest.json")
	f, err := w.Create(zpath)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(manifestJSON); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildAgentSpecZipFromEditorCard 根据控制台 agentSpecCard（manifest + resource 文件树）生成 ZIP。
func BuildAgentSpecZipFromEditorCard(artifactDirName, manifestContent string, resources map[string]json.RawMessage) ([]byte, error) {
	name := strings.TrimSpace(artifactDirName)
	if name == "" {
		return nil, errors.New("资源名必填")
	}
	man := strings.TrimSpace(manifestContent)
	if man == "" {
		man = "{}"
	}
	if !json.Valid([]byte(man)) {
		return nil, errors.New("manifest 须为合法 JSON")
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	mf, err := w.Create(path.Join(name, "manifest.json"))
	if err != nil {
		return nil, err
	}
	if _, err := mf.Write([]byte(man)); err != nil {
		_ = w.Close()
		return nil, err
	}
	type resItem struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	for key, raw := range resources {
		var it resItem
		_ = json.Unmarshal(raw, &it)
		rel := strings.TrimSpace(it.Name)
		if rel == "" {
			rel = strings.TrimSpace(key)
		}
		rel = strings.Trim(strings.ReplaceAll(rel, "\\", "/"), "/")
		if rel == "" || strings.Contains(rel, "..") {
			continue
		}
		base := path.Base(rel)
		if strings.EqualFold(base, "manifest.json") {
			continue
		}
		zp := path.Join(name, rel)
		f, err := w.Create(zp)
		if err != nil {
			_ = w.Close()
			return nil, err
		}
		if _, err := f.Write([]byte(it.Content)); err != nil {
			_ = w.Close()
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NacosAIDeleteEditingArtifactVersions 删除某 Skill/AgentSpec 的全部 editing 版本（ZIP 与行）。
func NacosAIDeleteEditingArtifactVersions(namespace, kind, name string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	nm := strings.TrimSpace(name)
	if nm == "" {
		return errors.New("name 必填")
	}
	a, err := findArtifact(ns, kind, nm)
	if err != nil {
		return err
	}
	var vs []model.NacosAIArtifactVersion
	if err := model.DB.Where("artifact_id = ? AND status = ?", a.Id, model.NacosAIVersionEditing).Find(&vs).Error; err != nil {
		return err
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		for i := range vs {
			_ = NacosRemoveStoredZIP(&vs[i])
			if err := tx.Delete(&model.NacosAIArtifactVersion{}, vs[i].Id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// NacosAIArtifactVersionSetOffline 将 online 版本置为 offline（客户端拉取将不可解析到该版本）。
func NacosAIArtifactVersionSetOffline(namespace, kind, name, version string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, kind, strings.TrimSpace(name))
	if err != nil {
		return err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return err
	}
	pv := findVersion(vs, strings.TrimSpace(version))
	if pv == nil {
		return fmt.Errorf("版本 %q 不存在", version)
	}
	if pv.Status != model.NacosAIVersionOnline {
		return fmt.Errorf("仅 online 版本可下线，当前为 %s", pv.Status)
	}
	return model.DB.Model(pv).Update("status", model.NacosAIVersionOffline).Error
}

// NacosAIArtifactVersionSetOnlineFromOffline 将 offline 恢复为 online。
func NacosAIArtifactVersionSetOnlineFromOffline(namespace, kind, name, version string) error {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, kind, strings.TrimSpace(name))
	if err != nil {
		return err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return err
	}
	pv := findVersion(vs, strings.TrimSpace(version))
	if pv == nil {
		return fmt.Errorf("版本 %q 不存在", version)
	}
	if pv.Status != model.NacosAIVersionOffline {
		return fmt.Errorf("仅 offline 版本可恢复上线，当前为 %s", pv.Status)
	}
	return model.DB.Model(pv).Update("status", model.NacosAIVersionOnline).Error
}

// NacosAIArtifactVersionEnsureOnline 将指定版本置为对客户端可见的 online：已是 online 则仅可选更新 latest；offline 则恢复；editing/reviewing 则走 force 发布。
func NacosAIArtifactVersionEnsureOnline(namespace, kind, name, version string, updateLatestLabel bool) error {
	ns := NormalizeNacosNamespaceID(namespace)
	nm := strings.TrimSpace(name)
	ver := strings.TrimSpace(version)
	if nm == "" || ver == "" {
		return errors.New("name 与 version 必填")
	}
	a, err := findArtifact(ns, kind, nm)
	if err != nil {
		return err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return err
	}
	target := findVersion(vs, ver)
	if target == nil {
		return fmt.Errorf("版本 %q 不存在", version)
	}
	switch target.Status {
	case model.NacosAIVersionOnline:
		if updateLatestLabel {
			labels := parseArtifactLabelsJSON(a.LabelsJSON)
			labels["latest"] = ver
			a.LabelsJSON = marshalArtifactLabels(labels)
			return model.DB.Model(a).Update("labels_json", a.LabelsJSON).Error
		}
		return nil
	case model.NacosAIVersionOffline:
		if err := model.DB.Model(target).Update("status", model.NacosAIVersionOnline).Error; err != nil {
			return err
		}
		if updateLatestLabel {
			labels := parseArtifactLabelsJSON(a.LabelsJSON)
			labels["latest"] = ver
			a.LabelsJSON = marshalArtifactLabels(labels)
			return model.DB.Model(a).Update("labels_json", a.LabelsJSON).Error
		}
		return nil
	case model.NacosAIVersionEditing, model.NacosAIVersionReviewing:
		return NacosAIPublish(ns, kind, nm, ver, updateLatestLabel, true)
	default:
		return fmt.Errorf("版本状态 %s 不支持上线", target.Status)
	}
}

// --- AgentSpec list/describe/get ---

type NacosAgentSpecListItem struct {
	NamespaceId      string            `json:"namespaceId,omitempty"`
	Name             string            `json:"name"`
	Description      *string           `json:"description"`
	Enable           bool              `json:"enable"`
	Labels           map[string]string `json:"labels"`
	Scope            *string           `json:"scope,omitempty"`
	BizTags          *string           `json:"bizTags,omitempty"`
	EditingVersion   *string           `json:"editingVersion"`
	ReviewingVersion *string           `json:"reviewingVersion"`
	OnlineCnt        int               `json:"onlineCnt"`
	DownloadCount    *int64            `json:"downloadCount,omitempty"`
	UpdateTime       int64             `json:"updateTime"`
}

type NacosAgentSpecVersionSummary struct {
	Version             string `json:"version"`
	Status              string `json:"status"`
	Author              string `json:"author,omitempty"`
	Description         string `json:"description,omitempty"`
	CreateTime          *int64 `json:"createTime,omitempty"`
	UpdateTime          *int64 `json:"updateTime,omitempty"`
	PublishPipelineInfo string `json:"publishPipelineInfo,omitempty"`
	DownloadCount       *int64 `json:"downloadCount,omitempty"`
}

type NacosAgentSpecDetail struct {
	NacosAgentSpecListItem
	Versions []NacosAgentSpecVersionSummary `json:"versions,omitempty"`
}

type NacosAgentSpecListData struct {
	TotalCount     int                      `json:"totalCount"`
	PageNumber     int                      `json:"pageNumber"`
	PagesAvailable int                      `json:"pagesAvailable"`
	PageItems      []NacosAgentSpecListItem `json:"pageItems"`
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func NacosAIListAgentSpecs(namespace, nameFilter, search string, pageNo, pageSize int) (*NacosAgentSpecListData, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	q := model.DB.Model(&model.NacosAIArtifact{}).Where("namespace_id = ? AND kind = ?", ns, model.NacosAIKindAgentSpec)
	if strings.TrimSpace(nameFilter) != "" {
		if search == "accurate" || search == "" {
			q = q.Where("name = ?", strings.TrimSpace(nameFilter))
		} else {
			q = q.Where("name LIKE ?", "%"+strings.TrimSpace(nameFilter)+"%")
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var arts []model.NacosAIArtifact
	offset := (pageNo - 1) * pageSize
	if err := q.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&arts).Error; err != nil {
		return nil, err
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if total == 0 {
		pages = 0
	}
	items := make([]NacosAgentSpecListItem, 0, len(arts))
	for _, a := range arts {
		vs, _ := listVersions(a.Id)
		on := countOnline(vs)
		ed := pickLatestVersionByStatus(vs, model.NacosAIVersionEditing)
		rv := pickLatestVersionByStatus(vs, model.NacosAIVersionReviewing)
		dc := a.DownloadCount
		desc := a.Description
		bt := a.BizTags
		sc := strings.TrimSpace(a.Scope)
		if sc == "" {
			sc = "PUBLIC"
		}
		item := NacosAgentSpecListItem{
			NamespaceId:      a.NamespaceId,
			Name:             a.Name,
			Description:      strPtr(desc),
			Enable:           a.Enable,
			Labels:           parseArtifactLabelsJSON(a.LabelsJSON),
			Scope:            strPtr(sc),
			EditingVersion:   strPtr(ed),
			ReviewingVersion: strPtr(rv),
			OnlineCnt:        on,
			DownloadCount:    &dc,
			UpdateTime:       a.UpdatedAt.UnixMilli(),
		}
		if bt != "" {
			item.BizTags = &bt
		}
		items = append(items, item)
	}
	return &NacosAgentSpecListData{
		TotalCount:     int(total),
		PageNumber:     pageNo,
		PagesAvailable: pages,
		PageItems:      items,
	}, nil
}

func NacosAIDescribeAgentSpec(namespace, specName string) (*NacosAgentSpecDetail, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindAgentSpec, specName)
	if err != nil {
		return nil, err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return nil, err
	}
	on := countOnline(vs)
	ed := pickLatestVersionByStatus(vs, model.NacosAIVersionEditing)
	rv := pickLatestVersionByStatus(vs, model.NacosAIVersionReviewing)
	dc := a.DownloadCount
	desc := a.Description
	bt := a.BizTags
	sc := strings.TrimSpace(a.Scope)
	if sc == "" {
		sc = "PUBLIC"
	}
	base := NacosAgentSpecListItem{
		NamespaceId:      a.NamespaceId,
		Name:             a.Name,
		Description:      strPtr(desc),
		Enable:           a.Enable,
		Labels:           parseArtifactLabelsJSON(a.LabelsJSON),
		Scope:            strPtr(sc),
		EditingVersion:   strPtr(ed),
		ReviewingVersion: strPtr(rv),
		OnlineCnt:        on,
		DownloadCount:    &dc,
		UpdateTime:       a.UpdatedAt.UnixMilli(),
	}
	if bt != "" {
		base.BizTags = &bt
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].CreatedAt.After(vs[j].CreatedAt) })
	sums := make([]NacosAgentSpecVersionSummary, 0, len(vs))
	for _, v := range vs {
		ct := v.CreatedAt.UnixMilli()
		utv := v.UpdatedAt.UnixMilli()
		dcv := v.DownloadCount
		sums = append(sums, NacosAgentSpecVersionSummary{
			Version:       v.Version,
			Status:        mapNacosAIVersionStatusForConsoleUI(v.Status),
			Author:        v.Author,
			CreateTime:    ptrI64(ct),
			UpdateTime:    ptrI64(utv),
			DownloadCount: &dcv,
		})
	}
	return &NacosAgentSpecDetail{NacosAgentSpecListItem: base, Versions: sums}, nil
}

// AgentSpecResource mirrors nacos-cli JSON.
type AgentSpecResource struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type AgentSpecPayload struct {
	NamespaceId string                       `json:"namespaceId"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	BizTags     string                       `json:"bizTags,omitempty"`
	Content     string                       `json:"content"`
	Resource    map[string]*AgentSpecResource `json:"resource,omitempty"`
}

func buildResourceRelativePath(resType, resName string) string {
	t := strings.TrimSpace(resType)
	n := strings.TrimSpace(resName)
	if t == "" {
		return n
	}
	prefix := t + "/"
	if strings.HasPrefix(n, prefix) {
		return n
	}
	return t + "/" + n
}

func isBinaryExt(lower string) bool {
	switch lower {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".pdf", ".zip", ".gz", ".wasm", ".bin", ".exe":
		return true
	default:
		return false
	}
}

func buildAgentSpecPayloadFromZip(zipData []byte, a *model.NacosAIArtifact) (*AgentSpecPayload, error) {
	root, files, err := parseZipArchive(zipData)
	if err != nil {
		return nil, err
	}
	manKey := path.Join(root, "manifest.json")
	manifestBytes, ok := files[manKey]
	if !ok {
		return nil, errors.New("ZIP 内缺少 manifest.json")
	}
	content := string(manifestBytes)
	resMap := make(map[string]*AgentSpecResource)
	for p, b := range files {
		if p == manKey {
			continue
		}
		var rel string
		if root == "" {
			rel = p
		} else {
			var okp bool
			rel, okp = strings.CutPrefix(p, root+"/")
			if !okp || rel == "" {
				continue
			}
		}
		if rel == "" {
			continue
		}
		parts := strings.SplitN(rel, "/", 2)
		var resType, resName string
		if len(parts) == 1 {
			resName = parts[0]
		} else {
			resType = parts[0]
			resName = parts[1]
		}
		ext := strings.ToLower(path.Ext(resName))
		meta := map[string]interface{}{}
		var body string
		if isBinaryExt(ext) {
			meta["encoding"] = "base64"
			body = base64.StdEncoding.EncodeToString(b)
		} else {
			body = string(b)
		}
		rr := &AgentSpecResource{
			Name:     resName,
			Type:     resType,
			Content:  body,
			Metadata: meta,
		}
		relPath := buildResourceRelativePath(resType, resName)
		resMap[relPath] = rr
	}
	return &AgentSpecPayload{
		NamespaceId: a.NamespaceId,
		Name:        a.Name,
		Description: a.Description,
		BizTags:     a.BizTags,
		Content:     content,
		Resource:    resMap,
	}, nil
}

// NacosAIDocumentAgentSpecVersion 管理端读取任意版本 ZIP 并解析为 console-ui-next AgentSpecDocument 形态。
func NacosAIDocumentAgentSpecVersion(namespace, specName, version string) (*AgentSpecPayload, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindAgentSpec, specName)
	if err != nil {
		return nil, err
	}
	zipData, err := NacosAIGetAgentSpecZIPAdmin(namespace, specName, version)
	if err != nil {
		return nil, err
	}
	return buildAgentSpecPayloadFromZip(zipData, a)
}

// NacosSkillConsoleRes console SkillDocument.resource 单项。
type NacosSkillConsoleRes struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NacosSkillConsoleDoc 与 console-ui-next SkillDocument 对齐。
type NacosSkillConsoleDoc struct {
	NamespaceId string                         `json:"namespaceId"`
	Name        string                         `json:"name"`
	Description string                       `json:"description"`
	SkillMd     string                         `json:"skillMd"`
	Resource    map[string]*NacosSkillConsoleRes `json:"resource"`
}

// NacosAIDocumentSkillVersion 管理端读取任意版本 ZIP 并解析为 console SkillDocument。
func NacosAIDocumentSkillVersion(namespace, skillName, version string) (*NacosSkillConsoleDoc, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindSkill, skillName)
	if err != nil {
		return nil, err
	}
	zipData, err := NacosAIGetSkillZIPAdmin(namespace, skillName, version)
	if err != nil {
		return nil, err
	}
	root, files, err := parseZipArchive(zipData)
	if err != nil {
		return nil, err
	}
	skillPath := path.Join(root, "SKILL.md")
	mdBytes, ok := files[skillPath]
	if !ok {
		return nil, fmt.Errorf("缺少 %s", skillPath)
	}
	resMap := make(map[string]*NacosSkillConsoleRes)
	for p, b := range files {
		if p == skillPath {
			continue
		}
		var rel string
		if root == "" {
			rel = p
		} else {
			var okp bool
			rel, okp = strings.CutPrefix(p, root+"/")
			if !okp || rel == "" {
				continue
			}
		}
		if rel == "" {
			continue
		}
		base := path.Base(rel)
		ext := strings.ToLower(path.Ext(base))
		meta := map[string]interface{}{}
		var body string
		if isBinaryExt(ext) {
			meta["encoding"] = "base64"
			body = base64.StdEncoding.EncodeToString(b)
		} else {
			body = string(b)
		}
		parts := strings.SplitN(rel, "/", 2)
		rt, rn := "", rel
		if len(parts) == 2 {
			rt, rn = parts[0], parts[1]
		}
		resMap[rel] = &NacosSkillConsoleRes{Name: rn, Type: rt, Content: body, Metadata: meta}
	}
	return &NacosSkillConsoleDoc{
		NamespaceId: a.NamespaceId,
		Name:        a.Name,
		Description: a.Description,
		SkillMd:     string(mdBytes),
		Resource:    resMap,
	}, nil
}

func NacosAIGetAgentSpecJSON(namespace, specName, label, version string) (*AgentSpecPayload, error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindAgentSpec, specName)
	if err != nil {
		return nil, err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return nil, err
	}
	var pv *model.NacosAIArtifactVersion
	if strings.TrimSpace(label) != "" {
		labels := parseArtifactLabelsJSON(a.LabelsJSON)
		if v, ok := labels[label]; ok {
			pv = findVersion(vs, v)
		}
		if pv == nil {
			return nil, fmt.Errorf("label %q 未找到", label)
		}
	} else if strings.TrimSpace(version) != "" {
		pv = findVersion(vs, version)
		if pv == nil {
			return nil, fmt.Errorf("版本 %q 不存在", version)
		}
	} else {
		pv, err = ResolveVersionForGet(a, vs, "", "")
		if err != nil {
			return nil, err
		}
	}
	if pv.Status != model.NacosAIVersionOnline {
		return nil, errors.New("仅已发布(online)版本可拉取")
	}
	zipData, err := NacosLoadVersionZIP(pv)
	if err != nil {
		return nil, err
	}
	out, err := buildAgentSpecPayloadFromZip(zipData, a)
	if err != nil {
		return nil, err
	}
	_ = model.DB.Model(pv).UpdateColumn("download_count", gorm.Expr("download_count + ?", 1))
	_ = model.DB.Model(a).UpdateColumn("download_count", gorm.Expr("download_count + ?", 1))
	return out, nil
}

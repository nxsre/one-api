package service

import "strings"

// NacosConsolePermissionListRow 供 GET /nacos/v3/auth/permission/list 的单行（Key 仅用于筛选，JSON 用 json:"-" 省略）。
type NacosConsolePermissionListRow struct {
	Role     string `json:"role"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Key      string `json:"-"`
}

func nacosConsolePermRowRole(key string) string {
	if strings.HasPrefix(key, "admin:") {
		return "ADMIN"
	}
	if strings.HasPrefix(key, "client:") {
		return "USER"
	}
	return "GLOBAL"
}

func nacosConsolePermSplitKey(key string) (resource, action string) {
	if strings.HasSuffix(key, ":read") {
		return strings.TrimSuffix(key, ":read"), "r"
	}
	if strings.HasSuffix(key, ":write") {
		return strings.TrimSuffix(key, ":write"), "w"
	}
	return key, "rw"
}

func nacosConsolePermissionListAllRows() []NacosConsolePermissionListRow {
	keys := NacosAllPermissionKeys()
	out := make([]NacosConsolePermissionListRow, 0, len(keys))
	for _, k := range keys {
		res, act := nacosConsolePermSplitKey(k)
		out = append(out, NacosConsolePermissionListRow{
			Role:     nacosConsolePermRowRole(k),
			Resource: res,
			Action:   act,
			Key:      k,
		})
	}
	return out
}

func nacosConsolePermRowMatches(r NacosConsolePermissionListRow, kw, likePat string, accurate bool) bool {
	kw = strings.TrimSpace(kw)
	if kw == "" {
		return true
	}
	if accurate {
		return r.Role == kw || r.Resource == kw || r.Action == kw || r.Key == kw
	}
	if likePat == "" {
		return true
	}
	pl := strings.ToLower(likePat)
	contains := func(s string) bool { return strings.Contains(strings.ToLower(s), pl) }
	return contains(r.Role) || contains(r.Resource) || contains(r.Action) || contains(r.Key)
}

// NacosConsolePermissionListPage 在静态权限目录上按关键词与 search=blur|accurate 过滤后分页。
func NacosConsolePermissionListPage(pageNo, pageSize int, keyword, mode string) (total int64, rows []NacosConsolePermissionListRow, err error) {
	if pageNo < 1 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 500 {
		pageSize = 500
	}
	accurate := strings.EqualFold(strings.TrimSpace(mode), "accurate")
	kw := strings.TrimSpace(keyword)
	likePat := strings.ReplaceAll(strings.ReplaceAll(kw, "%", ""), "_", "")

	all := nacosConsolePermissionListAllRows()
	filtered := make([]NacosConsolePermissionListRow, 0, len(all))
	for i := range all {
		if nacosConsolePermRowMatches(all[i], kw, likePat, accurate) {
			filtered = append(filtered, all[i])
		}
	}
	total = int64(len(filtered))
	start := (pageNo - 1) * pageSize
	if start >= len(filtered) {
		return total, nil, nil
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return total, filtered[start:end], nil
}

package channeltype

import "testing"

// TestSlugCoversAllValidTypes 确保每个有效渠道类型（[1, Dummy)）都注册了 slug，
// 避免后续 define 中追加新枚举时忘记同步映射。
func TestSlugCoversAllValidTypes(t *testing.T) {
	for id := 1; id < Dummy; id++ {
		slug := SlugByID(id)
		if slug == "" {
			t.Fatalf("channel type id %d 缺少 slug，请在 channelSlugByID 中补充", id)
		}
		if got := IDBySlug(slug); got != id {
			t.Fatalf("slug %q 反查到 id=%d，期望 %d", slug, got, id)
		}
	}
}

// TestSlugUniqueness 防止两个枚举值映射到同一个 slug。
func TestSlugUniqueness(t *testing.T) {
	seen := make(map[string]int, len(channelSlugByID))
	for id, slug := range channelSlugByID {
		if prev, ok := seen[slug]; ok {
			t.Fatalf("slug %q 同时映射到 channel type %d 和 %d", slug, prev, id)
		}
		seen[slug] = id
	}
}

// TestIDBySlugUnknownReturnsZero slug 未注册时应返回 Unknown(0)。
func TestIDBySlugUnknownReturnsZero(t *testing.T) {
	if got := IDBySlug("definitely-not-a-channel"); got != Unknown {
		t.Fatalf("未注册 slug 应返回 Unknown(%d)，实际 %d", Unknown, got)
	}
}

package controller

import "testing"

func TestMaskChannelKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"a", "****"},
		{"12345678", "****"},
		{"sk-1234567890abcd", "sk-1****abcd"},
		{"sk-proj-ABCDEFGHIJKLMNOP", "sk-p****MNOP"},
	}
	for _, c := range cases {
		if got := maskChannelKey(c.in); got != c.want {
			t.Fatalf("maskChannelKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// 脱敏串不得泄露 8 位以上密钥的中间内容。
func TestMaskChannelKeyNoLeak(t *testing.T) {
	full := "supersecretkey1234567890"
	masked := maskChannelKey(full)
	if masked == full {
		t.Fatal("masked equals full key")
	}
	if len([]rune(masked)) != 12 { // 4 + 4(****) + 4
		t.Fatalf("unexpected masked length: %q", masked)
	}
}

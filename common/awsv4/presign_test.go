package awsv4

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCheckPresignedExpiry_deadlineDoesNotAddFullClockSkew(t *testing.T) {
	// 签名时刻
	t0, err := time.Parse("20060102T150405Z", "20260115T120000Z")
	if err != nil {
		t.Fatal(err)
	}
	// 大时钟容差（如 config 里 s3_clock_skew_seconds=1800）：不得延长预签名截止
	hugeSkew := 30 * time.Minute
	// 签发后 31s：已过 30s 有效期，应判过期
	if err := checkPresignedExpiry("20260115T120000Z", 30, hugeSkew, t0.Add(31*time.Second)); err == nil {
		t.Fatal("expected expired")
	}
	// 签发后 29s：仍有效
	if err := checkPresignedExpiry("20260115T120000Z", 30, hugeSkew, t0.Add(29*time.Second)); err != nil {
		t.Fatalf("unexpected %v", err)
	}
}

func TestCheckPresignedExpiry_notYetValidRespectsSkew(t *testing.T) {
	t0, err := time.Parse("20060102T150405Z", "20260115T120000Z")
	if err != nil {
		t.Fatal(err)
	}
	skew := 2 * time.Minute
	// 比签名时刻早 3 分钟（但允许 2 分钟 skew 仅够「服务端略慢」侧；这里 now 比 t0 早太多应判 not yet valid）
	if err := checkPresignedExpiry("20260115T120000Z", 300, skew, t0.Add(-3*time.Minute)); err == nil {
		t.Fatal("expected not yet valid")
	}
	// 比签名早 1 分钟：在 skew 内应接受（尚未到「整段 expires」结束前都算签名有效链路上允许的早期访问）
	if err := checkPresignedExpiry("20260115T120000Z", 300, skew, t0.Add(-1*time.Minute)); err != nil {
		t.Fatalf("unexpected %v", err)
	}
}

func TestCanonicalQueryStringForPresign_order(t *testing.T) {
	q := url.Values{}
	q.Set("prefix", "a/b")
	q.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	q.Set("X-Amz-Signature", "should-be-omitted")
	got := canonicalQueryStringForPresign(q)
	if got == "" || len(got) < 20 {
		t.Fatalf("unexpected %q", got)
	}
	if strings.Contains(got, "X-Amz-Signature") || strings.Contains(got, "should-be-omitted") {
		t.Fatalf("signature must not appear in canonical query: %q", got)
	}
}

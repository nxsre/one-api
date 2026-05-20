package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSyncFromAliapiHTML(t *testing.T) {
	htmlContent := `
	<html>
	<body>
		<table>
			<tbody>
				<tr>
					<td>qwen-max</td>
					<td>阿里云</td>
					<td>推荐，支持FC</td>
					<td>输入 ￥0.04 / M tokens<br>输出 ￥0.12 / M tokens</td>
					<td>上下文 32.0k</td>
				</tr>
			</tbody>
		</table>
	</body>
	</html>
	`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(htmlContent))
	}))
	defer ts.Close()

	req := &modelCatalogSyncRequest{
		BaseURL: ts.URL,
	}

	rows, err := syncFromAliapiWithClient(context.Background(), req, &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("syncFromAliapi failed: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	r := rows[0]
	if r.ModelId != "qwen-max" {
		t.Errorf("expected model_id 'qwen-max', got %s", r.ModelId)
	}
	if r.OwnedBy != "阿里云" {
		t.Errorf("expected owned_by '阿里云', got %s", r.OwnedBy)
	}
	if r.CostInput != 0.04 {
		t.Errorf("expected input cost 0.04, got %f", r.CostInput)
	}
	if r.CostOutput != 0.12 {
		t.Errorf("expected output cost 0.12, got %f", r.CostOutput)
	}
	if r.ContextLimit != 32768 { // 32.0k * 1024
		t.Errorf("expected context limit 32768, got %d", r.ContextLimit)
	}
}

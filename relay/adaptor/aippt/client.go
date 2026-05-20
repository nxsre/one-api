package aippt

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/songquanpeng/one-api/common/client"
	"github.com/tidwall/gjson"
)

const defaultBaseURL = "https://co.aippt.cn"

// Client 调用 AiPPT 开放平台（与 aippt.sh 逻辑一致：标题生成 type=1 全流程）。
type Client struct {
	BaseURL   string
	AppKey    string
	SecretKey string
	UID       string
	HTTP      *http.Client
	tc        tokenCache
}

// GenerateResult 生成结果（PPTX 直链为上游临时地址）。
type GenerateResult struct {
	TaskID       string
	DesignID     string
	TemplateID   string
	DownloadURL  string
	OutlineDraft string
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	}
	return defaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	base := client.HTTPClient
	if base == nil {
		return client.NewOutboundHTTPClient(6 * time.Minute)
	}
	hc := *base
	hc.Timeout = 6 * time.Minute
	return &hc
}

func (c *Client) getToken() (string, error) {
	if tok, ok := c.tc.get(); ok {
		return tok, nil
	}
	ts := time.Now().Unix()
	u := c.base() + GrantTokenPath + "?uid=" + url.QueryEscape(c.uid())
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", c.AppKey)
	req.Header.Set("x-timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("x-signature", GenerateGrantSignature(c.SecretKey, "GET", GrantTokenPath, ts))

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if !GrantSuccessCode(body) {
		return "", fmt.Errorf("aippt token: %s", gjson.GetBytes(body, "msg").String())
	}
	tok := gjson.GetBytes(body, "data.token").String()
	if tok == "" {
		return "", fmt.Errorf("aippt token: empty data.token")
	}
	c.tc.set(tok, body)
	return tok, nil
}

func (c *Client) uid() string {
	if strings.TrimSpace(c.UID) != "" {
		return c.UID
	}
	return "openclaw_default"
}

func (c *Client) setAuth(h http.Header) error {
	tok, err := c.getToken()
	if err != nil {
		return err
	}
	h.Set("x-api-key", c.AppKey)
	h.Set("x-channel", "")
	h.Set("x-token", tok)
	return nil
}

// TestPresetList 渠道连通性测试：GET /api/ai/chat/config/list（预置词列表，轻量、不生成 PPT）。
// 见 https://open.aippt.cn/docs/zh/api/preset.html
func (c *Client) TestPresetList() error {
	saved := c.HTTP
	if base := client.HTTPClient; base != nil {
		tmp := *base
		tmp.Timeout = 45 * time.Second
		c.HTTP = &tmp
	} else {
		c.HTTP = client.NewOutboundHTTPClient(45 * time.Second)
	}
	defer func() { c.HTTP = saved }()

	q := url.Values{}
	q.Set("page", "1")
	q.Set("page_size", "1")
	u := c.base() + "/api/ai/chat/config/list?" + q.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	if err := c.setAuth(req.Header); err != nil {
		return fmt.Errorf("aippt 鉴权: %w", err)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("aippt 预置词列表: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := checkAipptCode(body, "preset list"); err != nil {
		return err
	}
	return nil
}

// GenerateFromTitle 标题 → 大纲 → 内容 → outline/save → 模板 → 导出 PPTX。
func (c *Client) GenerateFromTitle(title string) (*GenerateResult, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("aippt: title is empty")
	}

	taskID, err := c.createTaskType1(title)
	if err != nil {
		return nil, err
	}

	outline, err := c.fetchOutline(taskID)
	if err != nil {
		return nil, err
	}

	ticket, err := c.triggerContent(taskID, "")
	if err != nil {
		return nil, err
	}
	if err = c.waitContent(ticket, 2*time.Minute); err != nil {
		return nil, err
	}

	if err = c.outlineSaveFromPptData(taskID); err != nil {
		return nil, err
	}

	tpl, err := c.pickRandomTemplateID(taskID)
	if err != nil {
		return nil, err
	}

	designID, err := c.savePPT(taskID, tpl, title, "")
	if err != nil {
		return nil, err
	}

	dl, err := c.exportPPTXWithRetry(designID)
	if err != nil {
		return nil, err
	}

	return &GenerateResult{
		TaskID:       taskID,
		DesignID:     designID,
		TemplateID:   tpl,
		DownloadURL:  dl,
		OutlineDraft: outline,
	}, nil
}

func (c *Client) createTaskType1(title string) (string, error) {
	vals := url.Values{}
	vals.Set("title", title)
	vals.Set("type", "1")
	body := strings.NewReader(vals.Encode())
	req, err := http.NewRequest(http.MethodPost, c.base()+"/api/ai/chat/v2/task", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err = c.setAuth(req.Header); err != nil {
		return "", err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err = checkAipptCode(b, "create task"); err != nil {
		return "", err
	}
	id := gjson.GetBytes(b, "data.id")
	return strings.TrimSpace(id.String()), nil
}

func (c *Client) fetchOutline(taskID string) (string, error) {
	u := c.base() + "/api/ai/chat/outline?task_id=" + url.QueryEscape(taskID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	if err = c.setAuth(req.Header); err != nil {
		return "", err
	}
	hc := *c.httpClient()
	hc.Timeout = 3 * time.Minute
	cli := &hc
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	text, err := collectOutlineFromSSE(resp.Body)
	if err != nil {
		return "", err
	}
	return text, nil
}

func collectOutlineFromSSE(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	var parts []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		parsed := gjson.Parse(data)
		s := parsed.Get("content").String()
		if s == "" {
			s = parsed.Get("text").String()
		}
		if s != "" {
			parts = append(parts, s)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return strings.Join(parts, ""), nil
}

func (c *Client) triggerContent(taskID, templateID string) (string, error) {
	u := c.base() + "/api/ai/chat/v2/content?task_id=" + url.QueryEscape(taskID)
	if templateID != "" {
		u += "&template_id=" + url.QueryEscape(templateID)
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	if err = c.setAuth(req.Header); err != nil {
		return "", err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err = checkAipptCode(b, "content"); err != nil {
		return "", err
	}
	ticket := gjson.GetBytes(b, "data").String()
	if ticket == "" {
		ticket = gjson.GetBytes(b, "data").Raw
	}
	ticket = strings.Trim(ticket, `"`)
	if ticket == "" {
		return "", fmt.Errorf("aippt: empty content ticket: %s", string(b))
	}
	return ticket, nil
}

func (c *Client) waitContent(ticket string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		u := c.base() + "/api/ai/chat/v2/content/check?ticket=" + url.QueryEscape(ticket)
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		if err = c.setAuth(req.Header); err != nil {
			return err
		}
		resp, err := c.httpClient().Do(req)
		if err != nil {
			return err
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		st := gjson.GetBytes(b, "data.status").String()
		if st == "2" {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("aippt: wait content timeout")
}

func (c *Client) outlineSaveFromPptData(taskID string) error {
	u := c.base() + "/api/generate/data"
	vals := url.Values{}
	vals.Set("task_id", taskID)
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(vals.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err = c.setAuth(req.Header); err != nil {
		return err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err = checkAipptCode(b, "ppt data"); err != nil {
		return err
	}
	contentJSON := gjson.GetBytes(b, "data").Raw
	if contentJSON == "" {
		return fmt.Errorf("aippt: empty ppt data")
	}
	vals2 := url.Values{}
	vals2.Set("task_id", taskID)
	vals2.Set("content", contentJSON)
	req2, err := http.NewRequest(http.MethodPost, c.base()+"/api/ai/chat/v2/outline/save", strings.NewReader(vals2.Encode()))
	if err != nil {
		return err
	}
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err = c.setAuth(req2.Header); err != nil {
		return err
	}
	resp2, err := c.httpClient().Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	b2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return err
	}
	return checkAipptCode(b2, "outline save")
}

func (c *Client) pickRandomTemplateID(taskID string) (string, error) {
	u := c.base() + "/api/template_component/suit/search?page=1&size=20&task_id=" + url.QueryEscape(taskID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	if err = c.setAuth(req.Header); err != nil {
		return "", err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err = checkAipptCode(b, "templates"); err != nil {
		return "", err
	}
	items := gjson.GetBytes(b, "data.list").Array()
	if len(items) == 0 {
		return "", fmt.Errorf("aippt: no templates")
	}
	picked := items[rand.Intn(len(items))].Get("id")
	return strings.TrimSpace(picked.String()), nil
}

func (c *Client) savePPT(taskID, templateID, name, templateType string) (string, error) {
	vals := url.Values{}
	vals.Set("task_id", taskID)
	vals.Set("template_id", templateID)
	if name != "" {
		vals.Set("name", name)
	}
	if templateType != "" {
		vals.Set("template_type", templateType)
	}
	req, err := http.NewRequest(http.MethodPost, c.base()+"/api/design/v2/save", strings.NewReader(vals.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err = c.setAuth(req.Header); err != nil {
		return "", err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err = checkAipptCode(b, "save"); err != nil {
		return "", err
	}
	return strings.TrimSpace(gjson.GetBytes(b, "data.id").String()), nil
}

func (c *Client) exportPPTXWithRetry(designID string) (string, error) {
	var lastErr error
	for retry := 0; retry < 6; retry++ {
		dl, err := c.exportOneRound(designID, "ppt")
		if err == nil {
			return dl, nil
		}
		lastErr = err
		if strings.Contains(err.Error(), "20003") {
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	return "", lastErr
}

func (c *Client) exportOneRound(designID, format string) (string, error) {
	vals := url.Values{}
	vals.Set("id", designID)
	vals.Set("format", format)
	vals.Set("edit", "true")
	vals.Set("files_to_zip", "false")
	req, err := http.NewRequest(http.MethodPost, c.base()+"/api/download/export/file", strings.NewReader(vals.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err = c.setAuth(req.Header); err != nil {
		return "", err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	code := gjson.GetBytes(b, "code")
	if code.Int() == 20003 {
		return "", fmt.Errorf("aippt export queue: code 20003")
	}
	if code.Int() != 0 {
		return "", fmt.Errorf("aippt export: %s (code %s)", gjson.GetBytes(b, "msg").String(), code.String())
	}
	taskKey := gjson.GetBytes(b, "data").String()
	if taskKey == "" {
		taskKey = strings.Trim(gjson.GetBytes(b, "data").Raw, `"`)
	}
	if taskKey == "" {
		return "", fmt.Errorf("aippt: empty export task key: %s", string(b))
	}
	return c.waitDownloadURL(taskKey, 3*time.Minute)
}

func (c *Client) waitDownloadURL(taskKey string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		vals := url.Values{}
		vals.Set("task_key", taskKey)
		req, err := http.NewRequest(http.MethodPost, c.base()+"/api/download/export/file/result", strings.NewReader(vals.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if err = c.setAuth(req.Header); err != nil {
			return "", err
		}
		resp, err := c.httpClient().Do(req)
		if err != nil {
			return "", err
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		code := gjson.GetBytes(b, "code").Int()
		if code == 20003 {
			time.Sleep(5 * time.Second)
			continue
		}
		if code != 0 {
			return "", fmt.Errorf("aippt export result: %s", gjson.GetBytes(b, "msg").String())
		}
		data := gjson.GetBytes(b, "data")
		if data.IsArray() && len(data.Array()) > 0 {
			s := data.Array()[0].String()
			if strings.HasPrefix(s, "http") {
				return s, nil
			}
		}
		s := data.String()
		if strings.HasPrefix(s, "http") {
			return s, nil
		}
		if strings.HasPrefix(data.Raw, `"http`) {
			return strings.Trim(data.String(), `"`), nil
		}
		time.Sleep(3 * time.Second)
	}
	return "", fmt.Errorf("aippt: wait download url timeout")
}

func checkAipptCode(body []byte, ctx string) error {
	co := gjson.GetBytes(body, "code")
	if !co.Exists() {
		return fmt.Errorf("aippt %s: missing code in response", ctx)
	}
	if co.Int() == 0 || strings.TrimSpace(co.String()) == "0" {
		return nil
	}
	msg := gjson.GetBytes(body, "msg").String()
	return fmt.Errorf("aippt %s: code=%s msg=%s", ctx, co.String(), msg)
}

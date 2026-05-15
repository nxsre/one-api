package controller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/service"
)

func copilotSSEFlush(c *gin.Context) {
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

func copilotWriteData(c *gin.Context, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
	copilotSSEFlush(c)
	return err
}

func copilotWriteErrorData(c *gin.Context, msg string) {
	_, _ = c.Writer.Write([]byte("event:error\n"))
	b, _ := json.Marshal(gin.H{"code": 400, "message": msg, "explanation": msg})
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
	copilotSSEFlush(c)
}

func NacosConsoleCopilotConfigGet(c *gin.Context) {
	cfg := service.NacosCopilotEffectiveConfig()
	nacosV3OK(c, gin.H{
		"enabled": cfg.Enabled,
		"apiKey":  cfg.ApiKey,
		"model":   cfg.Model,
		"baseUrl": cfg.BaseURL,
	})
}

func NacosConsoleCopilotConfigPost(c *gin.Context) {
	var body service.NacosCopilotConfig
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	if err := service.NacosCopilotSaveConfig(body); err != nil {
		nacosV3Err(c, 400, err.Error())
		return
	}
	nacosV3OK(c, true)
}

func copilotOpenAIStream(c *gin.Context, cfg service.NacosCopilotConfig, system, user string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	copilotSSEFlush(c)

	if strings.TrimSpace(cfg.ApiKey) == "" {
		_ = copilotWriteData(c, gin.H{"type": "THINKING", "chunk": "未配置 Copilot API Key。"})
		_ = copilotWriteData(c, gin.H{"type": "DONE", "done": true, "explanation": "请在设置中保存 API Key，或配置环境变量 NACOS_COPILOT_API_KEY。"})
		return
	}

	payload := map[string]interface{}{
		"model":  cfg.Model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	jb, err := json.Marshal(payload)
	if err != nil {
		copilotWriteErrorData(c, err.Error())
		return
	}
	url := strings.TrimSuffix(cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(jb))
	if err != nil {
		copilotWriteErrorData(c, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.ApiKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		copilotWriteErrorData(c, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		rb, _ := io.ReadAll(resp.Body)
		copilotWriteErrorData(c, fmt.Sprintf("模型 HTTP %d: %s", resp.StatusCode, string(rb)))
		return
	}

	_ = copilotWriteData(c, gin.H{"type": "THINKING", "chunk": "请求模型中…"})
	br := bufio.NewReader(resp.Body)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				copilotWriteErrorData(c, err.Error())
			}
			break
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}
		var wrap struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &wrap) != nil {
			continue
		}
		if wrap.Error != nil && wrap.Error.Message != "" {
			copilotWriteErrorData(c, wrap.Error.Message)
			return
		}
		if len(wrap.Choices) == 0 {
			continue
		}
		piece := wrap.Choices[0].Delta.Content
		if piece != "" {
			_ = copilotWriteData(c, gin.H{"type": "CONTENT", "chunk": piece})
		}
	}
	_ = copilotWriteData(c, gin.H{"type": "DONE", "done": true})
}

// NacosCopilotSkillGenerate POST /v3/console/copilot/skill/generate
func NacosCopilotSkillGenerate(c *gin.Context) {
	var body struct {
		BackgroundInfo string `json:"backgroundInfo"`
	}
	_ = json.NewDecoder(c.Request.Body).Decode(&body)
	sys := `你是 Nacos 控制台 Copilot。根据用户背景生成 Skill，只输出一个 JSON 对象：顶层键 "skill"，值为 { "name", "description", "skillMd", "resource" }。skillMd 为 Markdown 技能说明正文；resource 可为 {}。不要输出 JSON 以外的文字。`
	user := strings.TrimSpace(body.BackgroundInfo)
	if user == "" {
		user = "（无背景，请给一个简短可用的示例 Skill）"
	}
	copilotOpenAIStream(c, service.NacosCopilotEffectiveConfig(), sys, user)
}

// NacosCopilotSkillOptimize POST /v3/console/copilot/skill/optimize
func NacosCopilotSkillOptimize(c *gin.Context) {
	var body map[string]interface{}
	_ = json.NewDecoder(c.Request.Body).Decode(&body)
	raw, _ := json.Marshal(body)
	sys := `你是 Nacos Skill 优化助手。阅读请求 JSON（含 skill、targetFileName、optimizationGoal 等），输出一个 JSON：顶层键 "optimizedSkill"，结构与 skill 相同（name, description, skillMd, resource）。只输出 JSON，不要其它说明。`
	user := string(raw)
	copilotOpenAIStream(c, service.NacosCopilotEffectiveConfig(), sys, user)
}

// NacosCopilotPromptOptimize POST /v3/console/copilot/prompt/optimize
func NacosCopilotPromptOptimize(c *gin.Context) {
	var body struct {
		Prompt           string `json:"prompt"`
		OptimizationGoal string `json:"optimizationGoal"`
	}
	_ = json.NewDecoder(c.Request.Body).Decode(&body)
	sys := `你是 Prompt 模板优化助手。根据用户给出的模板与优化目标，输出改进后的完整模板文本，可放在 JSON 的 "content" 字段中；或直接输出合法 JSON：{ "content": "..." }。不要输出与模板无关的寒暄。`
	user := fmt.Sprintf("模板:\n%s\n\n优化目标:\n%s", strings.TrimSpace(body.Prompt), strings.TrimSpace(body.OptimizationGoal))
	copilotOpenAIStream(c, service.NacosCopilotEffectiveConfig(), sys, user)
}

// NacosCopilotPromptDebug POST /v3/console/copilot/prompt/debug
func NacosCopilotPromptDebug(c *gin.Context) {
	var body struct {
		Prompt    string `json:"prompt"`
		UserInput string `json:"userInput"`
	}
	_ = json.NewDecoder(c.Request.Body).Decode(&body)
	sys := `你是 Prompt 调试助手。根据已渲染的 prompt 与用户输入，给出简明分析与改进建议，用中文回答，可分小节。`
	user := fmt.Sprintf("已渲染 Prompt:\n%s\n\n用户输入:\n%s", strings.TrimSpace(body.Prompt), strings.TrimSpace(body.UserInput))
	copilotOpenAIStream(c, service.NacosCopilotEffectiveConfig(), sys, user)
}

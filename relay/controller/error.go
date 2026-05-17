package controller

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type GeneralErrorResponse struct {
	Error    model.Error `json:"error"`
	Message  string      `json:"message"`
	Msg      string      `json:"msg"`
	Err      string      `json:"err"`
	ErrorMsg string      `json:"error_msg"`
	Header   struct {
		Message string `json:"message"`
	} `json:"header"`
	Response struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
}

func (e GeneralErrorResponse) ToMessage() string {
	if e.Error.Message != "" {
		return e.Error.Message
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != "" {
		return e.Err
	}
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	if e.Header.Message != "" {
		return e.Header.Message
	}
	if e.Response.Error.Message != "" {
		return e.Response.Error.Message
	}
	return ""
}

// ApplyRelayStatusCodeMapping 按渠道 status_code_mapping 重写返回给客户端的 HTTP 状态码。
func ApplyRelayStatusCodeMapping(c *gin.Context, errWith *model.ErrorWithStatusCode) {
	if errWith == nil || c == nil {
		return
	}
	raw := c.GetString(ctxkey.ChannelStatusCodeMapping)
	if raw == "" || raw == "{}" {
		return
	}
	statusMapping := make(map[string]interface{})
	if json.Unmarshal([]byte(raw), &statusMapping) != nil {
		return
	}
	if errWith.StatusCode == http.StatusOK {
		return
	}
	codeStr := strconv.Itoa(errWith.StatusCode)
	if v, ok := statusMapping[codeStr]; ok {
		if n, ok2 := parseStatusMappingValue(v); ok2 {
			errWith.StatusCode = n
		}
	}
}

func parseStatusMappingValue(v interface{}) (int, bool) {
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return 0, false
		}
		n, err := strconv.Atoi(strings.TrimSpace(t))
		return n, err == nil
	case float64:
		return int(t), true
	case json.Number:
		n, err := strconv.Atoi(t.String())
		return n, err == nil
	default:
		return 0, false
	}
}

func RelayErrorHandler(resp *http.Response) (ErrorWithStatusCode *model.ErrorWithStatusCode) {
	if resp == nil {
		return &model.ErrorWithStatusCode{
			StatusCode: 500,
			Error: model.Error{
				Message: "resp is nil",
				Type:    "upstream_error",
				Code:    "bad_response",
			},
		}
	}
	ErrorWithStatusCode = &model.ErrorWithStatusCode{
		StatusCode: resp.StatusCode,
		Error: model.Error{
			Message: "",
			Type:    "upstream_error",
			Code:    "bad_response_status_code",
			Param:   strconv.Itoa(resp.StatusCode),
		},
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if config.DebugEnabled {
		logger.SysLog(fmt.Sprintf("error happened, status code: %d, response: \n%s", resp.StatusCode, string(responseBody)))
	}
	err = resp.Body.Close()
	if err != nil {
		return
	}
	var errResponse GeneralErrorResponse
	err = json.Unmarshal(responseBody, &errResponse)
	if err != nil {
		return
	}
	if errResponse.Error.Message != "" {
		// OpenAI format error, so we override the default one
		ErrorWithStatusCode.Error = errResponse.Error
	} else {
		ErrorWithStatusCode.Error.Message = errResponse.ToMessage()
	}
	if ErrorWithStatusCode.Error.Message == "" {
		ErrorWithStatusCode.Error.Message = fmt.Sprintf("bad response status code %d", resp.StatusCode)
	}
	return
}

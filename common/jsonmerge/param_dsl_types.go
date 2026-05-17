package jsonmerge

import "errors"

// 与主流渠道「operations」覆盖 DSL 对齐：tidwall/gjson + sjson 路径语法、条件与多种 mode。

const (
	CtxRequestHeaders = "request_headers"
	CtxHeaderOverride = "header_override"
)

var errSourceHeaderNotFound = errors.New("source header does not exist")

// ConditionOperation 单条条件（gjson path）。
type ConditionOperation struct {
	Path           string      `json:"path"`
	Mode           string      `json:"mode"`
	Value          interface{} `json:"value"`
	Invert         bool        `json:"invert"`
	PassMissingKey bool        `json:"pass_missing_key"`
}

// ParamOperation 单条操作。
type ParamOperation struct {
	Path       string               `json:"path"`
	Mode       string               `json:"mode"`
	Value      interface{}          `json:"value"`
	KeepOrigin bool                 `json:"keep_origin"`
	From       string               `json:"from,omitempty"`
	To         string               `json:"to,omitempty"`
	Conditions []ConditionOperation `json:"conditions,omitempty"`
	Logic      string               `json:"logic,omitempty"`
}

// ReturnError 对应 mode return_error：由调用方转换为 HTTP 错误。
type ReturnError struct {
	Message    string
	StatusCode int
	Code       string
	Type       string
	SkipRetry  bool
}

func (e *ReturnError) Error() string {
	if e == nil {
		return "param override return_error"
	}
	if e.Message != "" {
		return e.Message
	}
	return "param override return_error"
}

// AsReturnError 用于 errors.As。
func AsReturnError(err error) (*ReturnError, bool) {
	var re *ReturnError
	if err != nil && errors.As(err, &re) {
		return re, true
	}
	return nil, false
}

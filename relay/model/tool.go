package model

type Tool struct {
	Index    int    `json:"index,omitempty"` // OpenAI 流式 chunk 中 tool_calls 的下标
	Id       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"` // when splicing claude tools stream messages, it is empty
	Function Function `json:"function"`
}

type Function struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`       // when splicing claude tools stream messages, it is empty
	Parameters  any    `json:"parameters,omitempty"` // request
	Arguments   any    `json:"arguments,omitempty"`  // response
}

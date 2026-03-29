package llm

import "context"

// Message 对齐 OpenAI Chat Completions 消息：user/assistant/system 使用 Content；
// assistant 可带 ToolCalls；role=tool 时使用 ToolCallID + Content（工具结果文本）。
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 表示 assistant 发起的单次函数调用（OpenAI 兼容格式）。
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 为 ToolCall 中的函数名与 JSON 参数字符串。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition 为 Chat Completions 的 tools[] 项（OpenAI 兼容）。
type ToolDefinition struct {
	Type     string         `json:"type"`
	Function FunctionSchema `json:"function"`
}

// FunctionSchema 描述可调用的函数及其 JSON Schema parameters。
type FunctionSchema struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
}

type Request struct {
	Model    string
	Messages []Message
	Tools    []ToolDefinition
}

// Usage 为单次补全的 token 用量（若服务端未返回则为零值）。
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Response struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
}

// StreamChunk 表示流式响应中的一条：Text 为增量文本；Err 非空表示流失败（含提前取消）。
type StreamChunk struct {
	Text string
	Err  error
}

type Client interface {
	Generate(ctx context.Context, req Request) (Response, error)
	// GenerateStream 以流式方式请求补全，通过只读 channel 逐段返回增量；channel 关闭表示正常结束。
	GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, error)
}

// StreamingClient 可选能力：流式请求并在服务端聚合为与非流式 Generate 等价的 Response；
// onContent 收到正文增量（不含 tool_calls 拼接逻辑）。未实现的 Provider 仍走 Generate。
type StreamingClient interface {
	GenerateStreaming(ctx context.Context, req Request, onContent func(delta string) error) (Response, error)
}

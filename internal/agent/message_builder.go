package agent

import "simple-agent/internal/llm"

// MessageBuilder 负责构造 llm.Message 片段（无状态，可安全复用）。
type MessageBuilder struct{}

// RequestMessages 组装发往 LLM 的完整请求：可选 system + 已持久化的消息 + 本轮 user。
func (MessageBuilder) RequestMessages(stored []llm.Message, system, userContent string) []llm.Message {
	var out []llm.Message
	if system != "" {
		out = append(out, llm.Message{Role: "system", Content: system})
	}
	out = append(out, stored...)
	out = append(out, llm.Message{Role: "user", Content: userContent})
	return out
}

// ToolInvocationTurn 返回与 OpenAI 兼容的一轮 tool 调用：user → assistant(tool_calls) → tool。
func (MessageBuilder) ToolInvocationTurn(userContent, toolCallID, toolName, argumentsJSON, toolResult string) []llm.Message {
	return []llm.Message{
		{Role: "user", Content: userContent},
		{
			Role: "assistant",
			ToolCalls: []llm.ToolCall{
				{
					ID:   toolCallID,
					Type: "function",
					Function: llm.FunctionCall{
						Name:      toolName,
						Arguments: argumentsJSON,
					},
				},
			},
		},
		{
			Role:       "tool",
			ToolCallID: toolCallID,
			Content:    toolResult,
		},
	}
}

// AssistantWithToolCalls 构造含 tool_calls 的 assistant 消息。
func (MessageBuilder) AssistantWithToolCalls(calls []llm.ToolCall) llm.Message {
	return llm.Message{Role: "assistant", ToolCalls: calls}
}

// ToolResult 构造 role=tool 的消息。
func (MessageBuilder) ToolResult(toolCallID, content string) llm.Message {
	return llm.Message{Role: "tool", ToolCallID: toolCallID, Content: content}
}

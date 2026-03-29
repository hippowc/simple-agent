package llm

import "unicode/utf8"

// EstimateTokens 粗算字符串 token 数（约 4 字符 ≈ 1 token），用于 API 未返回 usage 时的兜底。
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return (utf8.RuneCountInString(s) + 3) / 4
}

// EstimateMessagesTokens 估算 messages 总 token（请求侧上下文）。
func EstimateMessagesTokens(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += EstimateTokens(m.Role)
		n += EstimateTokens(m.Content)
		n += EstimateTokens(m.ToolCallID)
		for _, tc := range m.ToolCalls {
			n += EstimateTokens(tc.ID)
			n += EstimateTokens(tc.Type)
			n += EstimateTokens(tc.Function.Name)
			n += EstimateTokens(tc.Function.Arguments)
		}
	}
	return n
}

// EstimateResponseOutputTokens 估算本轮 assistant 输出 token（正文 + tool_calls）。
func EstimateResponseOutputTokens(r *Response) int {
	if r == nil {
		return 0
	}
	n := EstimateTokens(r.Content)
	for _, tc := range r.ToolCalls {
		n += EstimateTokens(tc.Function.Name) + EstimateTokens(tc.Function.Arguments)
	}
	return n
}

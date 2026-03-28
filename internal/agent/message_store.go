package agent

import "simple-agent/internal/llm"

// MessageStore 持久化多轮对话消息（user / assistant / tool），供下次请求与提交合并。
// 与 turnTranscript 的关系：RequestMessages 产出回合初始切片；成功结束时 AppendFrom 将 transcript 中从 commitFrom 起的后缀并入本存储。
type MessageStore struct {
	messages []llm.Message
	builder  MessageBuilder
}

// NewMessageStore 创建空的存储。
func NewMessageStore() *MessageStore {
	return &MessageStore{}
}

// RequestMessages 基于当前已存消息，组装本次发往 LLM 的请求（含 system 与本轮 user）。
func (s *MessageStore) RequestMessages(system, userContent string) []llm.Message {
	return s.builder.RequestMessages(s.messages, system, userContent)
}

// RecordToolInvocation 追加一轮标准工具调用消息（与 LLM tool_calls 协议一致）。
func (s *MessageStore) RecordToolInvocation(userContent, toolCallID, toolName, argumentsJSON, toolResult string) {
	s.messages = append(s.messages, s.builder.ToolInvocationTurn(userContent, toolCallID, toolName, argumentsJSON, toolResult)...)
}

// AppendFrom 将完整请求切片 msgs 中从 from 起的后缀合并进存储（通常 from 指向本轮 user）。
func (s *MessageStore) AppendFrom(msgs []llm.Message, from int) {
	if from < 0 || from >= len(msgs) {
		return
	}
	s.messages = append(s.messages, msgs[from:]...)
}

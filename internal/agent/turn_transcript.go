package agent

import "simple-agent/internal/llm"

// turnTranscript 是单用户回合内的可变消息列表（发往 LLM），以及成功落库时并入 MessageStore 的起始下标（通常指向本轮 user）。
// 持久化历史见 sessionRuntime.store；内层循环状态见 turnExecution.loop。
type turnTranscript struct {
	msgs       []llm.Message
	commitFrom int
	builder    MessageBuilder
}

func newTurnTranscript(msgs []llm.Message) *turnTranscript {
	commitFrom := 0
	if len(msgs) > 0 {
		commitFrom = len(msgs) - 1
	}
	return &turnTranscript{msgs: msgs, commitFrom: commitFrom}
}

func (t *turnTranscript) appendAssistantToolCalls(calls []llm.ToolCall) {
	t.msgs = append(t.msgs, t.builder.AssistantWithToolCalls(calls))
}

func (t *turnTranscript) appendToolMessage(msg llm.Message) {
	t.msgs = append(t.msgs, msg)
}

package agent

import "simple-agent/internal/llm"

// turnRuntime 单用户回合：一条用户输入对应的发往 LLM 的可变 transcript、工具定义与会话句柄。
type turnRuntime struct {
	session    *sessionRuntime
	transcript *turnTranscript
	toolDefs   []llm.ToolDefinition
}

func newTurnRuntime(sess *sessionRuntime, userInput string, toolDefs []llm.ToolDefinition) *turnRuntime {
	msgs := sess.store.RequestMessages(sess.systemPrompt, userInput)
	return &turnRuntime{
		session:    sess,
		transcript: newTurnTranscript(msgs),
		toolDefs:   toolDefs,
	}
}

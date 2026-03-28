package agent

import "simple-agent/internal/llm"

// agentLoopState 单回合内的 agent 内层循环状态（LLM ↔ tool 多步）：供策略选择与节点接力，不存放对话正文。
type agentLoopState struct {
	lastLLMResp  *llm.Response
	llmCallCount int
	maxLLMCalls  int
}

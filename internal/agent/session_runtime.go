package agent

import (
	"simple-agent/internal/llm"
	"simple-agent/internal/tools"
)

// sessionRuntime 会话级运行时：在 Agent 生命周期内跨多轮用户输入共享（存储、模型、工具、系统提示）。
type sessionRuntime struct {
	store        *MessageStore
	llmClient    llm.Client
	model        string
	systemPrompt string
	tools        *tools.Registry
	streamLLM    bool // 为 true 且 Client 实现 StreamingClient 时使用流式聚合

	contextWindowTokens     int   // 来自配置，0 表示不在 UI 显示上下文百分比
	cumulativeSessionTokens int64 // 本会话累计 token（优先使用 API total_tokens，否则估算）
}

func newSessionRuntime(store *MessageStore, client llm.Client, model, systemPrompt string, reg *tools.Registry, streamLLM bool, contextWindowTokens int) *sessionRuntime {
	return &sessionRuntime{
		store:               store,
		llmClient:           client,
		model:               model,
		systemPrompt:        systemPrompt,
		tools:               reg,
		streamLLM:           streamLLM,
		contextWindowTokens: contextWindowTokens,
	}
}

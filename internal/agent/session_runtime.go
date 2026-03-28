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
}

func newSessionRuntime(store *MessageStore, client llm.Client, model, systemPrompt string, reg *tools.Registry) *sessionRuntime {
	return &sessionRuntime{
		store:        store,
		llmClient:    client,
		model:        model,
		systemPrompt: systemPrompt,
		tools:        reg,
	}
}

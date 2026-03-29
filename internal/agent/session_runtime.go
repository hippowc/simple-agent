package agent

import (
	"simple-agent/internal/llm"
	"simple-agent/internal/tools"
)

// sessionRuntime 会话级运行时：在 Agent 生命周期内跨多轮用户输入共享（存储、模型、工具、系统提示）。
type sessionRuntime struct {
	store              *MessageStore
	llmClient          llm.Client
	model              string
	systemPrompt       string
	userPromptTemplate string // 来自配置 prompt.user_prompt；空表示直接使用用户输入
	tools              *tools.Registry
	streamLLM          bool // 为 true 且 Client 实现 StreamingClient 时使用流式聚合

	contextWindowTokens     int   // 来自配置，0 表示不在 UI 显示上下文百分比
	cumulativeSessionTokens int64 // 本会话累计 token（优先使用 API total_tokens，否则估算）
}

func newSessionRuntime(store *MessageStore, client llm.Client, model, systemPrompt, userPromptTemplate string, reg *tools.Registry, streamLLM bool, contextWindowTokens int) *sessionRuntime {
	return &sessionRuntime{
		store:               store,
		llmClient:           client,
		model:               model,
		systemPrompt:        systemPrompt,
		userPromptTemplate:  userPromptTemplate,
		tools:               reg,
		streamLLM:           streamLLM,
		contextWindowTokens: contextWindowTokens,
	}
}

func (s *sessionRuntime) replaceLLM(client llm.Client, model string, streamLLM bool, contextWindowTokens int) {
	s.llmClient = client
	s.model = model
	s.streamLLM = streamLLM
	s.contextWindowTokens = contextWindowTokens
}

func (s *sessionRuntime) setPrompts(systemPrompt, userPromptTemplate string) {
	s.systemPrompt = systemPrompt
	s.userPromptTemplate = userPromptTemplate
}

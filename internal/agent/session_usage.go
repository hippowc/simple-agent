package agent

import "simple-agent/internal/llm"

// usageEventAfterLLM 根据 API 用量与兜底估算，更新会话累计 token 并生成 UI 事件。
func (s *sessionRuntime) usageEventAfterLLM(u llm.Usage, msgs []llm.Message, resp *llm.Response) AgentEvent {
	var add int64
	switch {
	case u.TotalTokens > 0:
		add = int64(u.TotalTokens)
	case u.PromptTokens > 0 || u.CompletionTokens > 0:
		add = int64(u.PromptTokens + u.CompletionTokens)
	default:
		add = int64(llm.EstimateMessagesTokens(msgs) + llm.EstimateResponseOutputTokens(resp))
	}
	s.cumulativeSessionTokens += add

	displayPrompt := u.PromptTokens
	if displayPrompt == 0 {
		displayPrompt = llm.EstimateMessagesTokens(msgs)
	}
	displayCompletion := u.CompletionTokens
	if displayCompletion == 0 {
		displayCompletion = llm.EstimateResponseOutputTokens(resp)
	}

	pct := -1.0
	if s.contextWindowTokens > 0 && displayPrompt > 0 {
		pct = float64(displayPrompt) / float64(s.contextWindowTokens) * 100
		if pct > 100 {
			pct = 100
		}
	}

	return AgentEvent{
		Kind:                 EventKindUsage,
		SessionTokenTotal:    s.cumulativeSessionTokens,
		LastPromptTokens:     displayPrompt,
		LastCompletionTokens: displayCompletion,
		ContextPercent:       pct,
	}
}

package agent

import "strings"

// flowStep 表示内层循环下一步：由 runAgentLoop 的 switch 分派到对应 node.Run。
type flowStep int

const (
	stepLLM flowStep = iota
	stepToolCalls
	stepAbortEmpty
	stepFinalText
	stepMaxLLMCalls
)

// agentLoopPolicy 根据 agentLoopState 决定下一步，无状态、无副作用（与 ReAct / tool loop 调度一致）。
type agentLoopPolicy struct{}

// Next：loop.lastLLMResp 为 nil 时尚无可路由的 LLM 输出，应进入 LLM；否则按响应内容分支。
func (agentLoopPolicy) Next(exec *turnExecution) flowStep {
	s := &exec.loop
	if s.lastLLMResp != nil {
		if len(s.lastLLMResp.ToolCalls) > 0 {
			return stepToolCalls
		}
		if strings.TrimSpace(s.lastLLMResp.Content) == "" && s.lastLLMResp.FinishReason == "stop" {
			return stepAbortEmpty
		}
		return stepFinalText
	}
	if s.llmCallCount >= s.maxLLMCalls {
		return stepMaxLLMCalls
	}
	return stepLLM
}

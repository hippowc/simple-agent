package agent

import (
	"context"
	"fmt"

	"simple-agent/internal/tools"
)

const maxLLMCallsPerTurn = 16

// turnTerminator 负责本轮成功结束或错误退出（事件与 MessageStore）。
type turnTerminator struct{}

func (turnTerminator) successWithText(ctx context.Context, exec *turnExecution) {
	sendAgentEvent(ctx, exec.out, AgentEvent{Kind: EventKindLLM, Text: exec.loop.lastLLMResp.Content, Partial: false})
	exec.turn.session.store.AppendFrom(exec.turn.transcript.msgs, exec.turn.transcript.commitFrom)
}

func (turnTerminator) error(ctx context.Context, exec *turnExecution, detail string) {
	sendAgentEvent(ctx, exec.out, AgentEvent{Kind: EventKindError, Detail: detail})
}

func (turnTerminator) maxLLMCallsExceeded(ctx context.Context, exec *turnExecution) {
	sendAgentEvent(ctx, exec.out, AgentEvent{Kind: EventKindError, Detail: fmt.Sprintf("exceeded max LLM calls per turn (%d)", exec.loop.maxLLMCalls)})
}

// runAgentLoop：循环「agentLoopPolicy → switch → node.Run」；llmNode / toolNode 互不引用，仅由本流程串联。
func (a *Agent) runAgentLoop(ctx context.Context, userInput string, out chan<- AgentEvent) {
	turn := newTurnRuntime(a.session, userInput, tools.OpenAIToolDefinitions())
	exec := &turnExecution{
		turn: turn,
		out:  out,
		loop: agentLoopState{
			maxLLMCalls: maxLLMCallsPerTurn,
		},
	}
	s := exec.turn.session
	llmN := newLLMNode(s.llmClient, s.model)
	toolN := &toolNode{reg: s.tools}
	var policy agentLoopPolicy
	var term turnTerminator

	for {
		switch policy.Next(exec) {
		case stepLLM:
			if err := llmN.Run(ctx, exec); err != nil {
				term.error(ctx, exec, err.Error())
				return
			}
		case stepToolCalls:
			if err := toolN.Run(ctx, exec); err != nil {
				term.error(ctx, exec, err.Error())
				return
			}
		case stepAbortEmpty:
			term.error(ctx, exec, "model returned empty content")
			return
		case stepFinalText:
			term.successWithText(ctx, exec)
			return
		case stepMaxLLMCalls:
			term.maxLLMCallsExceeded(ctx, exec)
			return
		}
	}
}

package agent

import (
	"context"

	"simple-agent/internal/llm"
)

// llmNode 只通过 Run 对外暴露。
type llmNode struct {
	client llm.Client
	model  string
}

func newLLMNode(client llm.Client, model string) *llmNode {
	return &llmNode{client: client, model: model}
}

func (n *llmNode) Run(ctx context.Context, exec *turnExecution) error {
	exec.loop.llmCallCount++
	exec.loop.lastLLMStreamed = false
	req := llm.Request{
		Model:    n.model,
		Messages: exec.turn.transcript.msgs,
		Tools:    exec.turn.toolDefs,
	}
	if exec.turn.session.streamLLM {
		if sc, ok := n.client.(llm.StreamingClient); ok {
			resp, err := sc.GenerateStreaming(ctx, req, func(delta string) error {
				if delta == "" {
					return nil
				}
				sendAgentEvent(ctx, exec.out, AgentEvent{Kind: EventKindLLM, Text: delta, Partial: true})
				return nil
			})
			if err != nil {
				return err
			}
			exec.loop.lastLLMResp = &resp
			exec.loop.lastLLMStreamed = true
			sendAgentEvent(ctx, exec.out, exec.turn.session.usageEventAfterLLM(resp.Usage, exec.turn.transcript.msgs, &resp))
			return nil
		}
	}
	resp, err := n.client.Generate(ctx, req)
	if err != nil {
		return err
	}
	exec.loop.lastLLMResp = &resp
	sendAgentEvent(ctx, exec.out, exec.turn.session.usageEventAfterLLM(resp.Usage, exec.turn.transcript.msgs, &resp))
	return nil
}

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
	resp, err := n.client.Generate(ctx, llm.Request{
		Model:    n.model,
		Messages: exec.turn.transcript.msgs,
		Tools:    exec.turn.toolDefs,
	})
	if err != nil {
		return err
	}
	exec.loop.lastLLMResp = &resp
	return nil
}

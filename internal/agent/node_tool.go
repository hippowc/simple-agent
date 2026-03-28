package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"simple-agent/internal/llm"
	"simple-agent/internal/tools"
)

// toolNode 只通过 Run 对外暴露。
type toolNode struct {
	reg     *tools.Registry
	builder MessageBuilder
}

func (n *toolNode) Run(ctx context.Context, exec *turnExecution) error {
	calls := exec.loop.lastLLMResp.ToolCalls
	exec.turn.transcript.appendAssistantToolCalls(calls)
	for _, call := range calls {
		res := n.runOne(ctx, call)
		exec.turn.transcript.appendToolMessage(res.ToolMessage)
		sendAgentEvent(ctx, exec.out, AgentEvent{Kind: EventKindTool, ToolName: res.Name, Detail: res.Detail})
	}
	exec.loop.lastLLMResp = nil
	return nil
}

type toolExecutionResult struct {
	ToolMessage llm.Message
	Detail      string
	Name        string
}

func (n *toolNode) runOne(ctx context.Context, tc llm.ToolCall) toolExecutionResult {
	name := tc.Function.Name
	args, err := parseToolArgumentsJSON(tc.Function.Arguments)
	var detail string
	if err != nil {
		detail = "invalid arguments: " + err.Error()
		return toolExecutionResult{
			Name:        name,
			Detail:      detail,
			ToolMessage: n.builder.ToolResult(tc.ID, detail),
		}
	}
	result, callErr := n.reg.Call(ctx, name, tools.CallInput{Arguments: args})
	if callErr != nil {
		detail = callErr.Error()
	} else {
		detail = result
	}
	return toolExecutionResult{
		Name:        name,
		Detail:      detail,
		ToolMessage: n.builder.ToolResult(tc.ID, detail),
	}
}

func parseToolArgumentsJSON(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for k, v := range m {
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

package agent

import "simple-agent/internal/builtin"

func builtinOutputToAgentEvent(o builtin.Output) AgentEvent {
	switch o.Kind {
	case builtin.KindError:
		return AgentEvent{Kind: EventKindError, Detail: o.Err}
	case builtin.KindTool:
		return AgentEvent{Kind: EventKindTool, ToolName: o.ToolName, Detail: o.Detail}
	case builtin.KindInfo:
		return AgentEvent{Kind: EventKindInfo, Text: o.InfoText}
	default:
		return AgentEvent{Kind: EventKindError, Detail: "unknown builtin output"}
	}
}

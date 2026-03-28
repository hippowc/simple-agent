package ui

import (
	"context"

	"simple-agent/internal/agent"
)

// Agent 与 agent.Agent 对齐，便于测试注入。
type Agent interface {
	RunTurn(ctx context.Context, userInput string) <-chan agent.AgentEvent
}

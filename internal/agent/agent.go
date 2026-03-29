package agent

import (
	"context"
	"strings"

	"simple-agent/internal/builtin"
	"simple-agent/internal/common"
	"simple-agent/internal/llm"
	"simple-agent/internal/tools"
)

// Agent 是运行时实体：持有会话级 sessionRuntime，并对外提供统一入口（如 RunTurn）。
// 后续扩展子 Agent 时，调度逻辑也应集中在此包内。
type Agent struct {
	session *sessionRuntime
}

// NewFromConfig 根据配置创建 LLM 客户端、注册默认工具，并返回可用的 Agent。
func NewFromConfig(cfg common.Config) (*Agent, error) {
	client, err := llm.NewClient(cfg.LLM)
	if err != nil {
		return nil, err
	}
	reg := tools.NewRegistry()
	if err := registerDefaultTools(reg, cfg.Workspace); err != nil {
		return nil, err
	}
	systemPrompt, err := common.LoadSystemPromptAuto()
	if err != nil {
		return nil, err
	}
	store := NewMessageStore()
	streamLLM := true
	if cfg.LLM.Stream != nil {
		streamLLM = *cfg.LLM.Stream
	}
	return &Agent{
		session: newSessionRuntime(store, client, cfg.LLM.Model, systemPrompt, reg, streamLLM, cfg.LLM.ContextWindowTokens),
	}, nil
}

func registerDefaultTools(reg *tools.Registry, workspace string) error {
	if err := reg.Register(tools.NewReadFileTool(workspace)); err != nil {
		return err
	}
	if err := reg.Register(tools.NewWriteFileTool(workspace)); err != nil {
		return err
	}
	if err := reg.Register(tools.NewEditFileTool(workspace)); err != nil {
		return err
	}
	if err := reg.Register(tools.NewFindFilesTool(workspace)); err != nil {
		return err
	}
	if err := reg.Register(tools.NewGrepContentTool(workspace)); err != nil {
		return err
	}
	if err := reg.Register(tools.NewRunShellTool(workspace)); err != nil {
		return err
	}
	return nil
}

// RunTurn 在独立 goroutine 中执行一轮逻辑，通过返回的只读 channel 投递 AgentEvent；发完后关闭 channel。
func (a *Agent) RunTurn(ctx context.Context, userInput string) <-chan AgentEvent {
	out := make(chan AgentEvent, 16)
	go a.runTurnAsync(ctx, strings.TrimSpace(userInput), out)
	return out
}

func (a *Agent) runTurnAsync(ctx context.Context, userInput string, out chan<- AgentEvent) {
	defer close(out)
	a.runTurn(ctx, userInput, out)
}

// sendAgentEvent 将事件写入 out；若 ctx 已取消则返回 false（调用方应停止后续逻辑）。
func sendAgentEvent(ctx context.Context, out chan<- AgentEvent, ev AgentEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- ev:
		return true
	}
}

func (a *Agent) runTurn(ctx context.Context, userInput string, out chan<- AgentEvent) {
	if userInput == "" {
		sendAgentEvent(ctx, out, AgentEvent{Kind: EventKindError, Detail: "empty input"})
		return
	}

	if strings.HasPrefix(userInput, "/") {
		res := builtin.Dispatch(ctx, userInput, builtin.Deps{
			Registry: a.session.tools,
			Store:    a.session.store,
		})
		if res.Handled {
			for _, o := range res.Outputs {
				sendAgentEvent(ctx, out, builtinOutputToAgentEvent(o))
			}
			return
		}
	}

	a.runAgentLoop(ctx, userInput, out)
}

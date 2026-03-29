package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"simple-agent/internal/builtin"
	"simple-agent/internal/common"
	"simple-agent/internal/llm"
	"simple-agent/internal/tools"
)

// Agent 是运行时实体：持有会话级 sessionRuntime，并对外提供统一入口（如 RunTurn）。
// 后续扩展子 Agent 时，调度逻辑也应集中在此包内。
type Agent struct {
	mu         sync.RWMutex
	session    *sessionRuntime
	cfg        common.Config
	configPath string
}

// NewFromConfig 根据配置创建 LLM 客户端、注册默认工具，并返回可用的 Agent。
// configPath 为持久化路径（用户目录或当前目录下的 config.json），供 /model、/prompt 保存；可为空则内置命令无法写盘。
func NewFromConfig(cfg common.Config, configPath string) (*Agent, error) {
	cfgCopy, err := common.CloneConfig(cfg)
	if err != nil {
		return nil, err
	}
	prof, err := cfgCopy.ActiveLLMProfile()
	if err != nil {
		return nil, err
	}
	client, err := llm.NewClient(prof.LLMConfig)
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	reg := tools.NewRegistry()
	if err := registerDefaultTools(reg, wd); err != nil {
		return nil, err
	}
	systemPrompt, err := common.ResolveSystemPrompt(cfgCopy.Prompt)
	if err != nil {
		return nil, err
	}
	userPromptTpl, err := common.ResolveUserPromptTemplate(cfgCopy.Prompt)
	if err != nil {
		return nil, err
	}
	store := NewMessageStore()
	streamLLM := true
	if prof.Stream != nil {
		streamLLM = *prof.Stream
	}
	sess := newSessionRuntime(store, client, prof.Model, systemPrompt, userPromptTpl, reg, streamLLM, prof.ContextWindowTokens)
	return &Agent{
		session:    sess,
		cfg:        cfgCopy,
		configPath: configPath,
	}, nil
}

func (a *Agent) applySessionFromConfigUnlocked(cfg common.Config) error {
	prof, err := cfg.ActiveLLMProfile()
	if err != nil {
		return err
	}
	if strings.TrimSpace(prof.BaseURL) == "" || strings.TrimSpace(prof.APIKey) == "" || strings.TrimSpace(prof.Model) == "" {
		return fmt.Errorf("active profile needs base_url, api_key, and model")
	}
	client, err := llm.NewClient(prof.LLMConfig)
	if err != nil {
		return err
	}
	systemPrompt, err := common.ResolveSystemPrompt(cfg.Prompt)
	if err != nil {
		return err
	}
	userTpl, err := common.ResolveUserPromptTemplate(cfg.Prompt)
	if err != nil {
		return err
	}
	streamLLM := true
	if prof.Stream != nil {
		streamLLM = *prof.Stream
	}
	a.session.replaceLLM(client, prof.Model, streamLLM, prof.ContextWindowTokens)
	a.session.setPrompts(systemPrompt, userTpl)
	return nil
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
		if outs, ok := a.tryConfigSlashCommands(ctx, userInput); ok {
			for _, o := range outs {
				sendAgentEvent(ctx, out, builtinOutputToAgentEvent(o))
			}
			return
		}
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
		sendAgentEvent(ctx, out, AgentEvent{
			Kind:   EventKindError,
			Detail: fmt.Sprintf("unknown command %s — built-ins: /model, /prompt, /tools, /quit. Omit leading / to talk to the model.", slashCommandSummary(userInput)),
		})
		return
	}

	a.runAgentLoop(ctx, userInput, out)
}

func slashCommandSummary(line string) string {
	f := strings.Fields(line)
	if len(f) == 0 {
		return line
	}
	return f[0]
}

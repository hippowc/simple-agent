package builtin

import (
	"context"
	"fmt"
	"strings"
)

// Kind 表示内置命令产生的一条输出类别。
type Kind int

const (
	KindError Kind = iota
	KindTool
	KindInfo
)

// Output 为单条执行结果，供 agent 映射为 AgentEvent。
type Output struct {
	Kind     Kind
	Err      string // KindError
	ToolName string // KindTool
	Detail   string // KindTool
	InfoText string // KindInfo
}

// Outcome 表示是否由内置命令处理，以及需展示给 UI 的输出序列。
type Outcome struct {
	Handled bool
	Outputs []Output
}

// Dispatch 处理以 `/` 开头的内置命令（不含通过 agent 注册的 /model、/prompt）。
// 读/写文件请通过自然语言让模型调用工具，而非伪造成内置命令。
func Dispatch(_ context.Context, input string, deps Deps) Outcome {
	if deps.Registry == nil {
		return Outcome{}
	}

	if input == "/tools" {
		return Outcome{
			Handled: true,
			Outputs: []Output{{
				Kind:     KindInfo,
				InfoText: fmt.Sprintf("tools: %s", strings.Join(deps.Registry.List(), ", ")),
			}},
		}
	}

	return Outcome{}
}

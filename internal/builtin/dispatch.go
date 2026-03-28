package builtin

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"simple-agent/internal/tools"
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

// Dispatch 处理以 `/` 开头的内置命令；若非内置命令或无需处理则 Handled=false。
func Dispatch(ctx context.Context, input string, deps Deps) Outcome {
	if deps.Registry == nil {
		return Outcome{}
	}

	if strings.HasPrefix(input, "/read ") {
		return dispatchRead(ctx, input, deps)
	}
	if strings.HasPrefix(input, "/write ") {
		return dispatchWrite(ctx, input, deps)
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

func dispatchRead(ctx context.Context, input string, deps Deps) Outcome {
	path := strings.TrimSpace(strings.TrimPrefix(input, "/read "))
	result, err := deps.Registry.Call(ctx, "read_file", tools.CallInput{
		Arguments: map[string]string{"path": path},
	})
	if err != nil {
		return Outcome{
			Handled: true,
			Outputs: []Output{{Kind: KindError, Err: err.Error()}},
		}
	}
	argsJSON, _ := json.Marshal(map[string]string{"path": path})
	if deps.Store != nil {
		deps.Store.RecordToolInvocation(input, newToolCallID(), "read_file", string(argsJSON), result)
	}
	return Outcome{
		Handled: true,
		Outputs: []Output{{Kind: KindTool, ToolName: "read_file", Detail: result}},
	}
}

func dispatchWrite(ctx context.Context, input string, deps Deps) Outcome {
	parts := strings.SplitN(strings.TrimPrefix(input, "/write "), " ", 2)
	if len(parts) < 2 {
		return Outcome{
			Handled: true,
			Outputs: []Output{{Kind: KindError, Err: "write format: /write <path> <content>"}},
		}
	}
	path := strings.TrimSpace(parts[0])
	content := parts[1]
	result, err := deps.Registry.Call(ctx, "write_file", tools.CallInput{
		Arguments: map[string]string{
			"path":    path,
			"content": content,
		},
	})
	if err != nil {
		return Outcome{
			Handled: true,
			Outputs: []Output{{Kind: KindError, Err: err.Error()}},
		}
	}
	argsJSON, _ := json.Marshal(map[string]string{"path": path, "content": content})
	if deps.Store != nil {
		deps.Store.RecordToolInvocation(input, newToolCallID(), "write_file", string(argsJSON), result)
	}
	return Outcome{
		Handled: true,
		Outputs: []Output{{Kind: KindTool, ToolName: "write_file", Detail: result}},
	}
}

func newToolCallID() string {
	var b [10]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("call_%x", b)
}

package builtin

import "simple-agent/internal/tools"

// MessageStoreRecorder 用于内置命令成功执行后，将标准 tool 调用写入持久化存储。
type MessageStoreRecorder interface {
	RecordToolInvocation(userContent, toolCallID, toolName, argumentsJSON, toolResult string)
}

// Deps 为内置命令执行所需依赖，由 agent 注入。
type Deps struct {
	Registry *tools.Registry
	Store    MessageStoreRecorder
}

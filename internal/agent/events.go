package agent

// EventKind 区分事件语义，便于 UI 分别渲染；后续可增枚举值而不破坏旧逻辑。
type EventKind string

const (
	EventKindLLM       EventKind = "llm"        // 模型输出（流式时为多次增量）
	EventKindTool      EventKind = "tool"       // 工具执行完成，Detail 为结果
	EventKindToolStart EventKind = "tool_start" // 工具开始执行，仅填 ToolName，供 UI 显示 loading
	EventKindInfo      EventKind = "info"       // 非模型、非工具的提示信息（如 /tools 列表）
	EventKindError     EventKind = "error"      // 本轮内可恢复错误（致命失败仍可用 Detail 描述）
	EventKindUsage     EventKind = "usage"      // token 与上下文占用（见下方字段）
)

// AgentEvent 为 agent → UI 的单条输出，可按 Kind 扩展字段；未用到的字段保持零值。
type AgentEvent struct {
	Kind EventKind

	// LLM / Info
	Text string
	// Partial 为 true 时表示 LLM 流式增量，UI 应在同一行追加 Text；false 表示整段或非流式输出。
	Partial bool

	// Tool
	ToolName string
	Detail   string

	// Usage（Kind == EventKindUsage）
	SessionTokenTotal      int64
	LastPromptTokens       int
	LastCompletionTokens   int
	ContextPercent         float64 // <0 表示不在 UI 显示百分比（未配置 context_window_tokens）
}

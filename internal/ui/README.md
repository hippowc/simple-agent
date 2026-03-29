# `internal/ui`

终端对话界面：基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 与 [bubbles](https://github.com/charmbracelet/bubbles)（`viewport`、`textinput`）。入口为 **`ui.Run`**，由 `cmd/simple-agent` 调用；界面只通过 **`Agent`** 接口驱动 agent，不直接依赖 LLM 或工具实现。

## 代码结构（按职责）

| 文件 | 职责 |
|------|------|
| `run.go` | 启动壳：进度条、`tea.Program`、进入主界面。 |
| `model.go` | Bubble Tea `Model`：`Update`/`View`、`handleKey`/`submit`、`syncViewport`（时间线内容）。 |
| `model_events.go` | `AgentEvent` → `feedBlock` / 流式缓冲（`applyAgentEvent` 等）。 |
| `feed.go` | 时间线数据：`feedBlock`、`blockKind`、折叠与工具名等纯辅助。 |
| `timeline.go` | 时间线视图：lipgloss 样式与 `renderFeed`（纯渲染，无 Tea）。 |
| `input.go` | 键盘：折叠块序号解析（`blockToggleIndex`）。 |
| `tui.go` | `Agent` 接口（便于测试注入）。 |

## 启动流程

1. 全屏进入备用屏（`AltScreen`），显示初始化进度条。
2. 后台完成 `Agent` 构造后，切换到主界面：上方可滚动时间线 + 底部输入框。

## 界面布局

- **主区（viewport）**：对话时间线可滚动，长内容时整块一起滚动，避免与底栏错位。
- **底栏**：分隔线、单行输入（`›` 提示符）、再一条分隔线、**用量行**（session 累计 token、上一轮 prompt+completion、上下文百分比，百分比需 `config.json` 中 `llm.context_window_tokens`）、帮助文案。

## 操作方式

| 操作 | 说明 |
|------|------|
| **输入并 Enter** | 发送消息；Agent 忙时 Enter 不提交新消息（可滚屏）。 |
| **鼠标滚轮** | 在主区上下滚动时间线。 |
| **PgUp / PgDn** | 翻页滚动（由 viewport 处理）。 |
| **Alt + 1～9** | 展开/折叠对应序号的时间线块（工具调用等可折叠块）。 |
| **1～9**（无 Alt） | 在 **Agent 运行中**，或 **输入框为空** 时，切换第 1～9 个块的折叠；否则数字作为普通输入。 |
| **Ctrl+C** | 退出程序。 |

文本形式的退出：`quit`、`exit` 或 `/quit`（发送后退出）。**Ctrl+C** 也可退出；**Ctrl+V** 用于粘贴（由输入框默认处理，未单独拦截）。

## 内置斜杠命令

以 `/` 开头的行会优先尝试 **`/model`、`/prompt`**（由 Agent 处理）与 **`/tools`**（`builtin.Dispatch`）。无法识别的 `/…` **不会**调用模型，只显示错误提示。读/写文件请用自然语言让模型调工具。

- `/tools` — 列出已注册工具名。
- `/model` …、`/prompt` … — 见主 README 与代码内命令说明。

更完整的 Agent 与工具行为见 [`internal/agent/README.md`](../agent/README.md)。

## 文案

界面英文文案内置在 **`common.DefaultUIText()`**（`internal/common/ui_text.go`），不提供外部配置文件。

## 等待与 loading

- **忙碌态**：使用 [bubbles/spinner](https://github.com/charmbracelet/bubbles/tree/master/spinner)，当前为 **`Dot`**（Braille 点阵）；与 `waitAgentEvent` 并行，由 `spinner.TickMsg` 驱动刷新。
- **模型**：非流式且尚无正文时，仅显示 spinner（无额外标题/提示句）。流式且缓冲为空时，在**流式轨**上只显示 spinner；有缓冲时显示缓冲 + 光标。
- **仅工具、无对话文本**：空模型块直接收口为完成态，正文为空（无占位说明句）。
- **工具**：`ToolStart` 后正文为 spinner + 工具友好名；收到 **`EventKindTool`** 后合并为结果。

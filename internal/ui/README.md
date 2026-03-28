# `internal/ui`

终端对话界面：基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 与 [bubbles](https://github.com/charmbracelet/bubbles)（`viewport`、`textinput`）。入口为 **`ui.Run`**，由 `cmd/simple-agent` 调用；界面只通过 **`Agent`** 接口驱动 agent，不直接依赖 LLM 或工具实现。

## 启动流程

1. 全屏进入备用屏（`AltScreen`），显示 Logo 与初始化进度条。
2. 后台完成 `Agent` 构造后，切换到主界面：上方可滚动时间线 + 底部输入框。

## 界面布局

- **主区（viewport）**：顶部品牌区（Logo）与对话时间线在同一可滚动区域内，长内容时整块一起滚动，避免与底栏错位。
- **底栏**：分隔线、单行输入（`›` 提示符）、再一条分隔线、底部帮助文案。

## 操作方式

| 操作 | 说明 |
|------|------|
| **输入并 Enter** | 发送消息；Agent 忙时 Enter 不提交新消息（可滚屏）。 |
| **鼠标滚轮** | 在主区上下滚动时间线。 |
| **PgUp / PgDn** | 翻页滚动（由 viewport 处理）。 |
| **Alt + 1～9** | 展开/折叠对应序号的时间线块（工具调用等可折叠块）。 |
| **1～9**（无 Alt） | 在 **Agent 运行中**，或 **输入框为空** 时，切换第 1～9 个块的折叠；否则数字作为普通输入。 |
| **Ctrl+C** | 退出程序。 |

文本形式的退出：`quit` 或 `exit`（发送后退出）。

## 内置斜杠命令

由 Agent 侧 `builtin.Dispatch` 处理，在输入框中发送以 `/` 开头的行即可（与对 LLM 的自然语言输入共用同一入口）：

- `/tools` — 列出已注册工具名。
- `/read <path>` — 读文件（相对路径相对 **工作区**）。
- `/write <path> <content>` — 写文件；`<content>` 为剩余整段文本。

更完整的 Agent 与工具行为见 [`internal/agent/README.md`](../agent/README.md)。

## 配置对界面的影响

`config.json` 中 **`ui.llm_running_title`**：流式生成时，时间线里展示的英文提示文案（默认 `Generating…`）。

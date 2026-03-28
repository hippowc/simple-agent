# simple-agent

一个 Go 版本的简化 LLM Agent 项目骨架，目标是模仿 Claude Code 的核心分层。

## 模块结构

- `cmd/simple-agent`：程序入口（加载配置、启动 UI）；LLM 与 Tools 的组装在 `agent` 内完成。
- `internal/ui`：终端 UI（[Bubble Tea](https://github.com/charmbracelet/bubbletea)）：上方主区可滚动，底部单行输入，仅调用 `agent`。
- `internal/agent`：Agent 主循环与调度（LLM 调用、Tools 调用、结果返回 UI）。
- `internal/llm`：LLM 抽象与实现（当前实现 OpenAI 协议）。
- `internal/tools`：工具抽象与实现（`read_file` / `write_file`；`find_files` 基于 [doublestar](https://github.com/bmatcuk/doublestar) glob；`grep_content` 正则搜索；`run_shell` 在工作区下执行命令）。
- `internal/common`：公共能力（配置加载与写入）。

## 快速开始

1. 配置 `config.json`（首次运行会自动生成默认配置）。
2. 设置 `llm.api_key`。
3. 运行：

```bash
go run ./cmd/simple-agent
```

## 当前 TUI 内置命令

- `/tools`：查看工具列表
- `/read <path>`：读取文件
- `/write <path> <content>`：写入文件
- `quit` / `exit`：退出
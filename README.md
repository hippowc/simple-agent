# simple-agent

在终端里使用的简易 LLM Agent：对话界面 + 可调用本地工具（读/写/改文件、搜索、执行 shell 等），配置兼容 OpenAI 协议 API。

---

## 环境要求

- 已安装 **Go**（本仓库 `go.mod` 为 **1.26.1**，建议使用同主版本或兼容的较新 Go）。

---

## 获取与运行

```bash
git clone <你的仓库地址> simple-agent
cd simple-agent
go run ./cmd/simple-agent
```

首次若尚无配置文件，程序会在用户目录下创建空配置模板，见下文「配置文件放哪里」。

---

## 配置 API 与工作区

### 配置文件放哪里（优先级从高到低）

1. **当前工作目录**下的 `config.json`（适合按项目单独配置）。
2. 用户目录 **`%USERPROFILE%\.simple-agent\config\config.json`**（Windows）或 **`~/.simple-agent/config/config.json`**（Unix）。

若两处都不存在，会在 **用户目录** 自动创建一份空模板，你需要编辑并至少填写 **`llm.api_key`**（以及按需改模型、地址等）。

### 常用字段说明

| 字段 | 含义 |
|------|------|
| `workspace` | 工具默认工作目录（读文件、写文件、`run_shell` 的 cwd 等）。留空则使用启动时的当前目录。 |
| `llm.provider` | 当前实现按 OpenAI 风格客户端使用，一般填 `openai`。 |
| `llm.base_url` | API 根地址，例如 `https://api.openai.com/v1` 或兼容网关地址。 |
| `llm.api_key` | 密钥。 |
| `llm.model` | 模型名，例如 `gpt-4o-mini`。 |
| `llm.stream` | 可选；`true`（默认）使用 **chat 流式 + 聚合 tool_calls**；`false` 使用单次非流式请求。 |
| `llm.context_window_tokens` | 可选；模型上下文窗口上限（token），用于 TUI 底部显示 **上下文占用百分比**。`0` 或不写则百分比处显示 `—`（仍会显示 session / last 用量）。 |

流式请求会带上 `stream_options.include_usage` 以尽量从 API 读取 token；若你所用的兼容网关不支持该字段，可能返回错误，此时可将 **`llm.stream`** 设为 **`false`** 走非流式，或换用支持该选项的端点。

示例（请按需修改，勿提交真实密钥）：

```json
{
  "workspace": "C:/path/to/your/project",
  "llm": {
    "provider": "openai",
    "base_url": "https://api.openai.com/v1",
    "api_key": "YOUR_KEY",
    "model": "gpt-4o-mini",
    "context_window_tokens": 128000
  }
}
```

### 编译成可执行文件

```bash
go build -o simple-agent ./cmd/simple-agent
```

将生成的 `simple-agent` 与 `config.json` 放在同一目录，或依赖用户目录下的全局配置。

---

## 在终端里怎么用

1. 启动后出现加载条，就绪后进入对话界面。
2. 在底部 **`›`** 后输入内容，**Enter** 发送。
3. 用 **鼠标滚轮** 或 **PgUp / PgDn** 在主区域滚动查看长回复。
4. **Ctrl+C** 退出；或在输入框发送 **`quit`** / **`exit`**。

### 内置命令（以 `/` 开头）

在输入框中整行发送：

| 命令 | 作用 |
|------|------|
| `/tools` | 列出当前注册的工具名称。 |
| `/read <path>` | 读取工作区内的文件。 |
| `/write <path> <content>` | 写入文件；`content` 为该行剩余全部文本。 |

其他需求交给模型，由模型通过工具调用完成（具体工具集见源码 `internal/tools`）。

### 折叠时间线块

对带工具调用等可折叠块，可用 **Alt+1～9** 切换第 1～9 个块的展开/折叠；在 Agent 运行中或输入框为空时，也可用 **1～9**（无 Alt）快速切换。

更细的界面说明（布局、按键与配置项）见 **[`internal/ui/README.md`](internal/ui/README.md)**。

---

## 仓库结构（概览）

| 路径 | 作用 |
|------|------|
| `cmd/simple-agent` | 程序入口：加载配置、启动 TUI。 |
| `internal/ui` | 终端 UI。 |
| `internal/agent` | 对话回合与 LLM/工具编排。 |
| `internal/llm` | LLM 客户端抽象与 OpenAI 协议实现。 |
| `internal/tools` | 工具注册与实现。 |
| `internal/common` | 配置加载与默认值。 |

Agent 内部设计见 **[`internal/agent/README.md`](internal/agent/README.md)**。

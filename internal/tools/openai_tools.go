package tools

import "simple-agent/internal/llm"

// OpenAIToolDefinitions 返回注册给 Chat Completions 的 tools 列表（与 Call 所用名称一致）。
func OpenAIToolDefinitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{
		{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        "read_file",
				Description: "Read the full text content of a file under the workspace by relative or absolute path.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path relative to workspace or absolute within workspace.",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        "write_file",
				Description: "Write text content to a file under the workspace, creating parent directories if needed.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File path relative to workspace or absolute within workspace.",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Full file content to write.",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        "find_files",
				Description: "Find files under a directory whose paths match a glob (e.g. **/*.go). Paths use / separators relative to root.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "Glob pattern (e.g. **/*.go, **/test_*.txt).",
						},
						"root": map[string]interface{}{
							"type":        "string",
							"description": "Directory to search under workspace; default \".\".",
						},
						"max_results": map[string]interface{}{
							"type":        "string",
							"description": "Max files to return (default 500, cap 10000).",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        "grep_content",
				Description: "Search file contents with a regular expression in a file or recursively in a directory. Skips likely binary files.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "Regular expression (Go regexp syntax).",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "File or directory path under workspace.",
						},
						"glob": map[string]interface{}{
							"type":        "string",
							"description": "Optional filename glob filter (e.g. *.go) when searching directories.",
						},
						"max_results": map[string]interface{}{
							"type":        "string",
							"description": "Max matching lines to return (default 200).",
						},
					},
					"required": []string{"pattern", "path"},
				},
			},
		},
		{
			Type: "function",
			Function: llm.FunctionSchema{
				Name:        "run_shell",
				Description: "Run a shell command with cwd set to workspace. Prefers bash -c; on Windows may use PowerShell or cmd. Output may be truncated. Use with care.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Full command line to pass to the shell (e.g. ls -la or Get-ChildItem).",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}

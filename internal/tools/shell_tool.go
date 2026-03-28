package tools

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
)

const maxShellOutput = 256 * 1024

// RunShellTool 在工作区根目录下执行 shell 命令（优先 bash，否则按系统选择解释器）。
type RunShellTool struct {
	workspace string
}

func NewRunShellTool(workspace string) *RunShellTool {
	return &RunShellTool{workspace: workspace}
}

func (t *RunShellTool) Name() string { return "run_shell" }

func (t *RunShellTool) Description() string {
	return "Run a shell command with working directory set to the workspace. Uses bash -c when available; on Windows falls back to PowerShell or cmd if bash is not in PATH. Dangerous: only use trusted commands."
}

func (t *RunShellTool) Call(ctx context.Context, input CallInput) (string, error) {
	cmd := input.Arguments["command"]
	if cmd == "" {
		return "", errors.New("command is required")
	}
	dir, err := resolvePath(t.workspace, ".")
	if err != nil {
		return "", err
	}

	name, args := shellInvocation(cmd)
	c := exec.CommandContext(ctx, name, args...)
	c.Dir = dir
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	runErr := c.Run()
	text := out.String()
	if len(text) > maxShellOutput {
		text = text[:maxShellOutput] + "\n...(output truncated)"
	}
	if runErr != nil {
		if text != "" {
			return text, runErr
		}
		return "", runErr
	}
	if text == "" {
		return "(no output)", nil
	}
	return text, nil
}

// shellInvocation returns executable name and args such that the user command runs as a single script line.
func shellInvocation(command string) (string, []string) {
	if path, err := exec.LookPath("bash"); err == nil {
		return path, []string{"-c", command}
	}
	if runtime.GOOS == "windows" {
		if path, err := exec.LookPath("powershell"); err == nil {
			return path, []string{"-NoProfile", "-NonInteractive", "-Command", command}
		}
		return "cmd", []string{"/C", command}
	}
	if path, err := exec.LookPath("sh"); err == nil {
		return path, []string{"-c", command}
	}
	// last resort
	return "sh", []string{"-c", command}
}

package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileTool struct {
	workspace string
}

func NewReadFileTool(workspace string) *ReadFileTool {
	return &ReadFileTool{workspace: workspace}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read file content by path"
}

func (t *ReadFileTool) Call(_ context.Context, input CallInput) (string, error) {
	target, ok := input.Arguments["path"]
	if !ok || target == "" {
		return "", errors.New("path is required")
	}
	resolved, err := resolvePath(t.workspace, target)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type WriteFileTool struct {
	workspace string
}

func NewWriteFileTool(workspace string) *WriteFileTool {
	return &WriteFileTool{workspace: workspace}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write file content by path"
}

func (t *WriteFileTool) Call(_ context.Context, input CallInput) (string, error) {
	target, ok := input.Arguments["path"]
	if !ok || target == "" {
		return "", errors.New("path is required")
	}
	content := input.Arguments["content"]

	resolved, err := resolvePath(t.workspace, target)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(resolved, []byte(content), 0o644); err != nil {
		return "", err
	}
	return "ok", nil
}

// EditFileTool 在已存在文件中将唯一匹配的原文片段替换为新内容（或 replace_all 时替换全部）。
type EditFileTool struct {
	workspace string
}

func NewEditFileTool(workspace string) *EditFileTool {
	return &EditFileTool{workspace: workspace}
}

func (t *EditFileTool) Name() string { return "edit_file" }

func (t *EditFileTool) Description() string {
	return "Edit an existing file by replacing old_string with new_string. Without replace_all, old_string must appear exactly once."
}

func (t *EditFileTool) Call(_ context.Context, input CallInput) (string, error) {
	target, ok := input.Arguments["path"]
	if !ok || target == "" {
		return "", errors.New("path is required")
	}
	oldStr, ok := input.Arguments["old_string"]
	if !ok {
		return "", errors.New("old_string is required")
	}
	if oldStr == "" {
		return "", errors.New("old_string must not be empty")
	}
	newStr := input.Arguments["new_string"]

	resolved, err := resolvePath(t.workspace, target)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	content := string(data)

	replaceAll := parseArgBool(input.Arguments["replace_all"])
	var out string
	if replaceAll {
		out = strings.ReplaceAll(content, oldStr, newStr)
	} else {
		n := strings.Count(content, oldStr)
		switch n {
		case 0:
			return "", errors.New("old_string not found in file")
		case 1:
			out = strings.Replace(content, oldStr, newStr, 1)
		default:
			return "", fmt.Errorf("old_string matches %d times; must be unique, or set replace_all to true", n)
		}
	}
	if out == content {
		return "ok (no changes)", nil
	}
	if err := os.WriteFile(resolved, []byte(out), 0o644); err != nil {
		return "", err
	}
	return "ok", nil
}

func parseArgBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes"
}

func resolvePath(workspace, target string) (string, error) {
	if workspace == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		workspace = cwd
	}
	workspace = filepath.Clean(workspace)

	var fullPath string
	if filepath.IsAbs(target) {
		fullPath = filepath.Clean(target)
	} else {
		fullPath = filepath.Join(workspace, target)
	}

	rel, err := filepath.Rel(workspace, fullPath)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", errors.New("path out of workspace is not allowed")
	}
	return fullPath, nil
}

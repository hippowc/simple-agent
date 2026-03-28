package tools

import (
	"context"
	"errors"
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

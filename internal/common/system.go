package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const localSystemFile = "system.md"

// UserSystemPath 返回用户级 system 提示词路径：~/.simple-agent/config/system.md
func UserSystemPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}
	return filepath.Join(home, ".simple-agent", "config", "system.md"), nil
}

// LoadSystemPromptAuto 按顺序读取 system.md：1) 当前工作目录下 system.md；2) 用户目录 .simple-agent/config/system.md。
// 若均不存在则返回 ("", nil)；存在但读取失败则返回错误。内容会做 strings.TrimSpace。
func LoadSystemPromptAuto() (string, error) {
	if fi, err := os.Stat(localSystemFile); err == nil && !fi.IsDir() {
		return readSystemFile(localSystemFile)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	userPath, err := UserSystemPath()
	if err != nil {
		return "", err
	}
	if fi, err := os.Stat(userPath); err == nil && !fi.IsDir() {
		return readSystemFile(userPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return "", nil
}

// ResolveSystemPrompt 解析系统提示：system_prompt_file → system_prompt（@文件 或内联）→ LoadSystemPromptAuto。
// 相对路径相对于进程当前工作目录（一般为启动目录）。
func ResolveSystemPrompt(p PromptConfig) (string, error) {
	if path := strings.TrimSpace(p.SystemPromptFile); path != "" {
		return readPromptFile(resolvePromptPath(path))
	}
	if s := strings.TrimSpace(p.SystemPrompt); s != "" {
		if rest, ok := strings.CutPrefix(s, "@"); ok {
			return readPromptFile(resolvePromptPath(strings.TrimSpace(rest)))
		}
		return s, nil
	}
	return LoadSystemPromptAuto()
}

// ResolveUserPromptTemplate 解析用户侧提示模板：user_prompt_file → user_prompt（@文件 或内联）；皆可空。
func ResolveUserPromptTemplate(p PromptConfig) (string, error) {
	if path := strings.TrimSpace(p.UserPromptFile); path != "" {
		return readPromptFile(resolvePromptPath(path))
	}
	if s := strings.TrimSpace(p.UserPrompt); s != "" {
		if rest, ok := strings.CutPrefix(s, "@"); ok {
			return readPromptFile(resolvePromptPath(strings.TrimSpace(rest)))
		}
		return s, nil
	}
	return "", nil
}

func resolvePromptPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Join(wd, p)
}

func readPromptFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("prompt file path is empty")
	}
	return readSystemFile(path)
}

func readSystemFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

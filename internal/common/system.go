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

func readSystemFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const localConfigFile = "config.json"

type Config struct {
	Workspace string    `json:"workspace"`
	LLM       LLMConfig `json:"llm"`
}

type LLMConfig struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// DefaultConfig 返回带有一套合理默认值的配置（可用于本地模板或测试）。
func DefaultConfig() Config {
	workspace, _ := os.Getwd()
	return Config{
		Workspace: workspace,
		LLM: LLMConfig{
			Provider: "openai",
			BaseURL:  "https://api.openai.com/v1",
			Model:    "gpt-4o-mini",
		},
	}
}

// EmptyConfig 返回各字段为空的配置，用于在用户目录首次创建配置文件。
func EmptyConfig() Config {
	return Config{
		Workspace: "",
		LLM: LLMConfig{
			Provider: "",
			BaseURL:  "",
			APIKey:   "",
			Model:    "",
		},
	}
}

// UserConfigPath 返回默认用户级配置路径：~/.simple-agent/config/config.json
func UserConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home: %w", err)
	}
	return filepath.Join(home, ".simple-agent", "config", "config.json"), nil
}

// LoadConfigAuto 按顺序加载配置：1) 当前工作目录下 config.json；2) 用户目录下 .simple-agent/config/config.json；
// 若两处都不存在，则在用户目录创建空配置文件并返回（仅内存中会为当前运行填入 workspace 等运行时默认值）。
func LoadConfigAuto() (Config, error) {
	if fi, err := os.Stat(localConfigFile); err == nil && !fi.IsDir() {
		return LoadConfig(localConfigFile)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	userPath, err := UserConfigPath()
	if err != nil {
		return Config{}, err
	}
	if fi, err := os.Stat(userPath); err == nil && !fi.IsDir() {
		return LoadConfig(userPath)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	cfg := EmptyConfig()
	if err := SaveConfig(userPath, cfg); err != nil {
		return Config{}, err
	}
	applyRuntimeDefaults(&cfg)
	return cfg, nil
}

// LoadConfig 从指定路径读取 JSON 配置；文件必须已存在。
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, errors.New("config path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	applyRuntimeDefaults(&cfg)
	return cfg, nil
}

func applyRuntimeDefaults(cfg *Config) {
	if cfg.Workspace == "" {
		cfg.Workspace, _ = os.Getwd()
	}
	if llmAllEmpty(cfg.LLM) {
		return
	}
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "openai"
	}
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gpt-4o-mini"
	}
}

func llmAllEmpty(c LLMConfig) bool {
	return c.Provider == "" && c.BaseURL == "" && c.APIKey == "" && c.Model == ""
}

func SaveConfig(path string, cfg Config) error {
	if path == "" {
		return errors.New("config path is required")
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

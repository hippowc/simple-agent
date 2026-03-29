package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const localConfigFile = "config.json"

type Config struct {
	LLM    LLMSection   `json:"llm"`
	Prompt PromptConfig `json:"prompt,omitempty"`
}

// PromptConfig 可选提示词：非空时优先于自动 system.md（system）或与用户输入组合（user）。
// 文件：可使用 system_prompt_file / user_prompt_file，或在 system_prompt / user_prompt 中写 @相对或绝对路径（@ 前缀仅在对应 *_file 为空时生效）。
type PromptConfig struct {
	SystemPrompt     string `json:"system_prompt,omitempty"`
	UserPrompt       string `json:"user_prompt,omitempty"`
	SystemPromptFile string `json:"system_prompt_file,omitempty"`
	UserPromptFile   string `json:"user_prompt_file,omitempty"`
}

// LLMSection：Use 为当前使用的 profile 的 name；各 profile 用 name 区分，不再使用 per-profile default 字段。
type LLMSection struct {
	Use      string       `json:"use"`
	Profiles []LLMProfile `json:"profiles"`
}

// LLMProfile 单套模型连接。
type LLMProfile struct {
	LLMConfig
	Name string `json:"name,omitempty"`
}

// LLMConfig 为与 LLM API 通信的字段（不含 name）。
type LLMConfig struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	// Stream 为 nil 时默认 true（chat 流式 + 聚合 tool_calls）；设为 false 则使用非流式 Generate。
	Stream *bool `json:"stream,omitempty"`
	// ContextWindowTokens 模型上下文窗口上限（token），用于 UI 显示上下文占用百分比。0 表示不展示百分比。
	ContextWindowTokens int `json:"context_window_tokens,omitempty"`
}

// UnmarshalJSON 支持 llm.use；旧版带 profiles[].default 时会迁移为 use。
func (s *LLMSection) UnmarshalJSON(data []byte) error {
	var aux struct {
		Use      string            `json:"use"`
		Profiles []json.RawMessage `json:"profiles"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	type profileFile struct {
		LLMConfig
		Name    string `json:"name,omitempty"`
		Default bool   `json:"default,omitempty"`
	}
	var profiles []LLMProfile
	var defaultName string
	for _, raw := range aux.Profiles {
		var pf profileFile
		if err := json.Unmarshal(raw, &pf); err != nil {
			return err
		}
		if pf.Name == "" {
			pf.Name = "default"
		}
		profiles = append(profiles, LLMProfile{LLMConfig: pf.LLMConfig, Name: pf.Name})
		if pf.Default && defaultName == "" {
			defaultName = pf.Name
		}
	}
	s.Profiles = profiles
	s.Use = strings.TrimSpace(aux.Use)
	if s.Use == "" && defaultName != "" {
		s.Use = defaultName
	}
	if s.Use == "" && len(profiles) > 0 {
		s.Use = profiles[0].Name
	}
	return nil
}

func (s LLMSection) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Use      string       `json:"use"`
		Profiles []LLMProfile `json:"profiles"`
	}{
		Use:      s.Use,
		Profiles: s.Profiles,
	})
}

// ActiveLLMProfile 返回 llm.use 所指 name 的 profile；若 use 为空则取第一条。
func (c Config) ActiveLLMProfile() (LLMProfile, error) {
	ps := c.LLM.Profiles
	if len(ps) == 0 {
		return LLMProfile{}, errors.New("no llm profiles configured")
	}
	use := strings.TrimSpace(c.LLM.Use)
	if use == "" {
		return ps[0], nil
	}
	for _, p := range ps {
		if p.Name == use {
			return p, nil
		}
	}
	return LLMProfile{}, fmt.Errorf("llm.use %q not found in profiles", use)
}

// FormatUserPrompt 将配置中的 user_prompt 与用户输入组合。模板含 "{{input}}" 时替换；否则非空模板以两个换行接在用户输入之前。
func FormatUserPrompt(template, userInput string) string {
	t := strings.TrimSpace(template)
	if t == "" {
		return userInput
	}
	if strings.Contains(t, "{{input}}") {
		return strings.ReplaceAll(t, "{{input}}", userInput)
	}
	return t + "\n\n" + userInput
}

// DefaultConfig 返回带有一套合理默认值的配置（可用于本地模板或测试）。
func DefaultConfig() Config {
	return Config{
		LLM: LLMSection{
			Use: "default",
			Profiles: []LLMProfile{
				{
					Name: "default",
					LLMConfig: LLMConfig{
						Provider: "openai",
						BaseURL:  "https://api.openai.com/v1",
						Model:    "gpt-4o-mini",
					},
				},
			},
		},
	}
}

// EmptyConfig 返回各字段为空的配置，用于在用户目录首次创建配置文件。
func EmptyConfig() Config {
	return Config{
		LLM: LLMSection{
			Use: "default",
			Profiles: []LLMProfile{
				{
					Name: "default",
					LLMConfig: LLMConfig{
						Provider: "",
						BaseURL:  "",
						APIKey:   "",
						Model:    "",
					},
				},
			},
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
// 若两处都不存在，则在用户目录创建空配置文件并返回（并应用 LLM 等运行时默认值）。
// savePath 为应写回的配置文件绝对路径（供保存向导结果与 /model、/prompt 持久化）。
func LoadConfigAuto() (cfg Config, savePath string, err error) {
	if fi, err := os.Stat(localConfigFile); err == nil && !fi.IsDir() {
		abs, absErr := filepath.Abs(localConfigFile)
		if absErr != nil {
			return Config{}, "", absErr
		}
		cfg, err := LoadConfig(localConfigFile)
		return cfg, abs, err
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, "", err
	}

	userPath, err := UserConfigPath()
	if err != nil {
		return Config{}, "", err
	}
	if fi, err := os.Stat(userPath); err == nil && !fi.IsDir() {
		cfg, err := LoadConfig(userPath)
		return cfg, userPath, err
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, "", err
	}

	created := EmptyConfig()
	if err := SaveConfig(userPath, created); err != nil {
		return Config{}, "", err
	}
	applyRuntimeDefaults(&created)
	return created, userPath, nil
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

// ApplyRuntimeDefaults 为 LLM profile 填充 provider、base_url、model、stream 等默认值（与加载配置后一致）。
func ApplyRuntimeDefaults(cfg *Config) {
	applyRuntimeDefaults(cfg)
}

func applyRuntimeDefaults(cfg *Config) {
	if llmSectionAllEmpty(cfg.LLM) {
		return
	}
	for i := range cfg.LLM.Profiles {
		applyLLMProfileDefaults(&cfg.LLM.Profiles[i])
	}
	if len(cfg.LLM.Profiles) > 0 && strings.TrimSpace(cfg.LLM.Use) == "" {
		cfg.LLM.Use = cfg.LLM.Profiles[0].Name
	}
}

// CloneConfig 深拷贝配置（用于运行时修改）。
func CloneConfig(c Config) (Config, error) {
	b, err := json.Marshal(&c)
	if err != nil {
		return Config{}, err
	}
	var out Config
	if err := json.Unmarshal(b, &out); err != nil {
		return Config{}, err
	}
	return out, nil
}

// IsCompliant 当存在可用默认 profile 且 base_url、api_key、model 均非空时为 true。
func (c Config) IsCompliant() bool {
	prof, err := c.ActiveLLMProfile()
	if err != nil {
		return false
	}
	return strings.TrimSpace(prof.BaseURL) != "" &&
		strings.TrimSpace(prof.APIKey) != "" &&
		strings.TrimSpace(prof.Model) != ""
}

func applyLLMProfileDefaults(p *LLMProfile) {
	if llmConfigAllEmpty(p.LLMConfig) {
		return
	}
	if p.Provider == "" {
		p.Provider = "openai"
	}
	if p.BaseURL == "" {
		p.BaseURL = "https://api.openai.com/v1"
	}
	if p.Model == "" {
		p.Model = "gpt-4o-mini"
	}
	if p.Stream == nil {
		t := true
		p.Stream = &t
	}
}

func llmSectionAllEmpty(sec LLMSection) bool {
	if len(sec.Profiles) == 0 {
		return true
	}
	for _, p := range sec.Profiles {
		if !llmConfigAllEmpty(p.LLMConfig) {
			return false
		}
	}
	return true
}

func llmConfigAllEmpty(c LLMConfig) bool {
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

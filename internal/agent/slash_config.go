package agent

import (
	"context"
	"fmt"
	"strings"

	"simple-agent/internal/builtin"
	"simple-agent/internal/common"
)

func (a *Agent) tryConfigSlashCommands(_ context.Context, userInput string) ([]builtin.Output, bool) {
	parts := strings.Fields(userInput)
	if len(parts) == 0 {
		return nil, false
	}
	switch parts[0] {
	case "/model":
		out, err := a.handleModelCommand(parts)
		if err != nil {
			return []builtin.Output{{Kind: builtin.KindError, Err: err.Error()}}, true
		}
		return out, true
	case "/prompt":
		out, err := a.handlePromptCommand(userInput)
		if err != nil {
			return []builtin.Output{{Kind: builtin.KindError, Err: err.Error()}}, true
		}
		return out, true
	default:
		return nil, false
	}
}

func (a *Agent) handleModelCommand(parts []string) ([]builtin.Output, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfg, err := common.CloneConfig(a.cfg)
	if err != nil {
		return nil, err
	}

	switch len(parts) {
	case 1:
		return a.modelListOutputsUnlocked(&cfg), nil
	default:
		switch parts[1] {
		case "use":
			if len(parts) < 3 {
				return nil, fmt.Errorf("usage: /model use <name>")
			}
			name := parts[2]
			if err := a.modelUseUnlocked(&cfg, name); err != nil {
				return nil, err
			}
		case "add":
			if len(parts) < 6 {
				return nil, fmt.Errorf("usage: /model add <name> <base_url> <api_key> <model>")
			}
			name := parts[2]
			baseURL := parts[3]
			apiKey := parts[4]
			model := parts[5]
			if err := a.modelAddUnlocked(&cfg, name, baseURL, apiKey, model); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown subcommand %q (try: /model, /model use <name>, /model add ...)", parts[1])
		}
	}

	if a.configPath == "" {
		return nil, fmt.Errorf("config path unknown; cannot persist")
	}

	common.ApplyRuntimeDefaults(&cfg)
	if err := common.SaveConfig(a.configPath, cfg); err != nil {
		return nil, err
	}
	if err := a.applySessionFromConfigUnlocked(cfg); err != nil {
		return nil, err
	}
	a.cfg = cfg
	return []builtin.Output{{Kind: builtin.KindInfo, InfoText: "model configuration updated"}}, nil
}

func (a *Agent) modelListOutputsUnlocked(cfg *common.Config) []builtin.Output {
	var b strings.Builder
	b.WriteString("LLM profiles:\n")
	useName := strings.TrimSpace(cfg.LLM.Use)
	for i := range cfg.LLM.Profiles {
		p := &cfg.LLM.Profiles[i]
		mark := " "
		if p.Name == useName {
			mark = "*"
		}
		b.WriteString(fmt.Sprintf("  %s %-16s  model=%s\n    base_url=%s\n", mark, p.Name, p.Model, p.BaseURL))
	}
	return []builtin.Output{{Kind: builtin.KindInfo, InfoText: strings.TrimRight(b.String(), "\n")}}
}

func (a *Agent) modelUseUnlocked(cfg *common.Config, name string) error {
	found := -1
	for i := range cfg.LLM.Profiles {
		if cfg.LLM.Profiles[i].Name == name {
			found = i
			break
		}
	}
	if found < 0 {
		return fmt.Errorf("no profile named %q", name)
	}
	cfg.LLM.Use = name
	return nil
}

func (a *Agent) modelAddUnlocked(cfg *common.Config, name, baseURL, apiKey, model string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	for i := range cfg.LLM.Profiles {
		if cfg.LLM.Profiles[i].Name == name {
			return fmt.Errorf("profile %q already exists", name)
		}
	}
	t := true
	p := common.LLMProfile{
		Name: name,
		LLMConfig: common.LLMConfig{
			Provider: "openai",
			BaseURL:  strings.TrimSpace(baseURL),
			APIKey:   strings.TrimSpace(apiKey),
			Model:    strings.TrimSpace(model),
			Stream:   &t,
		},
	}
	cfg.LLM.Profiles = append(cfg.LLM.Profiles, p)
	cfg.LLM.Use = name
	return nil
}

func (a *Agent) handlePromptCommand(line string) ([]builtin.Output, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfg, err := common.CloneConfig(a.cfg)
	if err != nil {
		return nil, err
	}

	rest := strings.TrimSpace(strings.TrimPrefix(line, "/prompt"))
	if rest == "" {
		return a.promptShowUnlocked(&cfg), nil
	}

	if a.configPath == "" {
		return nil, fmt.Errorf("config path unknown; cannot persist")
	}

	if strings.HasPrefix(rest, "system ") {
		sub := strings.TrimSpace(strings.TrimPrefix(rest, "system "))
		if err := a.applyPromptSystemUnlocked(&cfg, sub); err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(rest, "user ") {
		sub := strings.TrimSpace(strings.TrimPrefix(rest, "user "))
		if err := a.applyPromptUserUnlocked(&cfg, sub); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("usage: /prompt | /prompt system ... | /prompt user ...")
	}

	if err := common.SaveConfig(a.configPath, cfg); err != nil {
		return nil, err
	}
	if err := a.applySessionFromConfigUnlocked(cfg); err != nil {
		return nil, err
	}
	a.cfg = cfg
	return []builtin.Output{{Kind: builtin.KindInfo, InfoText: "prompt updated (session active)"}}, nil
}

func (a *Agent) promptShowUnlocked(cfg *common.Config) []builtin.Output {
	sysF := cfg.Prompt.SystemPromptFile
	userF := cfg.Prompt.UserPromptFile
	sin := cfg.Prompt.SystemPrompt
	uin := cfg.Prompt.UserPrompt
	preview := func(s string) string {
		s = strings.TrimSpace(s)
		if len(s) > 200 {
			return s[:200] + "…"
		}
		return s
	}
	var b strings.Builder
	b.WriteString("prompt config:\n")
	if sysF != "" {
		fmt.Fprintf(&b, "  system_prompt_file: %s\n", sysF)
	}
	if sin != "" {
		fmt.Fprintf(&b, "  system_prompt (inline): %s\n", preview(sin))
	}
	if sysF == "" && sin == "" {
		b.WriteString("  system: (auto from system.md / default)\n")
	}
	if userF != "" {
		fmt.Fprintf(&b, "  user_prompt_file: %s\n", userF)
	}
	if uin != "" {
		fmt.Fprintf(&b, "  user_prompt (inline): %s\n", preview(uin))
	}
	if userF == "" && uin == "" {
		b.WriteString("  user template: (none)\n")
	}
	b.WriteString("effective this session:\n")
	fmt.Fprintf(&b, "  system: %s\n", preview(a.session.systemPrompt))
	fmt.Fprintf(&b, "  user tpl: %s\n", preview(a.session.userPromptTemplate))
	return []builtin.Output{{Kind: builtin.KindInfo, InfoText: strings.TrimRight(b.String(), "\n")}}
}

func (a *Agent) applyPromptSystemUnlocked(cfg *common.Config, sub string) error {
	if sub == "clear" {
		cfg.Prompt.SystemPrompt = ""
		cfg.Prompt.SystemPromptFile = ""
		return nil
	}
	if strings.HasPrefix(sub, "file ") {
		path := strings.TrimSpace(strings.TrimPrefix(sub, "file "))
		if path == "" {
			return fmt.Errorf("usage: /prompt system file <path>")
		}
		cfg.Prompt.SystemPromptFile = path
		cfg.Prompt.SystemPrompt = ""
		return nil
	}
	if strings.HasPrefix(sub, "@") {
		cfg.Prompt.SystemPromptFile = strings.TrimSpace(strings.TrimPrefix(sub, "@"))
		cfg.Prompt.SystemPrompt = ""
		return nil
	}
	cfg.Prompt.SystemPrompt = sub
	cfg.Prompt.SystemPromptFile = ""
	return nil
}

func (a *Agent) applyPromptUserUnlocked(cfg *common.Config, sub string) error {
	if sub == "clear" {
		cfg.Prompt.UserPrompt = ""
		cfg.Prompt.UserPromptFile = ""
		return nil
	}
	if strings.HasPrefix(sub, "file ") {
		path := strings.TrimSpace(strings.TrimPrefix(sub, "file "))
		if path == "" {
			return fmt.Errorf("usage: /prompt user file <path>")
		}
		cfg.Prompt.UserPromptFile = path
		cfg.Prompt.UserPrompt = ""
		return nil
	}
	if strings.HasPrefix(sub, "@") {
		cfg.Prompt.UserPromptFile = strings.TrimSpace(strings.TrimPrefix(sub, "@"))
		cfg.Prompt.UserPrompt = ""
		return nil
	}
	cfg.Prompt.UserPrompt = sub
	cfg.Prompt.UserPromptFile = ""
	return nil
}


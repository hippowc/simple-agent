package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RunSetupWizard 从 stdin 读取必填项 base_url、api_key、model，生成单 profile 配置（provider 等由 ApplyRuntimeDefaults 补齐）。
func RunSetupWizard() (Config, error) {
	fmt.Fprintln(os.Stderr, "首次配置：请填写 LLM 连接信息（可直接回车使用括号内默认值）。")
	r := bufio.NewReader(os.Stdin)

	baseURL := readLine(r, "base_url", "https://api.openai.com/v1")
	apiKey := readLine(r, "api_key", "")
	model := readLine(r, "model", "gpt-4o-mini")

	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)

	if baseURL == "" || apiKey == "" || model == "" {
		return Config{}, fmt.Errorf("base_url、api_key、model 均不能为空")
	}

	cfg := Config{
		LLM: LLMSection{
			Use: "default",
			Profiles: []LLMProfile{
				{
					Name: "default",
					LLMConfig: LLMConfig{
						BaseURL: baseURL,
						APIKey:  apiKey,
						Model:   model,
					},
				},
			},
		},
	}
	ApplyRuntimeDefaults(&cfg)
	return cfg, nil
}

func readLine(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return def
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

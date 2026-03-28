package llm

import (
	"fmt"
	"strings"

	"simple-agent/internal/common"
)

func NewClient(cfg common.LLMConfig) (Client, error) {
	switch strings.ToLower(cfg.Provider) {
	case "openai":
		return NewOpenAIClient(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider: %s", cfg.Provider)
	}
}

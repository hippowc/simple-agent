package main

import (
	"context"
	"fmt"
	"os"

	"simple-agent/internal/common"
	"simple-agent/internal/ui"
)

func main() {
	ctx := context.Background()

	cfg, savePath, err := common.LoadConfigAuto()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}

	if !cfg.IsCompliant() {
		fmt.Fprintln(os.Stderr, "未检测到有效的 LLM 配置（需要 base_url、api_key、model）。")
		cfg, err = common.RunSetupWizard()
		if err != nil {
			fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
			os.Exit(1)
		}
		if err := common.SaveConfig(savePath, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "save config failed: %v\n", err)
			os.Exit(1)
		}
	}

	if err := ui.Run(ctx, cfg, savePath, common.DefaultUIText(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "tui exited with error: %v\n", err)
		os.Exit(1)
	}
}

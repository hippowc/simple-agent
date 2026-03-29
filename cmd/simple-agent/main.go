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

	cfg, err := common.LoadConfigAuto()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}

	if err := ui.Run(ctx, cfg, common.DefaultUIText(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "tui exited with error: %v\n", err)
		os.Exit(1)
	}
}

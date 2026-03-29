package common

// UIText 终端 UI 可见文案（内置英文，见 DefaultUIText）。
type UIText struct {
	WelcomeMarkdown    string
	InputPlaceholder   string
	HelpWithBlocks     string
	HelpNoBlocks       string
	ViewLoading        string
	LabelInfo          string
	LabelError         string
	ToolOutputHeader   string
	BootInitializing   string
	CompletionHint     string
	FoldExpandAltFmt   string
	FoldExpandAltRange string
	ToolDisplayNames   map[string]string
}

// DefaultUIText 返回内置英文文案。
func DefaultUIText() UIText {
	return UIText{
		WelcomeMarkdown:    "Ready.\n\nSend a message or a /command. Scroll with wheel or PgUp/PgDn.",
		InputPlaceholder:   "Msg · /model /prompt /tools /quit · Enter · Ctrl+C",
		HelpWithBlocks:     "Enter · scroll · Alt+1–9 · Ctrl+C",
		HelpNoBlocks:       "Enter · scroll · Ctrl+C",
		ViewLoading:        "Loading…",
		LabelInfo:          "Note",
		LabelError:         "Error",
		ToolOutputHeader:   "· output",
		BootInitializing:   "Initializing, please wait…",
		CompletionHint:     "↑↓ · Tab · Esc",
		FoldExpandAltFmt:   "alt + %d to expand",
		FoldExpandAltRange: "alt + 1-9 to expand",
		ToolDisplayNames: map[string]string{
			"read_file":    "Read file",
			"write_file":   "Write file",
			"edit_file":    "Edit file",
			"find_files":   "Find files",
			"grep_content": "Grep",
			"run_shell":    "Shell",
		},
	}
}

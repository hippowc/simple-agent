package ui

import (
	"strings"
	"time"
	"unicode/utf8"
)

// 时间线块类型：用户气泡、模型回复、工具结果，以及会话级提示/错误。
type blockKind int

const (
	kindPrompt blockKind = iota
	kindModel
	kindTool
	kindInfo
	kindError
)

type blockStatus int

const (
	statusRunning blockStatus = iota
	statusDone
	statusError
)

// foldLineThreshold 仅用于工具：超过此行数则默认折叠。
const foldLineThreshold = 3

// headerRailTimeGap 标题行中竖线与时间戳之间的空隙（单空格）。
const headerRailTimeGap = " "

// feedBlock 单条时间线；title 仅存工具原始名，展示用 toolFriendlyName(title)。
type feedBlock struct {
	kind     blockKind
	title    string
	status   blockStatus
	body     string
	expanded bool
	at       time.Time
}

func toolFriendlyName(name string) string {
	m := map[string]string{
		"read_file":    "读文件",
		"write_file":   "写文件",
		"find_files":   "查找",
		"grep_content": "搜索",
		"run_shell":    "终端",
	}
	if s, ok := m[name]; ok {
		return s
	}
	return strings.ReplaceAll(name, "_", " ")
}

func lineCount(s string) int {
	s = strings.TrimRight(s, "\n")
	if strings.TrimSpace(s) == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func isToolFoldable(blk *feedBlock) bool {
	return blk != nil && blk.kind == kindTool && lineCount(blk.body) > foldLineThreshold
}

func defaultExpandedForTool(body string) bool {
	return lineCount(body) <= foldLineThreshold
}

func oneLinePreview(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	rs := []rune(s)
	if len(rs) > maxRunes {
		return string(rs[:maxRunes]) + "…"
	}
	return s
}

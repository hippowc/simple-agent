package ui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

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

// feedBlock 时间线块；title 仅存工具原始名，展示用 toolFriendlyName(title)。
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

var (
	styleDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleBody = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleLbl  = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	// 时间戳：比正文更淡，紧跟状态点
	styleTS = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	// 折叠指示：略提亮，避免与「灰色点」混淆
	styleChev = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	// 小状态点（仅保留这一处彩色点）
	dotRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("·")
	dotDone    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("·")
	dotError   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("·")
	chevCol    = styleChev.Render("›")
	chevExp    = styleChev.Render("∨")
)

func dotFor(st blockStatus) string {
	switch st {
	case statusRunning:
		return dotRunning
	case statusError:
		return dotError
	default:
		return dotDone
	}
}

func tsFmt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04:05")
}

func indentEachLine(s string, spaces int) string {
	if spaces <= 0 || s == "" {
		return s
	}
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = pad + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func renderFeed(width int, blocks []feedBlock, welcome string, streaming bool, streamBuf string) string {
	if len(blocks) == 0 {
		return styleDim.Render(welcome)
	}
	var b strings.Builder
	for i := range blocks {
		blk := &blocks[i]
		dot := dotFor(blk.status)
		chev := chevCol
		if blk.expanded {
			chev = chevExp
		}
		tsStyled := styleTS.Render(tsFmt(blk.at))

		var prefix strings.Builder
		prefix.WriteString(dot)
		prefix.WriteString(" ")
		prefix.WriteString(tsStyled)
		prefix.WriteString(" ")
		switch blk.kind {
		case kindTool:
			prefix.WriteString(styleLbl.Render(toolFriendlyName(blk.title)))
			prefix.WriteString(" ")
			prefix.WriteString(chev)
		case kindInfo:
			prefix.WriteString(styleLbl.Render("提示"))
			prefix.WriteString(" ")
			prefix.WriteString(chev)
		case kindError:
			prefix.WriteString(styleLbl.Render("失败"))
			prefix.WriteString(" ")
			prefix.WriteString(chev)
		default:
			prefix.WriteString(chev)
		}
		prefixStr := prefix.String()
		prefixW := lipgloss.Width(prefixStr)

		previewMax := width - prefixW - 1
		if previewMax < 0 {
			previewMax = 0
		}
		preview := ""
		if !blk.expanded && previewMax > 0 {
			preview = styleDim.Render(" " + oneLinePreview(blk.body, previewMax))
		}
		b.WriteString(prefixStr)
		b.WriteString(preview)
		b.WriteString("\n")

		if blk.expanded && strings.TrimSpace(blk.body) != "" {
			bodyW := width - 2
			if bodyW < 20 {
				bodyW = width
			}
			bodyRendered := lipgloss.NewStyle().Width(bodyW).Render(strings.TrimRight(blk.body, "\n"))
			bodyRendered = indentEachLine(bodyRendered, 2)
			b.WriteString(bodyRendered)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if streaming && streamBuf != "" {
		b.WriteString(styleBody.Render(streamBuf))
		b.WriteString(styleDim.Render("▌"))
		b.WriteString("\n")
	}
	return b.String()
}

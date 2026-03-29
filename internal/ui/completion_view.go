package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"simple-agent/internal/agent"
)

var (
	styleSuggestHint = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleSuggestSel  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("240"))
	styleSuggestItem = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

const suggestMaxRows = 5

func suggestExtraLines(n int) int {
	if n <= 0 {
		return 0
	}
	return 2 + min(suggestMaxRows, n) // 标题行 + 列表
}

func renderSuggestPanel(width int, items []agent.CompletionItem, sel int, hint string) string {
	if len(items) == 0 || width <= 0 {
		return ""
	}
	show := items
	if len(show) > suggestMaxRows {
		show = show[:suggestMaxRows]
	}
	if sel >= len(show) {
		sel = len(show) - 1
	}
	if sel < 0 {
		sel = 0
	}
	var b strings.Builder
	b.WriteString(styleSuggestHint.Render(truncateRunes(hint, width)))
	b.WriteString("\n")
	for i, it := range show {
		line := "  " + it.Label
		line = truncateRunes(line, width)
		if i == sel {
			b.WriteString(styleSuggestSel.Width(width).Render(line))
		} else {
			b.WriteString(styleSuggestItem.Render(line))
		}
		b.WriteString("\n")
	}
	if len(items) > suggestMaxRows {
		more := styleSuggestHint.Render(truncateRunes("  …", width))
		b.WriteString(more)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncateRunes(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxW {
		return s
	}
	if maxW <= 1 {
		return string(r[:maxW])
	}
	return string(r[:maxW-1]) + "…"
}

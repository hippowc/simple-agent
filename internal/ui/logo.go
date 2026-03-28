package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var (
	logoBorder = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	logoMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	logoWord   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	logoMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// LogoHeader 顶部品牌区，替代纯文字标题。termW 为终端宽度，过窄时使用单行紧凑样式。
func LogoHeader(termW int) string {
	if termW > 0 && termW < 44 {
		return logoCompact()
	}
	return logoFramed()
}

func logoCompact() string {
	return " " + logoMark.Render("◆") + " " + logoWord.Render("simple-agent")
}

func logoFramed() string {
	const innerW = 26
	core := "  ◆  simple-agent"
	padN := innerW - utf8.RuneCountInString(core)
	if padN < 0 {
		padN = 0
	}

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(logoBorder.Render("╭" + strings.Repeat("─", innerW) + "╮"))
	b.WriteString("\n  ")
	b.WriteString(logoBorder.Render("│"))
	b.WriteString(logoMark.Render("  ◆  "))
	b.WriteString(logoWord.Render("simple-agent"))
	b.WriteString(logoBorder.Render(strings.Repeat(" ", padN)))
	b.WriteString(logoBorder.Render("│"))
	b.WriteString("\n  ")
	b.WriteString(logoBorder.Render("╰" + strings.Repeat("─", innerW) + "╯"))
	b.WriteString("\n  ")
	b.WriteString(logoMuted.Render("── agent shell ──"))
	return b.String()
}

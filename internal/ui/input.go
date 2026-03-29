package ui

import tea "github.com/charmbracelet/bubbletea"

// blockToggleIndex 解析折叠时间线块的按键：Alt+1..9 在任意时刻可用；纯数字 1..9 仅在
// agent 运行中或输入框为空时可用（否则应输入到消息里）。
func blockToggleIndex(msg tea.KeyMsg, busy bool, inputEmpty bool) (int, bool) {
	k := tea.Key(msg)
	if k.Type != tea.KeyRunes || len(k.Runes) != 1 {
		return 0, false
	}
	r := k.Runes[0]
	if r < '1' || r > '9' {
		return 0, false
	}
	idx := int(r - '1')
	if k.Alt {
		return idx, true
	}
	if busy || inputEmpty {
		return idx, true
	}
	return 0, false
}

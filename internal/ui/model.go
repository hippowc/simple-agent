package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"simple-agent/internal/agent"
	"simple-agent/internal/common"
)

// baseFooterReserve：主区下方固定占用行数（分隔线、输入、分隔线、单行状态栏），不含补全面板。
const baseFooterReserve = 6

type agentEventMsg struct{ ev agent.AgentEvent }

type turnDoneMsg struct{}

func waitAgentEvent(ch <-chan agent.AgentEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return turnDoneMsg{}
		}
		return agentEventMsg{ev: ev}
	}
}

var styleHelp = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

// newSpinner 使用 bubbles 的 Dot（Braille 点阵，紧凑清晰）。
func newSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("241"))),
	)
}

// model：Bubble Tea 状态机。输入条 + 可滚动主区；忙碌时用 bubbles/spinner 作 loading。
type model struct {
	ctx    context.Context
	agent  Agent
	uiText common.UIText
	vp     viewport.Model
	ti     textinput.Model
	spin   spinner.Model
	width  int
	height int
	busy   bool

	turnCh <-chan agent.AgentEvent

	blocks       []feedBlock
	modelIdx     int
	streaming    bool
	streamPrefix string

	sessionTokens    int64
	lastPromptToks   int
	lastCompletion   int
	contextPct       float64 // <0：不显示百分比

	suggestItems []agent.CompletionItem
	suggestIdx   int
}

func newModel(ctx context.Context, ag Agent, ui common.UIText) *model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = ui.InputPlaceholder
	ti.Focus()
	ti.CharLimit = 0

	vp := viewport.New(0, 0)
	vp.SetContent(ui.WelcomeMarkdown)

	return &model{
		ctx:    ctx,
		agent:  ag,
		uiText: ui,
		ti:     ti,
		vp:     vp,
		spin:   newSpinner(),
		modelIdx:   -1,
		contextPct: -1,
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.applyViewportHeight()
		m.ti.Width = msg.Width
		m.syncViewport()
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd

	case agentEventMsg:
		m.applyAgentEvent(msg.ev)
		m.syncViewport()
		return m, waitAgentEvent(m.turnCh)

	case turnDoneMsg:
		m.busy = false
		m.streaming = false
		m.streamPrefix = ""
		m.spin = newSpinner()
		return m, nil

	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		m.syncViewport()
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)

	default:
		return m, nil
	}
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	}
	if idx, ok := blockToggleIndex(msg, m.busy, m.ti.Value() == ""); ok {
		if idx < len(m.blocks) {
			m.blocks[idx].expanded = !m.blocks[idx].expanded
			m.syncViewport()
		}
		return m, nil
	}
	if m.busy {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		return m, cmd
	}
	if len(m.suggestItems) > 0 {
		switch msg.String() {
		case "esc":
			m.suggestItems = nil
			m.suggestIdx = 0
			m.applyViewportHeight()
			return m, nil
		case "enter":
			// 补全打开时原先把 Enter 交给 textinput，不会走到底部的 submit，导致无法发送（例如已输完 /model use <name>）。
			m.suggestItems = nil
			m.suggestIdx = 0
			m.applyViewportHeight()
			return m.submit()
		case "up":
			m.suggestIdx = (m.suggestIdx - 1 + len(m.suggestItems)) % len(m.suggestItems)
			return m, nil
		case "down":
			m.suggestIdx = (m.suggestIdx + 1) % len(m.suggestItems)
			return m, nil
		case "shift+tab":
			m.suggestIdx = (m.suggestIdx - 1 + len(m.suggestItems)) % len(m.suggestItems)
			return m, nil
		case "tab":
			it := m.suggestItems[m.suggestIdx]
			m.ti.SetValue(it.Insert)
			m.ti.CursorEnd()
			m.refreshCompletions()
			return m, nil
		default:
			var cmd tea.Cmd
			m.ti, cmd = m.ti.Update(msg)
			m.refreshCompletions()
			return m, cmd
		}
	}
	if msg.String() == "enter" {
		return m.submit()
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	m.refreshCompletions()
	return m, cmd
}

func (m *model) footerReserve() int {
	return baseFooterReserve + suggestExtraLines(len(m.suggestItems))
}

func (m *model) applyViewportHeight() {
	vpH := m.height - m.footerReserve()
	if vpH < 5 {
		vpH = 5
	}
	m.vp.Height = vpH
	m.vp.Width = m.width
}

func (m *model) refreshCompletions() {
	line := m.ti.Value()
	if !strings.HasPrefix(line, "/") {
		m.suggestItems = nil
		m.suggestIdx = 0
		m.applyViewportHeight()
		return
	}
	m.suggestItems = m.agent.Completions(line)
	if m.suggestIdx >= len(m.suggestItems) {
		m.suggestIdx = 0
	}
	m.applyViewportHeight()
}

func (m *model) submit() (tea.Model, tea.Cmd) {
	line := strings.TrimSpace(m.ti.Value())
	m.ti.SetValue("")
	m.ti.CursorEnd()
	m.refreshCompletions()
	if line == "" {
		return m, nil
	}
	if line == "quit" || line == "exit" || line == "/quit" {
		return m, tea.Quit
	}

	m.blocks = append(m.blocks, feedBlock{
		kind:     kindPrompt,
		status:   statusDone,
		body:     line,
		expanded: true,
		at:       time.Now(),
	})
	m.ensureActiveModelBlock()
	m.syncViewport()

	m.busy = true
	m.spin = newSpinner()
	m.turnCh = m.agent.RunTurn(m.ctx, line)
	return m, tea.Batch(waitAgentEvent(m.turnCh), func() tea.Msg { return m.spin.Tick() })
}

func (m *model) syncViewport() {
	w := m.width
	if w <= 0 {
		w = 80
	}
	spinView := ""
	if m.busy {
		spinView = m.spin.View()
	}
	feed := renderFeed(w, m.blocks, m.streaming, m.streamPrefix, m.uiText, spinView)
	m.vp.SetContent(feed)
	m.vp.GotoBottom()
}

func (m *model) View() string {
	if m.width == 0 {
		return m.uiText.ViewLoading
	}
	rule := strings.Repeat("─", m.width)
	var b strings.Builder
	b.WriteString(m.vp.View())
	b.WriteString("\n")
	b.WriteString(rule)
	b.WriteString("\n")
	if len(m.suggestItems) > 0 {
		panel := renderSuggestPanel(m.width, m.suggestItems, m.suggestIdx, m.uiText.CompletionHint)
		if panel != "" {
			b.WriteString(panel)
			b.WriteString("\n")
			b.WriteString(rule)
			b.WriteString("\n")
		}
	}
	b.WriteString(m.ti.View())
	b.WriteString("\n")
	b.WriteString(rule)
	b.WriteString("\n")
	ctxStr := "—"
	if m.contextPct >= 0 {
		ctxStr = fmt.Sprintf("%.1f%%", m.contextPct)
	}
	stats := fmt.Sprintf("%d tok · %d+%d · %s",
		m.sessionTokens, m.lastPromptToks, m.lastCompletion, ctxStr)
	help := m.uiText.HelpWithBlocks
	if len(m.blocks) == 0 {
		help = m.uiText.HelpNoBlocks
	}
	b.WriteString(renderFooterOneLine(m.width, help, stats))
	return b.String()
}

// renderFooterOneLine：左侧快捷键提示，右侧 token/context 统计，单行对齐。
func renderFooterOneLine(width int, helpPlain, statsPlain string) string {
	if width <= 0 {
		return ""
	}
	right := styleHelp.Render(statsPlain)
	rw := lipgloss.Width(right)
	maxLeft := width - rw - 1
	if maxLeft < 6 {
		maxLeft = 6
	}
	leftPlain := truncateRunes(helpPlain, maxLeft)
	left := styleHelp.Render(leftPlain)
	lw := lipgloss.Width(left)
	gap := width - lw - rw
	for gap < 0 && maxLeft > 4 {
		maxLeft--
		leftPlain = truncateRunes(helpPlain, maxLeft)
		left = styleHelp.Render(leftPlain)
		lw = lipgloss.Width(left)
		gap = width - lw - rw
	}
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

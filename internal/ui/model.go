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

// footerLines：底栏占用行数（分隔线、输入、分隔线、用量行、帮助），用于计算 viewport 高度。
const footerLines = 7

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
		vpH := msg.Height - footerLines
		if vpH < 5 {
			vpH = 5
		}
		m.vp.Width = msg.Width
		m.vp.Height = vpH
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
	if msg.String() == "enter" {
		return m.submit()
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m *model) submit() (tea.Model, tea.Cmd) {
	line := strings.TrimSpace(m.ti.Value())
	m.ti.SetValue("")
	if line == "" {
		return m, nil
	}
	if line == "quit" || line == "exit" {
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
	b.WriteString(m.ti.View())
	b.WriteString("\n")
	b.WriteString(rule)
	b.WriteString("\n")
	ctxStr := "—"
	if m.contextPct >= 0 {
		// 窗口很大（如 200k）时整数百分比长期为 0%/1%，保留两位小数便于观察
		ctxStr = fmt.Sprintf("%.2f%%", m.contextPct)
	}
	stats := fmt.Sprintf("session %d tok · last %d+%d · ctx %s",
		m.sessionTokens, m.lastPromptToks, m.lastCompletion, ctxStr)
	b.WriteString(styleHelp.Render(stats))
	b.WriteString("\n")
	help := m.uiText.HelpWithBlocks
	if len(m.blocks) == 0 {
		help = m.uiText.HelpNoBlocks
	}
	b.WriteString(styleHelp.Render(help))
	return b.String()
}

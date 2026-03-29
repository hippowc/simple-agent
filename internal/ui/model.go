package ui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"simple-agent/internal/agent"
)

const welcomeText = "Ready.\n\nSend a message or a /command. Scroll with wheel or PgUp/PgDn."

// footerLines：底栏占用行数（分隔线、输入、分隔线、帮助），用于计算 viewport 高度。
const footerLines = 6

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

// model：Bubble Tea 状态机。输入条 + 可滚动主区；业务数据主要是 blocks 与 LLM 流式缓冲。
type model struct {
	ctx             context.Context
	agent           Agent
	vp              viewport.Model
	ti              textinput.Model
	width           int
	height          int
	busy            bool
	llmRunningTitle string

	turnCh <-chan agent.AgentEvent

	blocks       []feedBlock
	modelIdx     int  // 当前打开的 kindModel 块下标；-1 表示无
	streaming    bool // LLM 流式片段是否写入 streamPrefix
	streamPrefix string
}

func newModel(ctx context.Context, ag Agent, llmRunningTitle string) *model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "Message…  (/tools)  Enter send · Ctrl+C quit"
	ti.Focus()
	ti.CharLimit = 0

	vp := viewport.New(0, 0)
	vp.SetContent(welcomeText)

	if llmRunningTitle == "" {
		llmRunningTitle = "Generating…"
	}
	return &model{
		ctx:             ctx,
		agent:           ag,
		ti:              ti,
		vp:              vp,
		modelIdx:        -1,
		llmRunningTitle: llmRunningTitle,
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
		return m, nil

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
	m.turnCh = m.agent.RunTurn(m.ctx, line)
	return m, waitAgentEvent(m.turnCh)
}

func (m *model) syncViewport() {
	w := m.width
	if w <= 0 {
		w = 80
	}
	streaming := m.streaming && m.streamPrefix != ""
	feed := renderFeed(w, m.blocks, welcomeText, streaming, m.streamPrefix, m.llmRunningTitle)
	s := LogoHeader(w) + "\n\n" + feed
	m.vp.SetContent(s)
	m.vp.GotoBottom()
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading…"
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
	help := "Enter send · wheel scroll · Alt+1-9 toggle block · 1-9 when busy or input empty · Ctrl+C quit"
	if len(m.blocks) == 0 {
		help = "Enter send · wheel scroll · Ctrl+C quit"
	}
	b.WriteString(styleHelp.Render(help))
	return b.String()
}

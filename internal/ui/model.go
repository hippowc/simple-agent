package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"simple-agent/internal/agent"
)

const welcomeText = "Ready.\n\nSend a message or a /command. Scroll with wheel or PgUp/PgDn."

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

var (
	styleHelp = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type model struct {
	ctx    context.Context
	agent  Agent
	vp     viewport.Model
	ti     textinput.Model
	width  int
	height int
	busy   bool

	turnCh <-chan agent.AgentEvent

	blocks       []feedBlock
	modelIdx     int
	streaming    bool
	streamPrefix string
}

func newModel(ctx context.Context, ag Agent) *model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "Message…  (/tools)  Enter send · Ctrl+C quit"
	ti.Focus()
	ti.CharLimit = 0

	vp := viewport.New(0, 0)
	vp.SetContent(welcomeText)

	return &model{
		ctx:      ctx,
		agent:    ag,
		ti:       ti,
		vp:       vp,
		modelIdx: -1,
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		const fixedLines = 6
		vpH := msg.Height - fixedLines
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
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
		if m.busy {
			s := msg.String()
			if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
				idx := int(s[0] - '1')
				if idx < len(m.blocks) {
					m.blocks[idx].expanded = !m.blocks[idx].expanded
					m.syncViewport()
				}
				return m, nil
			}
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

	default:
		return m, nil
	}
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
		expanded: false,
		at:       time.Now(),
	})
	m.ensureActiveModelBlock()
	m.syncViewport()

	m.busy = true
	m.turnCh = m.agent.RunTurn(m.ctx, line)
	return m, waitAgentEvent(m.turnCh)
}

func (m *model) ensureActiveModelBlock() {
	need := m.modelIdx < 0
	if !need && m.modelIdx < len(m.blocks) {
		b := m.blocks[m.modelIdx]
		need = b.kind != kindModel || b.status == statusDone
	}
	if need {
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindModel,
			status:   statusRunning,
			body:     "",
			expanded: true,
			at:       time.Now(),
		})
		m.modelIdx = len(m.blocks) - 1
	}
}

func (m *model) flushStreamToModel() {
	if !m.streaming {
		return
	}
	m.ensureActiveModelBlock()
	m.blocks[m.modelIdx].body = m.streamPrefix
	m.blocks[m.modelIdx].status = statusDone
	m.modelIdx = -1
	m.streaming = false
	m.streamPrefix = ""
}

func (m *model) applyAgentEvent(ev agent.AgentEvent) {
	switch ev.Kind {
	case agent.EventKindLLM:
		if ev.Partial {
			m.ensureActiveModelBlock()
			if !m.streaming {
				m.streamPrefix = ""
				m.streaming = true
			}
			m.streamPrefix += ev.Text
			return
		}
		m.ensureActiveModelBlock()
		if m.streaming {
			m.blocks[m.modelIdx].body = m.streamPrefix + ev.Text
			m.streaming = false
			m.streamPrefix = ""
		} else {
			m.blocks[m.modelIdx].body = ev.Text
		}
		m.blocks[m.modelIdx].status = statusDone
		m.modelIdx = -1

	case agent.EventKindTool:
		m.flushStreamToModel()
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindTool,
			title:    ev.ToolName,
			status:   statusDone,
			body:     ev.Detail,
			expanded: false,
			at:       time.Now(),
		})
		m.modelIdx = -1

	case agent.EventKindInfo:
		m.flushStreamToModel()
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindInfo,
			status:   statusDone,
			body:     ev.Text,
			expanded: false,
			at:       time.Now(),
		})

	case agent.EventKindError:
		m.flushStreamToModel()
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindError,
			status:   statusError,
			body:     ev.Detail,
			expanded: true,
			at:       time.Now(),
		})
		m.modelIdx = -1

	default:
		m.flushStreamToModel()
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindInfo,
			status:   statusDone,
			body:     fmt.Sprintf("%+v", ev),
			expanded: false,
			at:       time.Now(),
		})
	}
}

func (m *model) syncViewport() {
	w := m.width
	if w <= 0 {
		w = 80
	}
	streaming := m.streaming && m.streamPrefix != ""
	s := renderFeed(w, m.blocks, welcomeText, streaming, m.streamPrefix)
	m.vp.SetContent(s)
	m.vp.GotoBottom()
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	rule := strings.Repeat("─", m.width)
	var b strings.Builder
	b.WriteString(LogoHeader(m.width))
	b.WriteString("\n\n")
	b.WriteString(m.vp.View())
	b.WriteString("\n")
	b.WriteString(rule)
	b.WriteString("\n")
	b.WriteString(m.ti.View())
	b.WriteString("\n")
	b.WriteString(rule)
	b.WriteString("\n")
	help := "Enter send · wheel scroll · 1-9 toggle section · Ctrl+C quit"
	if len(m.blocks) == 0 {
		help = "Enter send · wheel scroll · Ctrl+C quit"
	}
	b.WriteString(styleHelp.Render(help))
	return b.String()
}

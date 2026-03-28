package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"simple-agent/internal/agent"
	"simple-agent/internal/common"
)

// agentInitMsg 异步初始化 Agent 完成后的消息（由 appModel 处理）。
type agentInitMsg struct {
	Agent *agent.Agent
	Err   error
}

type bootTickMsg struct{}

// bootStartLoadMsg 在首帧绘制之后再触发 Agent 初始化，避免与首屏竞态导致进度界面被直接跳过。
type bootStartLoadMsg struct{}

type appModel struct {
	ctx   context.Context
	cfg   common.Config
	child tea.Model
	win   tea.WindowSizeMsg
}

func (a *appModel) Init() tea.Cmd {
	return a.child.Init()
}

func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case agentInitMsg:
		if msg.Err != nil {
			fmt.Fprintf(os.Stderr, "init agent failed: %v\n", msg.Err)
			return a, tea.Quit
		}
		a.child = newModel(a.ctx, msg.Agent, a.cfg.UI.LLMRunningTitle)
		var cmds []tea.Cmd
		cmds = append(cmds, a.child.Init())
		if a.win.Width > 0 {
			w := a.win
			cmds = append(cmds, func() tea.Msg { return w })
		}
		return a, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		a.win = msg
	}

	var cmd tea.Cmd
	a.child, cmd = a.child.Update(msg)
	return a, cmd
}

func (a *appModel) View() string {
	return a.child.View()
}

type bootModel struct {
	cfg    common.Config
	width  int
	height int
	prog   progress.Model
}

func newBootModel(cfg common.Config) *bootModel {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
		progress.WithWidth(40),
	)
	return &bootModel{cfg: cfg, prog: p}
}

func (b *bootModel) Init() tea.Cmd {
	// 勿在 Init 里立刻启动 agent.NewFromConfig：Cmd 会很快返回 agentInitMsg，
	// 首屏往往尚未绘制，用户看不到进度条。延后一帧再开始加载。
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg {
		return bootStartLoadMsg{}
	})
}

func (b *bootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case bootStartLoadMsg:
		return b, tea.Batch(
			b.prog.SetPercent(0.18),
			func() tea.Msg {
				ag, err := agent.NewFromConfig(b.cfg)
				return agentInitMsg{Agent: ag, Err: err}
			},
			tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return bootTickMsg{} }),
		)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return b, tea.Quit
		}
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.height = msg.Height
		w := msg.Width - 8
		if w > 72 {
			w = 72
		}
		if w < 20 {
			w = 20
		}
		b.prog.Width = w
		return b, nil

	case bootTickMsg:
		if b.prog.Percent() < 0.92 {
			c := b.prog.IncrPercent(0.035)
			return b, tea.Batch(
				c,
				tea.Tick(110*time.Millisecond, func(time.Time) tea.Msg { return bootTickMsg{} }),
			)
		}
		return b, tea.Tick(180*time.Millisecond, func(time.Time) tea.Msg { return bootTickMsg{} })

	case progress.FrameMsg:
		p, c := b.prog.Update(msg)
		b.prog = p.(progress.Model)
		return b, c
	}
	return b, nil
}

func (b *bootModel) View() string {
	w, h := b.width, b.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	logo := LogoHeader(w)
	bar := b.prog.View()
	hint := styleDim.Render("正在初始化，请稍候…")
	block := logo + "\n\n" + bar + "\n\n" + hint
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, block)
}

// Run 先全屏显示启动进度条并异步初始化 Agent，就绪后进入对话界面。
func Run(ctx context.Context, cfg common.Config, in io.Reader, out io.Writer) error {
	root := &appModel{
		ctx:   ctx,
		cfg:   cfg,
		child: newBootModel(cfg),
	}
	p := tea.NewProgram(root,
		tea.WithContext(ctx),
		tea.WithInput(in),
		tea.WithOutput(out),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}

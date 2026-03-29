package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"simple-agent/internal/common"
)

// 本文件：时间线视图的 lipgloss 样式与纯渲染函数（无 Tea 状态）。

var (
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleBody   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleLbl    = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	styleTS     = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	styleUserText = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))

	railUser    = lipgloss.NewStyle().Foreground(lipgloss.Color("60"))
	railModel   = lipgloss.NewStyle().Foreground(lipgloss.Color("79"))
	railModelHi = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	railTool    = lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	railInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	railErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	userOutline = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

)

func railStyle(kind blockKind, st blockStatus) lipgloss.Style {
	switch kind {
	case kindPrompt:
		return railUser
	case kindModel:
		if st == statusRunning {
			return railModelHi
		}
		return railModel
	case kindTool:
		return railTool
	case kindInfo:
		return railInfo
	case kindError:
		return railErr
	default:
		return railInfo
	}
}

func railMark(kind blockKind, st blockStatus) string {
	return railStyle(kind, st).Render("┃")
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

func formatExpandedBody(blk *feedBlock, contentW int, ui common.UIText) string {
	body := strings.TrimRight(blk.body, "\n")
	if body == "" {
		return ""
	}
	switch blk.kind {
	case kindTool:
		return formatToolBody(contentW, body, ui.ToolOutputHeader)
	case kindError:
		return formatErrorBody(contentW, body)
	default:
		return styleBody.Render(lipgloss.NewStyle().Width(contentW).Render(body))
	}
}

func formatToolBody(contentW int, body string, outputHeader string) string {
	var b strings.Builder
	b.WriteString(styleDim.Render(outputHeader))
	b.WriteString("\n")
	b.WriteString(styleBody.Render(lipgloss.NewStyle().Width(contentW).Render(body)))
	return b.String()
}

func formatErrorBody(contentW int, body string) string {
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return ""
	}
	first := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Width(contentW).Render(lines[0])
	if len(lines) == 1 {
		return first
	}
	rest := strings.Join(lines[1:], "\n")
	restR := styleBody.Render(lipgloss.NewStyle().Width(contentW).Render(rest))
	return first + "\n" + restR
}

func foldKeyHintBracket(blockIndex1 int, ui common.UIText) string {
	h := foldKeyHint(blockIndex1, ui)
	return styleDim.Render(" [" + h + "]")
}

func foldKeyHint(blockIndex1 int, ui common.UIText) string {
	if blockIndex1 >= 1 && blockIndex1 <= 9 {
		return fmt.Sprintf(ui.FoldExpandAltFmt, blockIndex1)
	}
	return ui.FoldExpandAltRange
}

func renderUserBlock(blk *feedBlock, width int) string {
	body := strings.TrimRight(blk.body, "\n")
	if body == "" {
		return "\n"
	}

	tsPart := styleTS.Render(tsFmt(blk.at))
	railPart := railUser.Render("┃")
	strip := tsPart + headerRailTimeGap + railPart
	stripW := lipgloss.Width(strip)
	padW := width - stripW
	if padW < 0 {
		padW = 0
	}

	var b strings.Builder
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(padW).Render(""),
		strip,
	))
	b.WriteString("\n")

	rightMargin := lipgloss.Width(headerRailTimeGap) + lipgloss.Width(railPart)
	avail := width - rightMargin - 4
	if avail < 12 {
		avail = width - rightMargin - 2
	}
	if avail < 8 {
		avail = 8
	}
	innerW := avail - 6
	if innerW < 8 {
		innerW = avail - 2
	}

	lines := strings.Split(body, "\n")
	var innerLines []string
	for _, line := range lines {
		if line == "" {
			innerLines = append(innerLines, lipgloss.NewStyle().Width(innerW).Render(""))
			continue
		}
		innerLines = append(innerLines, styleUserText.Width(innerW).Align(lipgloss.Right).Render(line))
	}
	inner := strings.Join(innerLines, "\n")
	boxed := userOutline.Render(inner)
	boxLines := strings.Split(boxed, "\n")

	for _, bl := range boxLines {
		left := width - rightMargin - lipgloss.Width(bl)
		if left < 0 {
			left = 0
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(left).Render(""),
			bl,
		))
		b.WriteString("\n")
	}
	return b.String()
}

func renderStandardBlock(blk *feedBlock, width int, blockIndex1 int, ui common.UIText, spinView string, streamGlobal bool) string {
	rail := railMark(blk.kind, blk.status)
	tsStyled := styleTS.Render(tsFmt(blk.at))

	var label string
	switch blk.kind {
	case kindTool:
		label = styleLbl.Render(toolFriendlyName(blk.title, ui.ToolDisplayNames))
	case kindInfo:
		label = styleLbl.Render(ui.LabelInfo)
	case kindError:
		label = styleLbl.Render(ui.LabelError)
	}

	var hb strings.Builder
	hb.WriteString(rail)
	hb.WriteString(headerRailTimeGap)
	hb.WriteString(tsStyled)
	if label != "" {
		hb.WriteString(" ")
		hb.WriteString(label)
	}

	var b strings.Builder
	b.WriteString(hb.String())
	b.WriteString("\n")

	contentW := width - 4
	if contentW < 16 {
		contentW = width
	}

	switch blk.kind {
	case kindTool:
		b.WriteString(renderToolOutput(blk, contentW, blockIndex1, spinView, ui))
	case kindModel:
		if blk.status == statusRunning && strings.TrimSpace(blk.body) == "" {
			if !streamGlobal && spinView != "" {
				b.WriteString(indentEachLine(styleDim.Render("  ")+spinView, 2))
				b.WriteString("\n")
			}
		} else if strings.TrimSpace(blk.body) != "" {
			b.WriteString(indentEachLine(formatExpandedBody(blk, contentW, ui), 2))
			b.WriteString("\n")
		}
	default:
		if strings.TrimSpace(blk.body) != "" {
			b.WriteString(indentEachLine(formatExpandedBody(blk, contentW, ui), 2))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

func renderToolLoading(blk *feedBlock, spinView string, ui common.UIText) string {
	name := toolFriendlyName(blk.title, ui.ToolDisplayNames)
	line := styleDim.Render("  ") + spinView + "  " + styleBody.Render(name)
	return indentEachLine(line, 2) + "\n"
}

func renderToolOutput(blk *feedBlock, contentW int, blockIndex1 int, spinView string, ui common.UIText) string {
	if blk.status == statusRunning && strings.TrimSpace(blk.body) == "" {
		return renderToolLoading(blk, spinView, ui)
	}
	body := strings.TrimRight(blk.body, "\n")
	if body == "" {
		return ""
	}
	fold := isToolFoldable(blk)
	if fold && !blk.expanded {
		br := foldKeyHintBracket(blockIndex1, ui)
		pm := contentW - lipgloss.Width(br) - 1
		if pm < 4 {
			pm = 4
		}
		line := styleDim.Render(oneLinePreview(body, pm)) + br
		return indentEachLine(line, 2) + "\n"
	}
	return indentEachLine(formatToolBody(contentW, body, ui.ToolOutputHeader), 2) + "\n"
}

func renderFeed(width int, blocks []feedBlock, streaming bool, streamBuf string, ui common.UIText, spinView string) string {
	if len(blocks) == 0 {
		return styleDim.Render(ui.WelcomeMarkdown)
	}
	var b strings.Builder
	for i := range blocks {
		blk := &blocks[i]
		idx1 := i + 1
		if blk.kind == kindPrompt {
			b.WriteString(renderUserBlock(blk, width))
		} else {
			b.WriteString(renderStandardBlock(blk, width, idx1, ui, spinView, streaming))
		}
	}
	if streaming {
		b.WriteString(railModelHi.Render("┃"))
		b.WriteString(headerRailTimeGap)
		if strings.TrimSpace(streamBuf) != "" {
			b.WriteString(styleBody.Render(streamBuf))
			b.WriteString(styleDim.Render("▌"))
		} else if spinView != "" {
			b.WriteString(spinView)
		}
		b.WriteString("\n")
	}
	return b.String()
}

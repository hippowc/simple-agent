package ui

import (
	"fmt"
	"strings"
	"time"

	"simple-agent/internal/agent"
)

// 本文件：将 agent.AgentEvent 转为时间线 feedBlock（及流式缓冲），与 Tea 消息循环解耦。

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
	m.blocks[m.modelIdx].expanded = true
	m.modelIdx = -1
	m.streaming = false
	m.streamPrefix = ""
}

func (m *model) finalizeEmptyRunningModel() {
	for i := len(m.blocks) - 1; i >= 0; i-- {
		b := &m.blocks[i]
		if b.kind == kindModel && b.status == statusRunning && strings.TrimSpace(b.body) == "" {
			b.body = ""
			b.status = statusDone
			b.expanded = true
			break
		}
	}
	m.modelIdx = -1
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
			if strings.TrimSpace(ev.Text) != "" {
				m.blocks[m.modelIdx].body = ev.Text
			} else {
				m.blocks[m.modelIdx].body = m.streamPrefix
			}
			m.streaming = false
			m.streamPrefix = ""
		} else {
			m.blocks[m.modelIdx].body = ev.Text
		}
		m.blocks[m.modelIdx].status = statusDone
		m.blocks[m.modelIdx].expanded = true
		m.modelIdx = -1

	case agent.EventKindToolStart:
		m.flushStreamToModel()
		m.finalizeEmptyRunningModel()
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindTool,
			title:    ev.ToolName,
			status:   statusRunning,
			body:     "",
			expanded: true,
			at:       time.Now(),
		})
		m.modelIdx = -1

	case agent.EventKindTool:
		m.flushStreamToModel()
		n := len(m.blocks)
		if n > 0 {
			last := &m.blocks[n-1]
			if last.kind == kindTool && last.status == statusRunning && last.title == ev.ToolName {
				last.body = ev.Detail
				last.status = statusDone
				last.expanded = defaultExpandedForTool(ev.Detail)
				m.modelIdx = -1
				return
			}
		}
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindTool,
			title:    ev.ToolName,
			status:   statusDone,
			body:     ev.Detail,
			expanded: defaultExpandedForTool(ev.Detail),
			at:       time.Now(),
		})
		m.modelIdx = -1

	case agent.EventKindInfo:
		m.flushStreamToModel()
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindInfo,
			status:   statusDone,
			body:     ev.Text,
			expanded: true,
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

	case agent.EventKindUsage:
		m.sessionTokens = ev.SessionTokenTotal
		m.lastPromptToks = ev.LastPromptTokens
		m.lastCompletion = ev.LastCompletionTokens
		m.contextPct = ev.ContextPercent
		return

	default:
		m.flushStreamToModel()
		body := fmt.Sprintf("%+v", ev)
		m.blocks = append(m.blocks, feedBlock{
			kind:     kindInfo,
			status:   statusDone,
			body:     body,
			expanded: true,
			at:       time.Now(),
		})
	}
}

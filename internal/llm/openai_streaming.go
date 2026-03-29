package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

// GenerateStreaming 实现 StreamingClient：SSE 流式请求，聚合为与非流式等价的 Response；
// 每个正文增量调用 onContent（可安全忽略空串）。
func (c *OpenAIClient) GenerateStreaming(ctx context.Context, req Request, onContent func(delta string) error) (Response, error) {
	if err := c.validateRequest(req); err != nil {
		return Response{}, err
	}
	model := c.resolveModel(req)
	payload := openAIChatStreamRequest{
		Model:    model,
		Messages: req.Messages,
		Stream:   true,
		Tools:    req.Tools,
		StreamOptions: &struct {
			IncludeUsage bool `json:"include_usage"`
		}{IncludeUsage: true},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}

	streamClient := &http.Client{
		Transport: c.httpClient.Transport,
		Timeout:   0,
	}
	if streamClient.Transport == nil {
		streamClient.Transport = http.DefaultTransport
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var apiErr openAIChatResponse
		if json.Unmarshal(b, &apiErr) == nil && apiErr.Error != nil && apiErr.Error.Message != "" {
			return Response{}, fmt.Errorf("openai error: %s", apiErr.Error.Message)
		}
		return Response{}, fmt.Errorf("openai request failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var content strings.Builder
	toolAcc := make(map[int]*streamToolAccum)
	var finishReason string
	var streamUsage Usage

	scanner := bufio.NewScanner(resp.Body)
	const maxToken = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxToken)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return Response{}, ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunkV2
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return Response{}, fmt.Errorf("decode stream chunk: %w", err)
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return Response{}, fmt.Errorf("openai error: %s", chunk.Error.Message)
		}
		if chunk.Usage != nil {
			streamUsage.PromptTokens = chunk.Usage.PromptTokens
			streamUsage.CompletionTokens = chunk.Usage.CompletionTokens
			streamUsage.TotalTokens = chunk.Usage.TotalTokens
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			finishReason = *ch.FinishReason
		}

		d := ch.Delta
		if d.Content != nil && *d.Content != "" {
			content.WriteString(*d.Content)
			if err := onContent(*d.Content); err != nil {
				return Response{}, err
			}
		}
		for _, tc := range d.ToolCalls {
			mergeStreamToolDelta(toolAcc, tc)
		}
	}

	if err := scanner.Err(); err != nil {
		return Response{}, err
	}

	toolCalls := finalizeStreamTools(toolAcc)
	return Response{
		Content:      content.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        streamUsage,
	}, nil
}

// openAIChatStreamRequest 流式请求体（含 include_usage）。
type openAIChatStreamRequest struct {
	Model         string           `json:"model"`
	Messages      []Message        `json:"messages"`
	Stream        bool             `json:"stream"`
	Tools         []ToolDefinition `json:"tools,omitempty"`
	StreamOptions *struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options,omitempty"`
}

type streamToolAccum struct {
	id   string
	typ  string
	name strings.Builder
	args strings.Builder
}

func mergeStreamToolDelta(m map[int]*streamToolAccum, tc streamDeltaTool) {
	idx := tc.Index
	if m[idx] == nil {
		m[idx] = &streamToolAccum{}
	}
	a := m[idx]
	if tc.ID != "" {
		a.id = tc.ID
	}
	if tc.Type != "" {
		a.typ = tc.Type
	}
	if tc.Function.Name != "" {
		a.name.WriteString(tc.Function.Name)
	}
	if tc.Function.Arguments != "" {
		a.args.WriteString(tc.Function.Arguments)
	}
}

func finalizeStreamTools(m map[int]*streamToolAccum) []ToolCall {
	if len(m) == 0 {
		return nil
	}
	indices := make([]int, 0, len(m))
	for k := range m {
		indices = append(indices, k)
	}
	sort.Ints(indices)
	out := make([]ToolCall, 0, len(indices))
	for _, i := range indices {
		a := m[i]
		typ := a.typ
		if typ == "" {
			typ = "function"
		}
		out = append(out, ToolCall{
			ID:   a.id,
			Type: typ,
			Function: FunctionCall{
				Name:      a.name.String(),
				Arguments: a.args.String(),
			},
		})
	}
	return out
}

// openAIStreamChunkV2 流式 completion 单条 data（含 tool_calls delta；末包可仅含 usage）。
type openAIStreamChunkV2 struct {
	Choices []struct {
		Delta streamDeltaV2 `json:"delta"`
		// FinishReason 在末包给出，如 "stop" / "tool_calls"
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type streamDeltaV2 struct {
	Role      string            `json:"role"`
	Content   *string           `json:"content"`
	ToolCalls []streamDeltaTool `json:"tool_calls"`
}

type streamDeltaTool struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

var _ StreamingClient = (*OpenAIClient)(nil)

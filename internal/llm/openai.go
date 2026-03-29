package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"simple-agent/internal/common"
)

type OpenAIClient struct {
	baseURL      string
	apiKey       string
	defaultModel string
	httpClient   *http.Client
}

func NewOpenAIClient(cfg common.LLMConfig) *OpenAIClient {
	return &OpenAIClient{
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:       cfg.APIKey,
		defaultModel: cfg.Model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type openAIChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Stream   bool             `json:"stream,omitempty"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		FinishReason string  `json:"finish_reason"`
		Message      Message `json:"message"`
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

func (c *OpenAIClient) Generate(ctx context.Context, req Request) (Response, error) {
	if err := c.validateRequest(req); err != nil {
		return Response{}, err
	}
	model := c.resolveModel(req)

	payload := openAIChatRequest{
		Model:    model,
		Messages: req.Messages,
		Tools:    req.Tools,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	var result openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Response{}, err
	}

	if resp.StatusCode >= 400 {
		if result.Error != nil && result.Error.Message != "" {
			return Response{}, fmt.Errorf("openai error: %s", result.Error.Message)
		}
		return Response{}, fmt.Errorf("openai request failed: %s", resp.Status)
	}
	if len(result.Choices) == 0 {
		return Response{}, errors.New("openai response has no choices")
	}

	ch := result.Choices[0]
	var u Usage
	if result.Usage != nil {
		u.PromptTokens = result.Usage.PromptTokens
		u.CompletionTokens = result.Usage.CompletionTokens
		u.TotalTokens = result.Usage.TotalTokens
	}
	return Response{
		Content:      ch.Message.Content,
		ToolCalls:    ch.Message.ToolCalls,
		FinishReason: ch.FinishReason,
		Usage:        u,
	}, nil
}

func (c *OpenAIClient) GenerateStream(ctx context.Context, req Request) (<-chan StreamChunk, error) {
	if err := c.validateRequest(req); err != nil {
		return nil, err
	}

	out := make(chan StreamChunk, 32)
	go c.streamChatCompletions(ctx, req, out)
	return out, nil
}

func (c *OpenAIClient) validateRequest(req Request) error {
	if c.apiKey == "" {
		return errors.New("openai api key is empty")
	}
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}
	if model == "" {
		return errors.New("model is empty")
	}
	return nil
}

func (c *OpenAIClient) resolveModel(req Request) string {
	if req.Model != "" {
		return req.Model
	}
	return c.defaultModel
}

func (c *OpenAIClient) streamChatCompletions(ctx context.Context, req Request, out chan<- StreamChunk) {
	defer close(out)

	model := c.resolveModel(req)
	payload := openAIChatRequest{
		Model:    model,
		Messages: req.Messages,
		Stream:   true,
		Tools:    req.Tools,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		out <- StreamChunk{Err: err}
		return
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
		out <- StreamChunk{Err: err}
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := streamClient.Do(httpReq)
	if err != nil {
		out <- StreamChunk{Err: err}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var apiErr openAIChatResponse
		if json.Unmarshal(b, &apiErr) == nil && apiErr.Error != nil && apiErr.Error.Message != "" {
			out <- StreamChunk{Err: fmt.Errorf("openai error: %s", apiErr.Error.Message)}
			return
		}
		out <- StreamChunk{Err: fmt.Errorf("openai request failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	const maxToken = 1024 * 1024
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxToken)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			out <- StreamChunk{Err: ctx.Err()}
			return
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
			return
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			out <- StreamChunk{Err: fmt.Errorf("decode stream chunk: %w", err)}
			return
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			out <- StreamChunk{Err: fmt.Errorf("openai error: %s", chunk.Error.Message)}
			return
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		out <- StreamChunk{Text: delta}
	}

	if err := scanner.Err(); err != nil {
		out <- StreamChunk{Err: err}
	}
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

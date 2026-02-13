package aiagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ollamaModel implements the model.LLM interface for Ollama via the
// OpenAI-compatible chat completions API (/v1/chat/completions).
type ollamaModel struct {
	modelName string
	endpoint  string // e.g. "http://localhost:11434"
	client    *http.Client
}

// NewOllamaModel creates a new model.LLM backed by a local Ollama instance.
func NewOllamaModel(modelName, endpoint string) model.LLM {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	return &ollamaModel{
		modelName: modelName,
		endpoint:  endpoint,
		client:    &http.Client{},
	}
}

func (m *ollamaModel) Name() string {
	return m.modelName
}

// GenerateContent translates the ADK LLMRequest into an OpenAI-compatible
// request, sends it to Ollama, and returns the response as an LLMResponse.
func (m *ollamaModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.doGenerate(ctx, req)
		yield(resp, err)
	}
}

// ---------- OpenAI-compatible types ----------

type oaiMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCalls  []oaiToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type oaiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function oaiToolFunction `json:"function"`
}

type oaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaiTool struct {
	Type     string              `json:"type"`
	Function oaiToolFunctionDecl `json:"function"`
}

type oaiToolFunctionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters,omitempty"`
}

type oaiRequest struct {
	Model       string       `json:"model"`
	Messages    []oaiMessage `json:"messages"`
	Tools       []oaiTool    `json:"tools,omitempty"`
	Temperature *float32     `json:"temperature,omitempty"`
	Stream      bool         `json:"stream"`
}

type oaiResponse struct {
	Choices []oaiChoice `json:"choices"`
	Usage   *oaiUsage   `json:"usage,omitempty"`
}

type oaiChoice struct {
	Index        int        `json:"index"`
	Message      oaiMessage `json:"message"`
	FinishReason string     `json:"finish_reason"`
}

type oaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ---------- Translation helpers ----------

// contentsToMessages converts genai.Content list to OpenAI messages.
func contentsToMessages(contents []*genai.Content, config *genai.GenerateContentConfig) []oaiMessage {
	var msgs []oaiMessage

	// Add system instruction if present
	if config != nil && config.SystemInstruction != nil {
		text := extractText(config.SystemInstruction)
		if text != "" {
			msgs = append(msgs, oaiMessage{Role: "system", Content: text})
		}
	}

	for _, c := range contents {
		if c == nil {
			continue
		}
		role := mapRole(string(c.Role))

		// Check if this content has function call parts
		for _, part := range c.Parts {
			if part.FunctionCall != nil {
				// Model made a function call — emit as assistant with tool_calls
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				msgs = append(msgs, oaiMessage{
					Role: "assistant",
					ToolCalls: []oaiToolCall{
						{
							ID:   part.FunctionCall.ID,
							Type: "function",
							Function: oaiToolFunction{
								Name:      part.FunctionCall.Name,
								Arguments: string(argsJSON),
							},
						},
					},
				})
			} else if part.FunctionResponse != nil {
				// Tool response
				respJSON, _ := json.Marshal(part.FunctionResponse.Response)
				msgs = append(msgs, oaiMessage{
					Role:       "tool",
					Content:    string(respJSON),
					ToolCallID: part.FunctionResponse.ID,
				})
			} else if part.Text != "" {
				msgs = append(msgs, oaiMessage{Role: role, Content: part.Text})
			}
		}
	}

	return msgs
}

func mapRole(role string) string {
	switch role {
	case "model":
		return "assistant"
	case "user":
		return "user"
	case "system":
		return "system"
	default:
		return role
	}
}

func extractText(c *genai.Content) string {
	if c == nil {
		return ""
	}
	var text string
	for _, p := range c.Parts {
		if p.Text != "" {
			if text != "" {
				text += "\n"
			}
			text += p.Text
		}
	}
	return text
}

// configToTools converts genai Tools/FunctionDeclarations to OpenAI tool format.
func configToTools(config *genai.GenerateContentConfig) []oaiTool {
	if config == nil {
		return nil
	}
	var tools []oaiTool
	for _, t := range config.Tools {
		for _, fd := range t.FunctionDeclarations {
			params := schemaToMap(fd.Parameters)
			tools = append(tools, oaiTool{
				Type: "function",
				Function: oaiToolFunctionDecl{
					Name:        fd.Name,
					Description: fd.Description,
					Parameters:  params,
				},
			})
		}
	}
	return tools
}

// schemaToMap converts a genai.Schema to a JSON-schema-compatible map.
func schemaToMap(s *genai.Schema) map[string]any {
	if s == nil {
		return map[string]any{"type": "object", "properties": map[string]any{}}
	}
	m := map[string]any{
		"type": string(s.Type),
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if len(s.Properties) > 0 {
		props := make(map[string]any)
		for name, prop := range s.Properties {
			props[name] = schemaToMap(prop)
		}
		m["properties"] = props
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	if s.Items != nil {
		m["items"] = schemaToMap(s.Items)
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	return m
}

// oaiResponseToLLM converts the OpenAI response to an ADK LLMResponse.
func oaiResponseToLLM(resp *oaiResponse) *model.LLMResponse {
	if len(resp.Choices) == 0 {
		return &model.LLMResponse{
			TurnComplete: true,
			FinishReason: genai.FinishReasonStop,
		}
	}

	choice := resp.Choices[0]
	content := &genai.Content{Role: "model"}

	// Handle tool calls
	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]any
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: args,
				},
			})
		}
	} else if choice.Message.Content != "" {
		content.Parts = append(content.Parts, genai.NewPartFromText(choice.Message.Content))
	}

	llmResp := &model.LLMResponse{
		Content:      content,
		TurnComplete: true,
	}

	switch choice.FinishReason {
	case "stop":
		llmResp.FinishReason = genai.FinishReasonStop
	case "tool_calls":
		llmResp.FinishReason = genai.FinishReasonStop
		llmResp.TurnComplete = false
	}

	if resp.Usage != nil {
		llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(resp.Usage.PromptTokens),
			CandidatesTokenCount: int32(resp.Usage.CompletionTokens),
			TotalTokenCount:      int32(resp.Usage.TotalTokens),
		}
	}

	return llmResp
}

// ---------- HTTP call ----------

func (m *ollamaModel) doGenerate(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	// Ensure there's at least one user message
	if len(req.Contents) == 0 {
		req.Contents = append(req.Contents, genai.NewContentFromText("Handle the requests as specified in the System Instruction.", "user"))
	}
	if last := req.Contents[len(req.Contents)-1]; last != nil && string(last.Role) != "user" && string(last.Role) != "tool" {
		req.Contents = append(req.Contents, genai.NewContentFromText("Continue processing previous requests as instructed. Exit or provide a summary if no more outputs are needed.", "user"))
	}

	messages := contentsToMessages(req.Contents, req.Config)
	tools := configToTools(req.Config)

	oaiReq := oaiRequest{
		Model:    m.modelName,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
	}

	if req.Config != nil && req.Config.Temperature != nil {
		oaiReq.Temperature = req.Config.Temperature
	}

	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	url := m.endpoint + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var oaiResp oaiResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return oaiResponseToLLM(&oaiResp), nil
}

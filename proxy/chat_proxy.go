package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"gopenbridge/config"
)

// ContentBlock represents a text block.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolUseBlock represents a function call request.
type ToolUseBlock struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResultBlock represents a function call result.
type ToolResultBlock struct {
	Type      string      `json:"type"`
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content"`
}

// Message is an incoming or outgoing message.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// Tool describes a function to expose.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// MessagesRequest is the expected request payload.
type MessagesRequest struct {
	Model       string      `json:"model"`
	Messages    []Message   `json:"messages"`
	MaxTokens   *int        `json:"max_tokens,omitempty"`
	Temperature *float64    `json:"temperature,omitempty"`
	Stream      *bool       `json:"stream,omitempty"`
	Tools       []Tool      `json:"tools,omitempty"`
	ToolChoice  interface{} `json:"tool_choice,omitempty"`
}

// ChatProxy handles Anthropic-style payloads and forwards to OpenAI.
type ChatProxy struct {
	cfg *config.Config
}

// NewChatProxy constructs a ChatProxy.
func NewChatProxy(cfg *config.Config) *ChatProxy {
	return &ChatProxy{cfg: cfg}
}

// ServeHTTP satisfies http.Handler.
func (p *ChatProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req MessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	res, err := p.processRequest(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// maskAPIKey obfuscates an API key by showing only its start and end.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// processRequest converts and forwards the request.
func (p *ChatProxy) processRequest(req *MessagesRequest) (map[string]interface{}, error) {
	// Convert messages and tools
	msgs := convertMessages(req.Messages)
	var funcs []map[string]interface{}
	if len(req.Tools) > 0 {
		funcs = convertTools(req.Tools)
	}
	// Determine max tokens
	maxT := p.cfg.MaxTokens
	if req.MaxTokens != nil && *req.MaxTokens < maxT {
		maxT = *req.MaxTokens
	}
	// Build payload
	payload := map[string]interface{}{
		"model":       req.Model,
		"messages":    msgs,
		"temperature": req.Temperature,
		"max_tokens":  maxT,
	}
	if len(funcs) > 0 {
		payload["functions"] = funcs
		payload["function_call"] = "auto"
	}
	// Marshal and send
	body, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, _ := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	httpRes, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpRes.Body.Close()
	data, _ := io.ReadAll(httpRes.Body)
	var ocRes map[string]interface{}
	if err := json.Unmarshal(data, &ocRes); err != nil {
		return nil, err
	}
	// Check for OpenAI API errors and log details
	if errRaw, exists := ocRes["error"]; exists {
		if errMap, ok := errRaw.(map[string]interface{}); ok {
			code := errMap["code"]
			msg := errMap["message"]
			errType := errMap["type"]
			log.Printf("ERROR: OpenAI API error code=%v type=%v message=%v", code, errType, msg)
			return nil, fmt.Errorf("OpenAI API error: %v", msg)
		}
		log.Printf("ERROR: OpenAI API error response: %v", errRaw)
		return nil, fmt.Errorf("OpenAI API error: %v", errRaw)
	}
	// Extract choice
	choices, _ := ocRes["choices"].([]interface{})
	var message map[string]interface{}
	if len(choices) > 0 {
		ch, _ := choices[0].(map[string]interface{})
		message, _ = ch["message"].(map[string]interface{})
	}
	// Build content blocks
	var content []interface{}
	stopReason := "end_turn"
	if fc, ok := message["function_call"].(map[string]interface{}); ok {
		// tool use
		args := map[string]interface{}{}
		if s, ok := fc["arguments"].(string); ok {
			json.Unmarshal([]byte(s), &args)
		}
		content = append(content, map[string]interface{}{ // tool_use block
			"type":  "tool_use",
			"id":    uuid.New().String()[:12],
			"name":  fc["name"],
			"input": args,
		})
		stopReason = "tool_use"
	} else {
		// text
		txt, _ := message["content"].(string)
		content = append(content, map[string]interface{}{ // text block
			"type": "text",
			"text": txt,
		})
	}
	// Assemble response
	usage := map[string]interface{}{
		"input_tokens":  ocRes["usage"].(map[string]interface{})["prompt_tokens"],
		"output_tokens": ocRes["usage"].(map[string]interface{})["completion_tokens"],
	}
	return map[string]interface{}{
		"id":            "msg_" + uuid.New().String()[:12],
		"model":         req.Model,
		"role":          "assistant",
		"type":          "message",
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage":         usage,
	}, nil
}

// convertMessages maps Anthropic payload to OpenAI messages.
func convertMessages(msgs []Message) []map[string]interface{} {
	var out []map[string]interface{}
	for _, msg := range msgs {
		switch c := msg.Content.(type) {
		case string:
			out = append(out, map[string]interface{}{"role": msg.Role, "content": c})
		case []interface{}:
			// collect text and tool_calls
			textAcc := ""
			var tcalls []map[string]interface{}
			var toolsRes []map[string]interface{}
			for _, blk := range c {
				b, ok := blk.(map[string]interface{})
				if !ok {
					continue
				}
				t, _ := b["type"].(string)
				switch t {
				case "text":
					if s, ok := b["text"].(string); ok {
						textAcc += s
					}
				case "tool_use":
					id, _ := b["id"].(string)
					name, _ := b["name"].(string)
					input := b["input"]
					args, _ := json.Marshal(input)
					tcalls = append(tcalls, map[string]interface{}{ // function call spec
						"id":   id,
						"type": "function",
						"function": map[string]interface{}{
							"name":      name,
							"arguments": string(args),
						},
					})
				case "tool_result":
					toolsRes = append(toolsRes, map[string]interface{}{ // tool response
						"role":         "tool",
						"content":      b["content"],
						"tool_call_id": b["tool_use_id"],
					})
				}
			}
			if textAcc != "" || len(tcalls) > 0 {
				entry := map[string]interface{}{"role": msg.Role, "content": textAcc}
				if len(tcalls) > 0 {
					entry["tool_calls"] = tcalls
				}
				out = append(out, entry)
			}
			out = append(out, toolsRes...)
		}
	}
	return out
}

// convertTools maps Tool definitions to OpenAI functions.
func convertTools(tools []Tool) []map[string]interface{} {
	var out []map[string]interface{}
	for _, t := range tools {
		out = append(out, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.InputSchema,
		})
	}
	return out
}

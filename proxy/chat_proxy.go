package proxy

import (
   "bytes"
   "database/sql"
   "encoding/json"
   "fmt"
   "io"
   "log"
   "net/http"
   "strings"
   "time"

   "github.com/google/uuid"
   _ "github.com/mattn/go-sqlite3"
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
   db  *sql.DB
}

// NewChatProxy constructs a ChatProxy.
// NewChatProxy constructs a ChatProxy with persistence initialized.
func NewChatProxy(cfg *config.Config) *ChatProxy {
   // Open SQLite database
   db, err := sql.Open("sqlite3", cfg.DBPath)
   if err != nil {
       log.Fatalf("Failed to open DB: %v", err)
   }
   // Enable SQLite WAL journaling and set synchronous to NORMAL for performance
   if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
       log.Printf("Failed to set journal_mode WAL: %v", err)
   }
   if _, err := db.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
       log.Printf("Failed to set synchronous NORMAL: %v", err)
   }
   // Create log table if not exists
   createTable := `CREATE TABLE IF NOT EXISTS api_logs (
       id TEXT PRIMARY KEY,
       timestamp DATETIME,
       provider TEXT,
       endpoint TEXT,
       model TEXT,
       request TEXT,
       response TEXT,
       status_code INTEGER,
       error_message TEXT,
       prompt_tokens INTEGER,
       completion_tokens INTEGER
   );`
   if _, err := db.Exec(createTable); err != nil {
       log.Fatalf("Failed to create table: %v", err)
   }
   return &ChatProxy{cfg: cfg, db: db}
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

// detectProvider determines the provider type from the base URL.
func detectProvider(baseURL string) string {
	baseURL = strings.ToLower(baseURL)

	// Check for specific provider patterns
	if strings.Contains(baseURL, "groq.com") {
		return "groq"
	}
	if strings.Contains(baseURL, "openrouter.ai") {
		return "openrouter"
	}
	if strings.Contains(baseURL, "api.openai.com") {
		return "openai"
	}
	if strings.Contains(baseURL, "fireworks.ai") {
		return "fireworks"
	}
	if strings.Contains(baseURL, "huggingface.co") {
		return "huggingface"
	}
	if strings.Contains(baseURL, "anthropic.com") {
		return "anthropic"
	}

	// Default to standard OpenAI-compatible format (tools)
	return "openai-compatible"
}

// processRequest converts and forwards the request.
func (p *ChatProxy) processRequest(req *MessagesRequest) (map[string]interface{}, error) {
   // Generate log ID
   logID := uuid.New().String()[:12]
   // Detect provider type
   provider := detectProvider(p.cfg.BaseURL)
   // Convert messages and tools
   msgs := convertMessages(req.Messages)
	var toolsOrFuncs []map[string]interface{}
	if len(req.Tools) > 0 {
		toolsOrFuncs = convertToolsForProvider(req.Tools, provider)
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
	// Add tools/functions based on provider
	if len(toolsOrFuncs) > 0 {
		switch provider {
		case "groq":
			// Groq uses legacy functions format
			payload["functions"] = toolsOrFuncs
			if req.ToolChoice != nil {
				payload["function_call"] = req.ToolChoice
			} else {
				payload["function_call"] = "auto"
			}
			if p.cfg.Debug {
				log.Printf("DEBUG: Using Groq functions format")
			}
		default:
			// OpenRouter, OpenAI, Fireworks, and most others use tools format
			payload["tools"] = toolsOrFuncs
			if req.ToolChoice != nil {
				payload["tool_choice"] = req.ToolChoice
			} else {
				payload["tool_choice"] = "auto"
			}
			if p.cfg.Debug {
				log.Printf("DEBUG: Using standard tools format for provider: %s", provider)
			}
		}
	}
	// Marshal and send
	body, _ := json.Marshal(payload)
	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + "/chat/completions"
	// Debug: log request payload
	if p.cfg.Debug {
		log.Printf("DEBUG: Request to %s: payload %s", endpoint, string(body))
	}
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
	// Debug: log response status and body
	if p.cfg.Debug {
		log.Printf("DEBUG: Response status %s body: %s", httpRes.Status, string(data))
	}
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

	// Detect tool invocation (try multiple formats)
	// 1. Modern tools format: tool_calls array (OpenRouter, OpenAI with tools)
	if toolCalls, ok := message["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
		if p.cfg.Debug {
			log.Printf("DEBUG: Detected tool_calls format (OpenRouter/OpenAI tools)")
		}
		for _, tc := range toolCalls {
			tcMap, _ := tc.(map[string]interface{})
			funcData, _ := tcMap["function"].(map[string]interface{})

			args := map[string]interface{}{}
			if s, ok := funcData["arguments"].(string); ok {
				json.Unmarshal([]byte(s), &args)
			}

			toolID, _ := tcMap["id"].(string)
			if toolID == "" {
				toolID = uuid.New().String()[:12]
			}

			content = append(content, map[string]interface{}{
				"type":  "tool_use",
				"id":    toolID,
				"name":  funcData["name"],
				"input": args,
			})
		}
		stopReason = "tool_use"
	} else {
		// 2. Legacy formats: function_call or tool (Groq, older OpenAI)
		var fc map[string]interface{}
		if raw, ok := message["function_call"].(map[string]interface{}); ok {
			if p.cfg.Debug {
				log.Printf("DEBUG: Detected function_call format (Groq/legacy)")
			}
			fc = raw
		} else if raw, ok := message["tool"].(map[string]interface{}); ok {
			if p.cfg.Debug {
				log.Printf("DEBUG: Detected tool format")
			}
			fc = raw
		}

		if fc != nil {
			// Single function/tool call
			args := map[string]interface{}{}
			if s, ok := fc["arguments"].(string); ok {
				json.Unmarshal([]byte(s), &args)
			}
			content = append(content, map[string]interface{}{
				"type":  "tool_use",
				"id":    uuid.New().String()[:12],
				"name":  fc["name"],
				"input": args,
			})
			stopReason = "tool_use"
		} else {
			// No tool calls - just text
			txt, _ := message["content"].(string)
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": txt,
			})
		}
	}
	// Assemble response
	usage := map[string]interface{}{
		"input_tokens":  ocRes["usage"].(map[string]interface{})["prompt_tokens"],
		"output_tokens": ocRes["usage"].(map[string]interface{})["completion_tokens"],
	}
	// Persist log entry
	ptF, _ := usage["input_tokens"].(float64)
	ctF, _ := usage["output_tokens"].(float64)
	_, errExec := p.db.Exec(
		`INSERT INTO api_logs(id, timestamp, provider, endpoint, model, request, response, status_code, error_message, prompt_tokens, completion_tokens) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		logID,
		time.Now().UTC(),
		p.cfg.BaseURL,
		endpoint,
		req.Model,
		string(body),
		string(data),
		httpRes.StatusCode,
		"", // no error message
		int(ptF),
		int(ctF),
	)
	if errExec != nil {
		log.Printf("Failed to persist API log: %v", errExec)
	}
	return map[string]interface{}{
		"id":            "msg_" + logID,
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

// convertToolsForProvider maps Tool definitions to provider-specific format.
func convertToolsForProvider(tools []Tool, provider string) []map[string]interface{} {
	var out []map[string]interface{}
	for _, t := range tools {
		switch provider {
		case "groq":
			// Groq uses legacy functions format: name, description, parameters
			out = append(out, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			})
		default:
			// OpenRouter, OpenAI, Fireworks use tools format with type and function wrapper
			out = append(out, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.InputSchema,
				},
			})
		}
	}
	return out
}

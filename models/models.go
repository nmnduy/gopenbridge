package models

// Define data models used by the application, ported from Python models.

// ContentBlock represents a basic text block.
type ContentBlock struct {
	Type string `json:"type" yaml:"type"`
	Text string `json:"text" yaml:"text"`
}

// ToolUseBlock represents a tool invocation with input parameters.
type ToolUseBlock struct {
	Type  string                 `json:"type" yaml:"type"`
	ID    string                 `json:"id" yaml:"id"`
	Name  string                 `json:"name" yaml:"name"`
	Input map[string]interface{} `json:"input" yaml:"input"`
}

// ToolResultBlock represents the result of a tool invocation.
type ToolResultBlock struct {
	Type      string      `json:"type" yaml:"type"`
	ToolUseID string      `json:"tool_use_id" yaml:"tool_use_id"`
	Content   interface{} `json:"content" yaml:"content"`
}

// Message represents a chat message with a role and content.
// Content may be a plain string or a list of blocks.
type Message struct {
	Role    string      `json:"role" yaml:"role"`
	Content interface{} `json:"content" yaml:"content"`
}

// Tool describes a callable tool with its schema.
type Tool struct {
	Name        string                 `json:"name" yaml:"name"`
	Description *string                `json:"description,omitempty" yaml:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema" yaml:"input_schema"`
}

// MessagesRequest models a request payload of chat messages.
type MessagesRequest struct {
	Model       string      `json:"model" yaml:"model"`
	Messages    []Message   `json:"messages" yaml:"messages"`
	MaxTokens   *int        `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	Temperature *float64    `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	Stream      *bool       `json:"stream,omitempty" yaml:"stream,omitempty"`
	Tools       []Tool      `json:"tools,omitempty" yaml:"tools,omitempty"`
	ToolChoice  interface{} `json:"tool_choice" yaml:"tool_choice"`
}

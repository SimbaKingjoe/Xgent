package llm

import (
	"context"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents an LLM response
type Response struct {
	Content      string
	FinishReason string
	Usage        Usage
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Client interface for LLM providers
type Client interface {
	Chat(ctx context.Context, messages []Message) (*Response, error)
	Stream(ctx context.Context, messages []Message, callback func(string) error) error
	Name() string
}

// Config for LLM client
type Config struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
}

package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GeminiClient implements Google Gemini API
type GeminiClient struct {
	model   string
	apiKey  string
	baseURL string
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(model, apiKey string) *GeminiClient {
	return &GeminiClient{
		model:   model,
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
	}
}

type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float32 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

// Chat sends a chat request to Gemini
func (c *GeminiClient) Chat(ctx context.Context, messages []Message) (*Response, error) {
	contents := convertToGeminiContents(messages)

	reqBody := geminiRequest{
		Contents: contents,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", c.baseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error: %d, %s", resp.StatusCode, string(body))
	}

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from Gemini")
	}

	return &Response{
		Content:      result.Candidates[0].Content.Parts[0].Text,
		FinishReason: result.Candidates[0].FinishReason,
	}, nil
}

// Stream streams chat responses from Gemini
func (c *GeminiClient) Stream(ctx context.Context, messages []Message, callback func(string) error) error {
	contents := convertToGeminiContents(messages)

	reqBody := geminiRequest{
		Contents: contents,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", c.baseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gemini API error: %d, %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")
		if jsonData == "" {
			continue
		}

		var chunk geminiResponse
		if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text
			if text != "" {
				if err := callback(text); err != nil {
					return err
				}
			}
		}
	}

	return scanner.Err()
}

// Name returns the client name
func (c *GeminiClient) Name() string {
	return "gemini:" + c.model
}

// convertToGeminiContents converts messages to Gemini format
func convertToGeminiContents(messages []Message) []geminiContent {
	var contents []geminiContent
	var systemPrompt string

	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		content := msg.Content
		if systemPrompt != "" && role == "user" {
			// Prepend system prompt to first user message
			content = systemPrompt + "\n\n" + content
			systemPrompt = ""
		}

		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: content}},
		})
	}

	return contents
}

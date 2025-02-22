// go-server/llm/client.go
package llm

import (
	"context"
	"fmt"
	"io"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// DefaultModel is the fallback model if none is specified.
const DefaultModel = "deepseek-r1-distill-qwen-7b"

// LLMClient defines the methods our API needs.
type LLMClient interface {
	GetCompletion(ctx context.Context, req CompletionRequest) (string, error)
	StreamCompletion(ctx context.Context, req CompletionRequest) (<-chan string, <-chan error)
}

// Client wraps the OpenAI API client for LLM interactions.
type Client struct {
	api   *openai.Client
	model string // Store selected model
}

// SetModel allows updating the model dynamically.
func (c *Client) SetModel(model string) {
	c.model = model
}

// CompletionRequest holds parameters for an LLM request.
type CompletionRequest struct {
	Model       string
	Messages    []openai.ChatCompletionMessage
	Temperature float32
	Stream      bool
}

// Message represents a message in a chat conversation (OpenAI format).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewClient initializes a new LLM client.
// If baseURL is non-empty, it overrides the default base URL.
func NewClient(apiKey, baseURL string) *Client {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	return &Client{
		api: openai.NewClientWithConfig(config),
	}
}

// GetCompletion performs a non-streaming completion.
func (c *Client) GetCompletion(ctx context.Context, req CompletionRequest) (string, error) {
	// Use the stored model if req.Model is empty.
	if req.Model == "" {
		req.Model = c.model
	}

	resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		Stream:      false,
	})
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("LLM response empty")
	}
	return resp.Choices[0].Message.Content, nil
}

// StreamCompletion streams an LLM response.
func (c *Client) StreamCompletion(ctx context.Context, req CompletionRequest) (<-chan string, <-chan error) {
	out := make(chan string)
	errs := make(chan error, 1)

	// Get model dynamically (hot-loads from env var)
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = req.Model // Fall back to request model
	}

	go func() {
		defer close(out)
		defer close(errs)

		stream, err := c.api.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:       model, // Always use latest model
			Messages:    req.Messages,
			Temperature: req.Temperature,
			Stream:      true,
		})
		if err != nil {
			errs <- fmt.Errorf("LLM streaming failed: %w", err)
			return
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				errs <- fmt.Errorf("LLM streaming error: %w", err)
				return
			}
			if len(response.Choices) > 0 {
				out <- response.Choices[0].Delta.Content
			}
		}
	}()

	return out, errs
}

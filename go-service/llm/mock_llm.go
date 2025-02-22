package llm

import (
	"context"
)

// mockLLMClient implements LLMClient.
type mockLLMClient struct{}

func (m *mockLLMClient) GetCompletion(ctx context.Context, req CompletionRequest) (string, error) {
	return "mocked response", nil
}

func (m *mockLLMClient) StreamCompletion(ctx context.Context, req CompletionRequest) (<-chan string, <-chan error) {
	out := make(chan string, 1)
	errs := make(chan error, 1)
	out <- "mocked streaming response"
	close(out)
	close(errs)
	return out, errs
}

// NewMockLLMClient returns a new instance of mockLLMClient.
func NewMockLLMClient() LLMClient {
	return &mockLLMClient{}
}

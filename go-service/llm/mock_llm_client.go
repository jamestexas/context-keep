package llm

import (
	"context"
)

type MockLLMClient struct{}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{}
}

func (m *MockLLMClient) GetCompletion(ctx context.Context, req CompletionRequest) (string, error) {
	return "Hello", nil
}

func (m *MockLLMClient) StreamCompletion(ctx context.Context, req CompletionRequest) (<-chan string, <-chan error) {
	out := make(chan string, 1)
	errs := make(chan error, 1)
	out <- "mocked streaming response"
	close(out)
	close(errs)
	return out, errs
}

// api/server_summary_test.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/jamestexas/context-keep/go-service/llm"
	redis_client "github.com/jamestexas/context-keep/go-service/redis"

	"github.com/stretchr/testify/assert"
)

// fakeRedisStore is a simple fake for redis.Storage.
type fakeRedisStore struct {
	summaryMap map[string]string
}

var _ redis_client.Storage = (*fakeRedisStore)(nil) // Ensure fakeRedisStore implements Storage

func (f *fakeRedisStore) GetSummary(ctx context.Context, convoID string) (string, error) {
	s, ok := f.summaryMap[convoID]
	if !ok {
		return "", redis.Nil // redis.Nil is a common way to indicate a missing key in Redis
	}
	return s, nil
}

func (f *fakeRedisStore) DeleteConversation(ctx context.Context, convoID string) error {
	delete(f.summaryMap, convoID)
	return nil
}

func (f *fakeRedisStore) DeleteSummary(ctx context.Context, convoID string) error {
	delete(f.summaryMap, convoID)
	return nil
}

func (f *fakeRedisStore) StoreEvent(ctx context.Context, convoID, eventID, parentID, eventText string, tags []string) error {
	return nil
}

func (f *fakeRedisStore) GetRecentEvents(ctx context.Context, convoID string, count int) ([]string, error) {
	return nil, nil
}

// Add the missing StoreSummary method to implement the Storage interface.
func (f *fakeRedisStore) StoreSummary(ctx context.Context, convoID, summary string) error {
	if f.summaryMap == nil {
		f.summaryMap = make(map[string]string)
	}
	f.summaryMap[convoID] = summary
	return nil
}
func (f *fakeRedisStore) GetEvent(ctx context.Context, convoID, eventID string) (redis_client.EventNode, error) {
	// For testing, we can return an empty EventNode or a stub.
	return redis_client.EventNode{}, errors.New("not implemented")
}

// Implement minimal methods needed by APIHandler.
// For the purpose of this test, you only need GetSummary.
// You might also need to stub StoreEvent, DeleteConversation etc. if you test those endpoints.

// TestHandleGetSummary_Success remains mostly the same.
func TestHandleGetSummary_Success(t *testing.T) {
	convoID := "convo1"
	summary := "This is a test summary"
	fakeStore := &fakeRedisStore{
		summaryMap: map[string]string{convoID: summary},
	}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/summary/%s", convoID), nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, convoID)) // Inject convoID into context
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, summary, resp["summary"])
}

func TestHandleGetSummary_MissingConvoID(t *testing.T) {
	fakeStore := &fakeRedisStore{summaryMap: map[string]string{}}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/summary", nil) // Updated route
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetSummary_NotFound(t *testing.T) {
	// Fake store does not have the summary for the given convoID.
	fakeStore := &fakeRedisStore{summaryMap: map[string]string{}}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/summary/convo2", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, "convo2")) // Inject convoID into context
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	// Expect a 404 Not Found error.
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

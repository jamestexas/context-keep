// go-service/api/server_test.go

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jamestexas/context-keep/go-service/llm"
	"github.com/jamestexas/context-keep/go-service/redis"
	"github.com/stretchr/testify/assert"
)

func TestHandleRoot(t *testing.T) {
	redisStore := redis.NewRedisStore("localhost:6379")
	// Use our mock that now satisfies llm.LLMClient.
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(redisStore, llmClient)

	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Welcome to Context-Keep API!", resp["message"])
}

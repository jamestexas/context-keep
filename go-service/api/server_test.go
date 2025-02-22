// go-service/api/server_test.go

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jamestexas/context-keep/go-service/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	// Import redis package
)

// SummarizeRequest struct
type SummarizeRequest = StreamRequest

func TestHandleRoot(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
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

func TestHandleStoreEvent(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	reqBody := `{"convo_id": "convo1", "event_text": "Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/store", http.NoBody)
	req.Body = ioutil.NopCloser(strings.NewReader(reqBody))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Stored successfully", resp["message"])
}

func TestHandleStoreEvent_InvalidRequest(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodPost, "/store", strings.NewReader("invalid"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleDebugConversation(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/debug/conversation/convo1", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, "convo1"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleDebugConversation_MissingConvoID(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/debug/conversation", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleChat(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	reqBody := `{"convo_id": "convo1", "event_text": "Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, "convo1"))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", resp["response"])
}

func TestHandleChatStream(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	reqBody := `{"convo_id": "convo1", "event_text": "Hello"}`
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, "convo1"))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "mocked streaming response")
}

func TestHandleChat_InvalidRequest(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("invalid"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleDeleteConversation(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodDelete, "/conversation/convo1", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, "convo1"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Conversation deleted", resp["message"])
}

func TestHandleDeleteConversation_MissingConvoID(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodDelete, "/conversation", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleDeleteSummary(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodDelete, "/delete/summary/convo1", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, "convo1"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "Summary deleted", resp["message"])
}

func TestHandleDeleteSummary_MissingConvoID(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodDelete, "/delete/summary", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetHierarchicalMemory(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/conversation/convo1", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyConvoID, "convo1"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestHandleGetHierarchicalMemory_MissingConvoID(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/conversation", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleGetModel(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodGet, "/model", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, llm.DefaultModel, resp["model"])
}

func TestHandleSummarize(t *testing.T) {
	mockStore := new(MockStorage) // Use MockStorage
	mockLLM := llm.NewMockLLMClient()
	handler := NewAPIHandler(mockStore, mockLLM)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "successful summarization",
			body: SummarizeRequest{
				ConvoID:   "convo1",
				EventText: "test text", // Updated field name
			},
			setupMock: func() {
				mockStore.On("GetSummary", mock.Anything, "convo1").Return("", nil)
				mockStore.On("StoreSummary", mock.Anything, "convo1", mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request body",
			body:           "invalid",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "store error",
			body: SummarizeRequest{
				ConvoID:   "convo1",
				EventText: "test text", // Updated field name
			},
			setupMock: func() {
				mockStore.On("GetSummary", mock.Anything, "convo1").Return("", nil)
				mockStore.On("StoreSummary", mock.Anything, "convo1", mock.Anything).Return(errors.New("store error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore.ExpectedCalls = nil
			tt.setupMock()

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/summarize", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.HandleSummarize(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockStore.AssertExpectations(t)
		})
	}
}

func TestHandleSummarize_InvalidRequest(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodPost, "/summarize", strings.NewReader("invalid"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleSummarizeStream(t *testing.T) {
	mockStore := new(MockStorage) // Use MockStorage
	mockLLM := llm.NewMockLLMClient()
	handler := NewAPIHandler(mockStore, mockLLM)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "successful summarize stream",
			body: SummarizeRequest{
				ConvoID:   "convo1",
				EventText: "test text", // Updated field name
			},
			setupMock: func() {
				// Expect GetSummary call (e.g. returning an existing summary)
				mockStore.On("GetSummary", mock.Anything, "convo1").Return("existing summary", nil)
				// Expect GetRecentEvents call with count=10
				mockStore.On("GetRecentEvents", mock.Anything, "convo1", 10).Return([]string{}, nil)
				// Expect StoreSummary call with the streaming response
				mockStore.On("StoreSummary", mock.Anything, "convo1", "mocked streaming response").Return(nil)
				// Expect StoreEvent call for the stream endpoint.
				mockStore.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(2)
			},

			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request body",
			body:           "invalid",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "store error",
			body: SummarizeRequest{
				ConvoID:   "convo1",
				EventText: "test text", // Updated field name
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore.ExpectedCalls = nil
			tt.setupMock()

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/summarize/stream", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.HandleSummarizeStream(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockStore.AssertExpectations(t)
		})
	}
}

func TestHandleSummarizeStream_InvalidRequest(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()
	handler := NewAPIHandler(fakeStore, llmClient)
	router := handler.Router()

	req := httptest.NewRequest(http.MethodPost, "/summarize", strings.NewReader("invalid"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandleChatStreamTest(t *testing.T) { // Renamed to avoid conflict
	mockStore := new(MockStorage) // Use MockStorage
	mockLLM := llm.NewMockLLMClient()
	handler := NewAPIHandler(mockStore, mockLLM)

	tests := []struct {
		name           string
		body           interface{}
		setupMock      func()
		expectedStatus int
	}{
		{
			name: "successful chat stream",
			body: ChatRequest{
				EventText: "test message", // Correct field name
			},
			setupMock: func() {
				mockStore.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid request body",
			body:           "invalid",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "store error",
			body: ChatRequest{
				EventText: "test message", // Correct field name
			},
			setupMock: func() {
				mockStore.On("StoreEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("store error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore.ExpectedCalls = nil
			tt.setupMock()

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/chat/stream", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handler.HandleChatStream(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockStore.AssertExpectations(t)
		})
	}
}

func TestStartServer(t *testing.T) {
	fakeStore := &fakeRedisStore{}
	llmClient := llm.NewMockLLMClient()

	go func() {
		StartServer(":8080", fakeStore, llmClient)
	}()

	resp, err := http.Get("http://localhost:8080/")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

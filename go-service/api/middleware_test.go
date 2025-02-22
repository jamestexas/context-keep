package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jamestexas/context-keep/go-service/llm"
	"github.com/stretchr/testify/assert"
)

// dummyHandler is a helper handler that writes out a JSON with a value from context.
func dummyHandler(key ContextKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the value from context and encode it.
		val, _ := r.Context().Value(key).(string)
		json.NewEncoder(w).Encode(map[string]string{"value": val})
	}
}

func TestConvoIDInjector_WithConvoID(t *testing.T) {
	// Create a chi router so that chi.URLParam works.
	r := chi.NewRouter()
	// Create a handler chain with the convoIDInjector middleware and our dummy handler.
	r.With(convoIDInjector).Get("/test/{convo_id}", dummyHandler(ContextKeyConvoID))

	// Create a test request with a URL that includes a convo_id.
	req := httptest.NewRequest(http.MethodGet, "/test/12345", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Check that the response contains the conversation ID from the URL.
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "12345", resp["value"])
}

func TestConvoIDInjector_WithoutConvoID(t *testing.T) {
	// Create a handler chain without a convo_id.
	r := chi.NewRouter()
	r.With(convoIDInjector).Get("/test", dummyHandler(ContextKeyConvoID))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Expect an empty string (or missing key) because no convo_id was in the URL.
	var resp map[string]string
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	// Here we expect that the middleware does not inject a value if not present.
	assert.Equal(t, "", resp["value"])
}

func TestModelInjector(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{"env not set", "", llm.DefaultModel},
		{"env set to custom", "custom-model", "custom-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore the original environment variable.
			original := os.Getenv("LLM_MODEL")
			os.Setenv("LLM_MODEL", tt.envValue)
			defer os.Setenv("LLM_MODEL", original)

			r := chi.NewRouter()
			r.With(modelInjector).Get("/test", dummyHandler(ContextKeyCurrentModel))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			var resp map[string]string
			err := json.NewDecoder(rr.Body).Decode(&resp)
			assert.NoError(t, err)

			assert.Equal(t, tt.expected, strings.TrimSpace(resp["value"]))
		})
	}
}

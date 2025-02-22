package api

import (
	"context"
	"net/http"
	"os"

	chi "github.com/go-chi/chi/v5"
	"github.com/jamestexas/context-keep/go-service/llm"
)

const ()

// ContextKey is a custom type for context keys.
type ContextKey string

const (
	ContextKeyConvoID      ContextKey = "convoID"      // ContextKeyConvoID is the context key for the conversation ID.
	ContextKeyCurrentModel ContextKey = "currentModel" // ContextKeyCurrentModel is the context key for the current model.

)

// convoIDInjector middleware extracts the convo_id URL parameter and adds it to the context.
func convoIDInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the convo_id parameter exists
		convoID := chi.URLParam(r, "convo_id")
		if convoID != "" {
			// Store it in the context
			ctx := context.WithValue(r.Context(), ContextKeyConvoID, convoID)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// modelInjector middleware adds the current model to the request context.
func modelInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		model := os.Getenv("LLM_MODEL")
		if model == "" {
			model = llm.DefaultModel
		}
		// Add model to context
		ctx := context.WithValue(r.Context(), ContextKeyCurrentModel, model)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package main

import (
	"context"
	"encoding/json"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	"github.com/jamestexas/context-keep/go-service/redis"
)

// APIHandler manages HTTP requests
type APIHandler struct {
	storage *redis.RedisStore
	memory  *redis.MemoryManager
}

// NewAPIHandler initializes APIHandler
func NewAPIHandler(storage *redis.RedisStore) *APIHandler {
	memManager := redis.NewMemoryManager(storage)
	return &APIHandler{storage: storage, memory: memManager}
}

// HandleGetHierarchicalMemory retrieves structured conversation history
func (h *APIHandler) HandleGetHierarchicalMemory(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	convoID := chi.URLParam(r, "convo_id") // Get convo ID from URL

	if convoID == "" {
		http.Error(w, "Missing convo_id", http.StatusBadRequest)
		return
	}

	memoryData, err := h.memory.GetHierarchicalMemory(ctx, convoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with JSON-encoded structured memory data
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(memoryData)
}

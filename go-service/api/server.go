package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"

	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jamestexas/context-keep/go-service/llm"
	"github.com/jamestexas/context-keep/go-service/redis"
	"github.com/jamestexas/context-keep/go-service/utils"
)

// APIHandler manages API requests
type APIHandler struct {
	storage *redis.RedisStore
	llm     llm.LLMClient        // Accept the interface.
	memory  *redis.MemoryManager // ✅ Add this
}

// StreamRequest represents the common request body for streaming endpoints.
type StreamRequest struct {
	ConvoID   string `json:"convo_id"`
	EventText string `json:"event_text"`
}

// NewAPIHandler initializes APIHandler
func NewAPIHandler(storage *redis.RedisStore, llmClient llm.LLMClient) *APIHandler {
	memManager := redis.NewMemoryManager(storage) // ✅ Initialize MemoryManager
	return &APIHandler{
		storage: storage,
		llm:     llmClient,
		memory:  memManager,
	}
}

// Router sets up all API routes
func (h *APIHandler) Router() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	// Custom Middleware
	r.Use(modelInjector)
	r.Use(convoIDInjector)
	r.Use(middleware.Compress(5))

	// Routes
	r.Get("/", h.HandleRoot)
	r.Get("/summary/{convo_id}", h.HandleGetSummary)
	r.Get("/conversation/{convo_id}", h.HandleGetHierarchicalMemory) // ✅ Add this if missing

	r.Post("/store", h.HandleStoreEvent)
	r.Get("/debug/conversation/{convo_id}", h.HandleDebugConversation)
	r.Post("/chat", h.HandleChatStream)           // ✅ Updated to streaming
	r.Post("/summarize", h.HandleSummarizeStream) // ✅ Updated to streaming
	r.Delete("/conversation/{convo_id}", h.HandleDeleteConversation)
	r.Delete("/delete/summary/{convo_id}", h.HandleDeleteSummary)

	return r
}

// ✅ HandleRoot (Welcome Route)
func (h *APIHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "Welcome to Context-Keep API!"})
}

// HandleGetSummary retrieves the summary for a given conversation.
func (h *APIHandler) HandleGetSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Retrieve conversation ID from context using the key defined in middleware.go.
	convoID, ok := ctx.Value(ContextKeyConvoID).(string)
	if !ok || convoID == "" {
		writeError(w, http.StatusBadRequest, "Missing conversation ID")
		return
	}
	summary, err := h.storage.GetSummary(ctx, convoID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Summary not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"summary": summary})
}

// ✅ HandleGetModel (Cleaner JSON Handling)
func (h *APIHandler) HandleGetModel(w http.ResponseWriter, r *http.Request) {
	model, ok := r.Context().Value("currentModel").(string)
	if !ok {
		model = llm.DefaultModel
	}
	writeJSON(w, http.StatusOK, map[string]string{"model": model})
}

// ✅ HandleStoreEvent (Cleaner JSON Handling)
func (h *APIHandler) HandleStoreEvent(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[StreamRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	ctx := r.Context()

	// Generate a unique event ID (use timestamp or UUID)
	eventID := generateEventID()

	// Use empty parentID (or derive from context)
	parentID := "root" // Adjust logic if parent tracking is needed

	// Store event properly
	if err := h.storage.StoreEvent(ctx, req.ConvoID, eventID, parentID, req.EventText, []string{}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to store event")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Stored successfully"})
}

// HandleDebugConversation retrieves recent events using the conversation ID from context.
func (h *APIHandler) HandleDebugConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Retrieve conversation ID from context.
	convoID, ok := ctx.Value(ContextKeyConvoID).(string)
	if !ok || convoID == "" {
		writeError(w, http.StatusBadRequest, "Missing conversation ID")
		return
	}

	data, err := h.storage.GetRecentEvents(ctx, convoID, 10)
	if err != nil {
		writeError(w, http.StatusNotFound, "No conversation found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"raw_data": data})
}

func (h *APIHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[StreamRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	ctx := r.Context()

	// Generate event ID
	eventID := generateEventID()

	// Assume conversation starts from root unless tracking parent
	parentID := "root"

	// Store event properly
	if err := h.storage.StoreEvent(ctx, req.ConvoID, eventID, parentID, req.EventText, []string{}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to store event")
		return
	}

	// Call LLM for response
	llmResponse, err := h.llm.GetCompletion(ctx, llm.CompletionRequest{
		Model:       llm.DefaultModel,
		Messages:    []openai.ChatCompletionMessage{{Role: "user", Content: req.EventText}}, // ✅ Use OpenAI's struct
		Temperature: 0.7,
		Stream:      false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LLM error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"response": llmResponse})
}

// ✅ HandleChatStream (Streaming LLM Chat)
func (h *APIHandler) HandleChatStream(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[StreamRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	ctx := r.Context()

	// Generate event ID
	eventID := generateEventID()
	parentID := "root"

	// Store event
	if err := h.storage.StoreEvent(ctx, req.ConvoID, eventID, parentID, req.EventText, []string{}); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to store event")
		return
	}

	// Stream response from LLM
	out, errs := h.llm.StreamCompletion(ctx, llm.CompletionRequest{
		Model:       llm.DefaultModel,
		Messages:    []openai.ChatCompletionMessage{{Role: "user", Content: req.EventText}},
		Temperature: 0.7,
		Stream:      true,
	})

	// Stream without extra processing.
	if err := utils.StreamSSEResponse(w, out, errs, func(chunk string) {}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Stream error: %v", err))
	}
}

// ✅ HandleSummarize
func (h *APIHandler) HandleSummarize(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[StreamRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	ctx := r.Context()
	convoSummary, err := h.storage.GetSummary(ctx, req.ConvoID)
	if err != nil {
		convoSummary = ""
	}

	// Generate summary with LLM
	llmResponse, err := h.llm.GetCompletion(ctx, llm.CompletionRequest{
		Model:       llm.DefaultModel,
		Messages:    []openai.ChatCompletionMessage{{Role: "system", Content: "Summarize the conversation:"}, {Role: "user", Content: convoSummary}},
		Temperature: 0.3,
		Stream:      false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LLM error")
		return
	}

	// Store summary
	if err := h.storage.StoreSummary(ctx, req.ConvoID, llmResponse); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to store summary")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"summary": llmResponse})
}

func (h *APIHandler) HandleSummarizeStream(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[StreamRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	ctx := r.Context()
	// Fetch structured conversation history
	memoryData, err := h.memory.GetHierarchicalMemory(ctx, req.ConvoID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to retrieve memory")
		return
	}

	// Extract stored summary (or provide fallback)
	convoSummary := memoryData["summary"].(string)
	if convoSummary == "" {
		convoSummary = "No prior summary available."
	}

	// Convert events to readable history format
	events := memoryData["events"].([]map[string]any)
	eventHistory := ""
	for _, event := range events {
		eventHistory += fmt.Sprintf("- %s: %s\n", event["event_id"], event["summary"])
	}

	// Stream summarization response
	out, errs := h.llm.StreamCompletion(ctx, llm.CompletionRequest{
		Model: llm.DefaultModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: "You are an assistant that provides only a concise, factual summary. Do not reflect on the request. Simply summarize the conversation in a few sentences."},
			{Role: "user", Content: fmt.Sprintf("Previous Summary: %s\n\nEvent History:\n%s", convoSummary, eventHistory)},
			{Role: "user", Content: req.EventText},
		},
		Temperature: 0.3,
		Stream:      true,
	})

	var finalSummary string
	if err := utils.StreamSSEResponse(w, out, errs, func(chunk string) {
		finalSummary += chunk
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Stream error: %v", err))
		return
	}

	// Store the final summary after utils.
	if err := h.storage.StoreSummary(ctx, req.ConvoID, finalSummary); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to store summary")
	}
}

// HandleDeleteConversation deletes a conversation using the conversation ID from context.
func (h *APIHandler) HandleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Retrieve conversation ID from context.
	convoID, ok := ctx.Value(ContextKeyConvoID).(string)
	if !ok || convoID == "" {
		writeError(w, http.StatusBadRequest, "Missing conversation ID")
		return
	}

	if err := h.storage.DeleteConversation(ctx, convoID); err != nil {
		writeError(w, http.StatusNotFound, "Conversation not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Conversation deleted"})
}

// HandleGetHierarchicalMemory retrieves structured conversation history
func (h *APIHandler) HandleGetHierarchicalMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Retrieve conversation ID from context injected by your middleware.
	convoID, ok := ctx.Value(ContextKeyConvoID).(string)
	if !ok || convoID == "" {
		writeError(w, http.StatusBadRequest, "Missing conversation ID")
		return
	}

	memoryData, err := h.memory.GetHierarchicalMemory(ctx, convoID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Respond with JSON-encoded structured memory data
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, memoryData)
}

// HandleDeleteSummary deletes a summary using the conversation ID from context.
func (h *APIHandler) HandleDeleteSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Retrieve conversation ID from context.
	convoID, ok := ctx.Value(ContextKeyConvoID).(string)
	if !ok || convoID == "" {
		writeError(w, http.StatusBadRequest, "Missing conversation ID")
		return
	}

	if err := h.storage.DeleteSummary(ctx, convoID); err != nil {
		writeError(w, http.StatusNotFound, "Summary not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Summary deleted"})
}

// ✅ StartServer (Cleaner)
func StartServer(port string, storage *redis.RedisStore, llmClient *llm.Client) {
	handler := NewAPIHandler(storage, llmClient)
	router := handler.Router()

	fmt.Printf("🚀 API Server running on %s\n", port)
	log.Fatal(http.ListenAndServe(port, router))
}

// decodeJSON decodes the JSON body from the request into a value of type T.
func decodeJSON[T any](r *http.Request) (T, error) {
	var v T
	err := json.NewDecoder(r.Body).Decode(&v)
	return v, err
}

func generateEventID() string {
	return fmt.Sprintf("event-%d", time.Now().UnixNano())
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes an error response in JSON format.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

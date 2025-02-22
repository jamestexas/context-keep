package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jamestexas/context-keep/go-service/api"
	"github.com/jamestexas/context-keep/go-service/llm"
	"github.com/jamestexas/context-keep/go-service/redis"
	"github.com/sashabaranov/go-openai"
)

var globalLLMClient *llm.Client // Global LLM client

const DefaultRedisAddr = "localhost:6379"              // Default Redis address (localhost)
const DEFAULT_LLM_API_URL = "http://127.0.0.1:1234/v1" // The base URL for the LLM API (localhost / http:// etc.)
const DEFAULT_LLM_API_KEY = "lm-studio"                // The default API key for the LLM API

// clearOnLocalhost clears Redis if running locally
func clearOnLocalhost(storage *redis.RedisStore, redisAddr string) {
	if redisAddr == DefaultRedisAddr {
		fmt.Println("🚨 Clearing Redis for fresh state...")
		if err := storage.ClearAll(context.Background()); err != nil {
			log.Fatalf("❌ Failed to clear Redis: %v", err)
		}
		fmt.Println("✅ Redis cleared!")
	}
}

// getStorage returns a Redis storage instance and the Redis address
func getStorage() (storage *redis.RedisStore, redisAddr string) {
	redisAddr = os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = DefaultRedisAddr
	}
	storage = redis.NewRedisStore(redisAddr)
	return
}

// getLmClient returns an LLM client
func getLmClient() *llm.Client {
	if globalLLMClient != nil {
		return globalLLMClient // ✅ Reuse existing client
	}

	api_key := os.Getenv("OPENAI_API_KEY")
	api_url := os.Getenv("BASE_LLM_API_URL")
	model := os.Getenv("LLM_MODEL") // ✅ Read from environment

	if api_key == "" {
		fmt.Println("⚠️  No OpenAI API Key provided. Using default LM Studio key.")
		api_key = DEFAULT_LLM_API_KEY
	}
	if api_url == "" {
		fmt.Println("⚠️  No API URL provided. Using default LM Studio URL.")
		api_url = DEFAULT_LLM_API_URL
	}
	if model == "" {
		model = "mistral-7b-v0.3" // ✅ Default model
		fmt.Printf("⚠️  No model specified. Using default: %s\n", model)
	} else {
		fmt.Printf("✅ Using LLM model: %s\n", model)
	}

	globalLLMClient = llm.NewClient(api_key, api_url) // ✅ Set global client
	return globalLLMClient
}

// streamTestQuery streams an LLM test query output
func streamTestQuery(llmClient *llm.Client) {
	ctx := context.Background()
	fmt.Println("🧪 Running LLM test query (streaming)...")

	// Define test request
	testRequest := llm.CompletionRequest{
		Model:       llm.DefaultModel,
		Messages:    []openai.ChatCompletionMessage{{Role: "user", Content: "Summarize recursion in one sentence."}},
		Temperature: 0.3,
		Stream:      true, // ✅ Enable streaming
	}

	// Stream response
	out, errs := llmClient.StreamCompletion(ctx, testRequest)

	// Read streamed response
	for {
		select {
		case chunk, ok := <-out:
			if !ok {
				fmt.Println("\n✅ Stream completed!")
				return
			}
			fmt.Print(chunk) // Print chunks as they arrive
		case err, ok := <-errs:
			if ok {
				fmt.Printf("\n❌ Streaming error: %v\n", err)
				return
			}
		}
	}
}
func main() {
	// Determine Redis address
	storage, redisAddr := getStorage()
	// Initialize Redis storage

	// Auto-clear Redis if running locally
	clearOnLocalhost(storage, redisAddr)

	// Initialize LLM client
	llmClient := getLmClient()

	// Run a test query
	streamTestQuery(llmClient)

	// Start API Server (Chi-based)
	api.StartServer(":8080", storage, llmClient)
}

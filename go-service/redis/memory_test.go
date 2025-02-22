package redis

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemoryManager(t *testing.T) {
	store := NewRedisStore("localhost:6379")
	memManager := NewMemoryManager(store)
	ctx := context.Background()

	// Store some test data
	_ = store.StoreSummary(ctx, "test_convo", "This is a hierarchical memory test")
	_ = store.StoreEvent(ctx, "test_convo", "event_1", "root", "Root event", []string{"Start of conversation"})
	_ = store.StoreEvent(ctx, "test_convo", "event_2", "event_1", "Follow-up event", []string{"A follow-up message"})

	// Test hierarchical memory retrieval
	result, err := memManager.GetHierarchicalMemory(ctx, "test_convo")
	assert.Nil(t, err)
	assert.Contains(t, result, "Summary: This is a hierarchical memory test")
	assert.Contains(t, result, "Event ID: event_1")
	assert.Contains(t, result, "Event ID: event_2")
}

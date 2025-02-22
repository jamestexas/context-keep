package redis

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisStorage(t *testing.T) {
	store := NewRedisStore("localhost:6379")
	ctx := context.Background()

	// Test storing a summary
	err := store.StoreSummary(ctx, "test_convo", "This is a test summary")
	assert.Nil(t, err)

	// Test retrieving the summary
	summary, err := store.GetSummary(ctx, "test_convo")
	assert.Nil(t, err)
	assert.Equal(t, "This is a test summary", summary)

	// Test storing an event
	err = store.StoreEvent(ctx, "test_convo", "event_1", "root", "Test event", []string{"Hello", "World"})
	assert.Nil(t, err)

	// Test retrieving the event
	event, err := store.GetEvent(ctx, "test_convo", "event_1")
	assert.Nil(t, err)
	assert.Equal(t, "Test event", event.Summary)
	assert.ElementsMatch(t, []string{"Hello", "World"}, event.Content)
}

package redis

import (
	"context"
)

// MemoryManager handles hierarchical memory retrieval.
type MemoryManager struct {
	storage Storage // changed from *RedisStore to the Storage interface.
}

// NewMemoryManager initializes MemoryManager.
func NewMemoryManager(storage Storage) *MemoryManager {
	return &MemoryManager{storage: storage}
}

// GetHierarchicalMemory retrieves hierarchical summaries & messages.
func (m *MemoryManager) GetHierarchicalMemory(ctx context.Context, convoID string) (map[string]interface{}, error) {
	rootSummary, err := m.storage.GetSummary(ctx, convoID)
	if err != nil {
		rootSummary = "No summary available."
	}

	// Fetch recent events (last 10).
	eventIDs, err := m.storage.GetRecentEvents(ctx, convoID, 10)
	if err != nil {
		return nil, err
	}

	// Retrieve event trees for the most recent events.
	var events []map[string]interface{}
	for _, eventID := range eventIDs {
		eventTree, err := m.fetchEventTree(ctx, convoID, eventID)
		if err == nil {
			events = append(events, eventTree)
		}
	}

	// Final structured response.
	return map[string]interface{}{
		"summary": rootSummary,
		"events":  events,
	}, nil
}

// fetchEventTree retrieves an event and its children as structured JSON.
func (m *MemoryManager) fetchEventTree(ctx context.Context, convoID, eventID string) (map[string]interface{}, error) {
	event, err := m.storage.GetEvent(ctx, convoID, eventID)
	if err != nil {
		return nil, err
	}

	// Create structured output.
	eventData := map[string]interface{}{
		"event_id": event.EventID,
		"summary":  event.Summary,
		"content":  event.Content,
		"children": []map[string]interface{}{}, // Will be filled recursively.
	}

	// Recursively fetch children.
	for _, childID := range event.ChildIDs {
		childData, err := m.fetchEventTree(ctx, convoID, childID)
		if err == nil {
			eventData["children"] = append(eventData["children"].([]map[string]interface{}), childData)
		}
	}

	return eventData, nil
}

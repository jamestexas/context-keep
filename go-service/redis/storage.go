package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Storage defines the methods our API depends on.
type Storage interface {
	StoreEvent(ctx context.Context, convoID, eventID, parentID, summary string, content []string) error
	GetSummary(ctx context.Context, convoID string) (string, error)
	GetRecentEvents(ctx context.Context, convoID string, count int) ([]string, error)
	StoreSummary(ctx context.Context, convoID, summary string) error
	GetEvent(ctx context.Context, convoID, eventID string) (EventNode, error)
	DeleteConversation(ctx context.Context, convoID string) error
	DeleteSummary(ctx context.Context, convoID string) error
}

// ConversationRoot represents the root summary
type ConversationRoot struct {
	ConvoID  string   `json:"convo_id"`
	Summary  string   `json:"summary"`
	ChildIDs []string `json:"child_ids"`
}

// EventNode represents a single conversation event
type EventNode struct {
	EventID  string   `json:"event_id"`
	ParentID string   `json:"parent_id"`
	Summary  string   `json:"summary"`
	Content  []string `json:"content"`
	ChildIDs []string `json:"child_ids"`
}

// RedisStore wraps the Redis connection
type RedisStore struct {
	Client *redis.Client
}

// NewRedisStore initializes Redis connection
func NewRedisStore(addr string) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := client.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("❌ Failed to connect to Redis: %v", err))
	}

	fmt.Println("✅ Connected to Redis")
	return &RedisStore{Client: client}
}

// StoreEvent stores an event in Redis and adds it to the event list
func (r *RedisStore) StoreEvent(ctx context.Context, convoID, eventID, parentID, summary string, content []string) error {
	eventKey := fmt.Sprintf("conversation:%s:event:%s", convoID, eventID)
	event := EventNode{
		EventID:  eventID,
		ParentID: parentID,
		Summary:  summary,
		Content:  content,
		ChildIDs: []string{},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Store the event
	if err := r.Client.Set(ctx, eventKey, data, 0).Err(); err != nil {
		return err
	}

	// Add event ID to the list of events for this conversation
	eventsKey := fmt.Sprintf("conversation:%s:events", convoID)
	if err := r.Client.RPush(ctx, eventsKey, eventID).Err(); err != nil {
		return err
	}

	// Update parent to include this child
	if parentID != "" {
		parentKey := fmt.Sprintf("conversation:%s:event:%s", convoID, parentID)
		parentData, err := r.Client.Get(ctx, parentKey).Result()
		if err == nil {
			var parent EventNode
			if err := json.Unmarshal([]byte(parentData), &parent); err == nil {
				parent.ChildIDs = append(parent.ChildIDs, eventID)
				updatedParentData, _ := json.Marshal(parent)
				_ = r.Client.Set(ctx, parentKey, updatedParentData, 0).Err()
			}
		}
	}

	return nil
}

// GetEvent retrieves an event
func (r *RedisStore) GetEvent(ctx context.Context, convoID, eventID string) (event EventNode, err error) {
	key := fmt.Sprintf("conversation:%s:event:%s", convoID, eventID)
	data, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		return event, err
	}

	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return event, err
	}

	return event, nil
}

// StoreSummary stores a conversation summary
func (r *RedisStore) StoreSummary(ctx context.Context, convoID, summary string) error {
	key := fmt.Sprintf("summary:%s", convoID)
	return r.Client.Set(ctx, key, summary, 0).Err()
}

// GetSummary retrieves a summary
func (r *RedisStore) GetSummary(ctx context.Context, convoID string) (summary string, err error) {
	key := fmt.Sprintf("summary:%s", convoID)
	summary, err = r.Client.Get(ctx, key).Result()
	return
}

// ClearAll removes all keys from Redis (only use in dev!)
func (r *RedisStore) ClearAll(ctx context.Context) error {
	return r.Client.FlushAll(ctx).Err()
}

func (r *RedisStore) GetRecentEvents(ctx context.Context, convoID string, count int) (eventIDs []string, err error) {
	eventsKey := fmt.Sprintf("conversation:%s:events", convoID)
	eventIDs, err = r.Client.LRange(ctx, eventsKey, int64(-count), -1).Result()
	return

}

func (r *RedisStore) DeleteConversation(ctx context.Context, convoID string) error {
	key := fmt.Sprintf("conversation:%s", convoID)
	return r.Client.Del(ctx, key).Err()
}

func (r *RedisStore) DeleteSummary(ctx context.Context, convoID string) error {
	key := fmt.Sprintf("summary:%s", convoID)
	return r.Client.Del(ctx, key).Err()
}

package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tiktoken-go/tokenizer"
)

// RedisStore manages Redis connection for token cache
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStore creates a new Redis store
func NewRedisStore(addr, password string) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: rdb,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// GetTokenCount retrieves cached token count for messages
func (r *RedisStore) GetTokenCount(messages []Message) (int, bool, error) {
	key := r.getCacheKey(messages)

	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	var count int
	if err := json.Unmarshal([]byte(val), &count); err != nil {
		return 0, false, err
	}

	return count, true, nil
}

// SetTokenCount caches token count for messages
func (r *RedisStore) SetTokenCount(messages []Message, count int, ttl time.Duration) error {
	key := r.getCacheKey(messages)

	data, err := json.Marshal(count)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, key, data, ttl).Err()
}

// CountTokens counts tokens in messages using tiktoken
func (r *RedisStore) CountTokens(messages []Message) (int, error) {
	// Use cl100k_base encoding (used by GPT models)
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return 0, fmt.Errorf("failed to get tokenizer: %w", err)
	}

	totalTokens := 0
	for _, msg := range messages {
		// Count tokens in content
		tokens, _, err := enc.Encode(msg.Content)
		if err != nil {
			return 0, fmt.Errorf("failed to encode content: %w", err)
		}
		totalTokens += len(tokens)

		// Add overhead for role and structure (approximate)
		totalTokens += 4
	}

	return totalTokens, nil
}

// getCacheKey generates a cache key from messages
func (r *RedisStore) getCacheKey(messages []Message) string {
	// Create a hash of the messages for the cache key
	data, _ := json.Marshal(messages)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("token_count:%s", hex.EncodeToString(hash[:]))
}

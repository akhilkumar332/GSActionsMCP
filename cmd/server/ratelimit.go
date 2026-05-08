package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// RateTokensPerSec is the rate at which tokens are added to the bucket (5 tokens/sec)
	RateTokensPerSec = 5.0
	// BurstCapacity is the maximum number of tokens in the bucket (10 burst)
	BurstCapacity = 10.0
)

var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = 1

local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
	tokens = capacity
	last_refill = now
end

local elapsed = now - last_refill
local refill = (elapsed / 1000.0) * rate
tokens = math.min(capacity, tokens + refill)

if tokens >= requested then
	tokens = tokens - requested
	redis.call("HSET", key, "tokens", tokens, "last_refill", now)
	redis.call("EXPIRE", key, math.ceil(capacity / rate) + 1)
	return 1
else
	redis.call("HSET", key, "tokens", tokens, "last_refill", last_refill)
	redis.call("EXPIRE", key, math.ceil(capacity / rate) + 1)
	return 0
end
`)

type rateLimiter struct {
	client *redis.Client
}

func (rl *rateLimiter) Allow(ctx context.Context, userID string) bool {
	if rl.client == nil {
		// Fallback if redis is not ready
		return false
	}

	now := time.Now().UnixMilli()
	key := "ratelimit:" + userID

	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	result, err := rateLimitScript.Run(ctx, rl.client, []string{key}, RateTokensPerSec, BurstCapacity, now).Result()
	if err != nil {
		log.Printf("Rate limit Lua script error for user %s: %v", userID, err)
		return false // Default deny on error
	}

	allowed, ok := result.(int64)
	if !ok {
		log.Printf("Rate limit result type assertion failed for user %s: expected int64, got %T", userID, result)
		return false
	}
	return allowed == 1
}

func (rl *rateLimiter) cleanup() {
	// No-op: Redis EXPIRE handles cleanup
}

package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRedisRateLimitAllowsUpToMaxCountThenRejects(t *testing.T) {
	_, redisClient := useRateLimitMiniRedis(t)

	ctx := context.Background()
	key := "rateLimit:model-fixed-window"

	for attempt := 1; attempt <= 2; attempt++ {
		allowed, err := checkRedisRateLimit(ctx, redisClient, key, 2, 60)
		require.NoError(t, err)
		assert.True(t, allowed, "attempt %d must stay within the limit", attempt)
	}

	allowed, err := checkRedisRateLimit(ctx, redisClient, key, 2, 60)
	require.NoError(t, err)
	assert.False(t, allowed, "the attempt past maxCount must be rejected")

	ttl, err := redisClient.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Positive(t, ttl, "the counter must carry the window TTL")
	assert.LessOrEqual(t, ttl, 60*time.Second)
}

func TestCheckRedisRateLimitSkipsCountingWhenMaxCountIsZero(t *testing.T) {
	_, redisClient := useRateLimitMiniRedis(t)

	ctx := context.Background()
	key := "rateLimit:model-unlimited"

	for attempt := 1; attempt <= 5; attempt++ {
		allowed, err := checkRedisRateLimit(ctx, redisClient, key, 0, 60)
		require.NoError(t, err)
		assert.True(t, allowed, "maxCount=0 must not limit attempt %d", attempt)
	}

	exists, err := redisClient.Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Zero(t, exists, "maxCount=0 must not create a counter key")
}

func TestCheckRedisRateLimitKeepsPerKeyWindowsSeparate(t *testing.T) {
	_, redisClient := useRateLimitMiniRedis(t)

	ctx := context.Background()

	allowed, err := checkRedisRateLimit(ctx, redisClient, "rateLimit:model-user-a", 1, 60)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = checkRedisRateLimit(ctx, redisClient, "rateLimit:model-user-b", 1, 60)
	require.NoError(t, err)
	assert.True(t, allowed, "a separate key must keep its own window")

	allowed, err = checkRedisRateLimit(ctx, redisClient, "rateLimit:model-user-a", 1, 60)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCheckRedisRateLimitWindowIsIndependentOfLocalTimezone(t *testing.T) {
	_, redisClient := useRateLimitMiniRedis(t)

	previousLocation := time.Local
	time.Local = time.FixedZone("test-utc-plus-eight", 8*60*60)
	t.Cleanup(func() { time.Local = previousLocation })

	ctx := context.Background()
	key := "rateLimit:model-timezone"

	allowed, err := checkRedisRateLimit(ctx, redisClient, key, 1, 60)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = checkRedisRateLimit(ctx, redisClient, key, 1, 60)
	require.NoError(t, err)
	assert.False(t, allowed, "a non-UTC host must not widen the fixed window")

	ttl, err := redisClient.TTL(ctx, key).Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, 60*time.Second)
}

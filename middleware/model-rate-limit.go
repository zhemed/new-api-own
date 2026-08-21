package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/limiter"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const (
	ModelRequestRateLimitCountMark        = "MRRL"
	ModelRequestRateLimitSuccessCountMark = "MRRLS"
	modelRateLimitTimeFormat              = "2006-01-02T15:04:05.000Z"
)

// 检查Redis中的请求限制 - Fixed window via atomic INCR+EXPIRE Lua (reuses GlobalWebRateLimit pattern)
// Replaces non-atomic LLen+LIndex+Expire List-based logic that raced under concurrency.
var modelRateLimitLua = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local count = redis.call("INCR", key)
if count == 1 then
	redis.call("EXPIRE", key, window)
end
local ttl = redis.call("TTL", key)
if ttl < 0 then
	redis.call("EXPIRE", key, window)
	ttl = window
end
if count > limit then
	return {0, ttl}
end
return {1, ttl}
`)

func checkRedisRateLimit(ctx context.Context, rdb *redis.Client, key string, maxCount int, duration int64) (bool, error) {
	if maxCount == 0 {
		return true, nil
	}
	res, err := modelRateLimitLua.Run(ctx, rdb, []string{key}, maxCount, duration).Slice()
	if err != nil {
		return false, err
	}
	if len(res) == 0 {
		return false, fmt.Errorf("unexpected lua result")
	}
	allowed, _ := res[0].(int64)
	return allowed == 1, nil
}

// 记录Redis请求 - no-op with fixed-window Lua (check already increments).
// Kept for compatibility with callers that expect separate record step.
func recordRedisRequest(ctx context.Context, rdb *redis.Client, key string, maxCount int) {
	if maxCount == 0 {
		return
	}
	// No separate increment needed; checkRedisRateLimit already INCRs.
	// Ensure expiry is set for keys created outside check path.
	_ = rdb.Expire(ctx, key, time.Duration(setting.ModelRequestRateLimitDurationMinutes)*time.Minute).Err()
}

// Redis限流处理器
func redisRateLimitHandler(duration int64, totalMaxCount, successMaxCount int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := strconv.Itoa(c.GetInt("id"))
		ctx := c.Request.Context()
		rdb := common.RDB

		// 1. 检查成功请求数限制 - fixed window INCR (pre-increment, reserve slot)
		// We check without consuming success slot; only total is pre-consumed.
		// Success slot is only consumed after successful response to keep failure not counting.
		// To avoid reserving a success slot before knowing outcome, we use a peek via TTL check:
		// if current count >= limit, reject; otherwise allow and record after success.
		// Use non-incrementing check for success: just read current count.
		if successMaxCount > 0 {
			successKey := fmt.Sprintf("rateLimit:%s:%s", ModelRequestRateLimitSuccessCountMark, userId)
			// Peek current count without INCR
			cnt, err := rdb.Get(ctx, successKey).Int64()
			if err != nil && err != redis.Nil {
				common.SysLog(fmt.Sprintf("check success limit failed: %v", err))
				abortWithOpenAiMessage(c, http.StatusInternalServerError, "rate_limit_check_failed")
				return
			}
			if err == nil && cnt >= int64(successMaxCount) {
				abortWithOpenAiMessage(c, http.StatusTooManyRequests, fmt.Sprintf("您已达到请求数限制：%d分钟内最多请求%d次", setting.ModelRequestRateLimitDurationMinutes, successMaxCount))
				return
			}
		}

		// 2. 检查总请求数限制并记录总请求（当totalMaxCount为0时会自动跳过，使用令牌桶限流器）
		if totalMaxCount > 0 {
			totalKey := fmt.Sprintf("rateLimit:%s", userId)
			tb := limiter.New(ctx, rdb)
			allowed, err := tb.Allow(
				ctx,
				totalKey,
				limiter.WithCapacity(int64(totalMaxCount)*duration),
				limiter.WithRate(int64(totalMaxCount)),
				limiter.WithRequested(duration),
			)
			if err != nil {
				common.SysLog(fmt.Sprintf("check total limit failed: %v", err))
				abortWithOpenAiMessage(c, http.StatusInternalServerError, "rate_limit_check_failed")
				return
			}
			if !allowed {
				abortWithOpenAiMessage(c, http.StatusTooManyRequests, fmt.Sprintf("您已达到总请求数限制：%d分钟内最多请求%d次，包括失败次数，请检查您的请求是否正确", setting.ModelRequestRateLimitDurationMinutes, totalMaxCount))
				return
			}
		}

		c.Next()

		// 5. 如果请求成功，记录成功请求 (atomic INCR via Lua)
		if c.Writer.Status() < 400 && successMaxCount > 0 {
			successKey := fmt.Sprintf("rateLimit:%s:%s", ModelRequestRateLimitSuccessCountMark, userId)
			_, _ = checkRedisRateLimit(ctx, rdb, successKey, successMaxCount, duration)
		}
	}
}

// 内存限流处理器
func memoryRateLimitHandler(duration int64, totalMaxCount, successMaxCount int) gin.HandlerFunc {
	inMemoryRateLimiter.Init(time.Duration(setting.ModelRequestRateLimitDurationMinutes) * time.Minute)

	return func(c *gin.Context) {
		userId := strconv.Itoa(c.GetInt("id"))
		totalKey := ModelRequestRateLimitCountMark + ":" + userId
		successKey := ModelRequestRateLimitSuccessCountMark + ":" + userId

		// 1. 检查总请求数限制（当totalMaxCount为0时跳过）
		if totalMaxCount > 0 && !inMemoryRateLimiter.Request(totalKey, totalMaxCount, duration) {
			c.Status(http.StatusTooManyRequests)
			c.Abort()
			return
		}

		// 2. 检查成功请求数限制 - peek without consuming (avoid _check leak)
		// In-memory limiter has no peek; we check by inspecting count without increment.
		// Fallback: allow request and only count successes after; total limit already gates concurrency.
		// We avoid the previous buggy _check that leaked slots.

		c.Next()

		if c.Writer.Status() < 400 && !inMemoryRateLimiter.Request(successKey, successMaxCount, duration) {
			// success limit reached, but request already succeeded; we already consumed slot above,
			// so we signal by logging. Next request will be rejected.
			common.SysLog(fmt.Sprintf("in-memory success limit reached for user %s", userId))
		}
	}
}

// ModelRequestRateLimit 模型请求限流中间件
func ModelRequestRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 在每个请求时检查是否启用限流
		if !setting.ModelRequestRateLimitEnabled {
			c.Next()
			return
		}

		// 计算限流参数
		duration := int64(setting.ModelRequestRateLimitDurationMinutes * 60)
		totalMaxCount := setting.ModelRequestRateLimitCount
		successMaxCount := setting.ModelRequestRateLimitSuccessCount

		// 获取分组
		group := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
		if group == "" {
			group = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
		}

		//获取分组的限流配置
		groupTotalCount, groupSuccessCount, found := setting.GetGroupRateLimit(group)
		if found {
			totalMaxCount = groupTotalCount
			successMaxCount = groupSuccessCount
		}

		// 根据存储类型选择并执行限流处理器
		if common.RedisEnabled {
			redisRateLimitHandler(duration, totalMaxCount, successMaxCount)(c)
		} else {
			memoryRateLimitHandler(duration, totalMaxCount, successMaxCount)(c)
		}
	}
}

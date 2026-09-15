package bocket

import (
	"net/http"

	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
)

// NewTokenBucket 用官方令牌桶替代手写版
func NewTokenBucket(capacity int, limit float64) *rate.Limiter {
	//返回一个容量为b，令牌生成速率为r的限流器
	return rate.NewLimiter(rate.Limit(limit), capacity)
}

// RateLimiter 限流中间件
func RateLimiter(l *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow方法返回事件是否拿到令牌，拿到就放行，未拿到直接拦截（互斥锁保证并发安全）
		if l.Allow() {
			c.Next()
			return
		}
		c.JSON(http.StatusTooManyRequests, gin.H{
			"msg": "活动太火爆，请稍后再试",
		})
		c.Abort() // 终止后续处理，绝不让请求流入下一层

	}
}

/*
	手写限流中间件
// TokenBucket 定义令牌桶结构
type TokenBucket struct {
	limit    float64    // 每秒生成的数量
	tokens   float64    // 当前令牌数量
	capacity int        // 令牌容量
	last     time.Time  // 上次结算的时间点
	mu       sync.Mutex // 互斥锁，保证高并发下令牌增减的并发安全
}

// NewTokenBucket 初始化令牌桶并启动后台定时投放协程
func NewTokenBucket(capacity int, limit float64) *TokenBucket {
	tb := &TokenBucket{
			capacity: capacity,
			tokens:   float64(capacity), // 初始状态满桶
			limit:    limit,
			last:     time.Now(),
	}
	return tb
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.last).Seconds()
	tb.tokens += elapsed * tb.limit

	if tb.tokens > float64(tb.capacity) {
			tb.tokens = float64(tb.capacity)
	}

	tb.last = time.Now()

	if tb.tokens >= 1 {
			tb.tokens--
			return true
	}
	return false
}
*/

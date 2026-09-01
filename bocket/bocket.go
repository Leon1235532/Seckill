package bocket

import (
	"net/http"

	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
)

// NewTokenBucket 用官方令牌桶替代手写版，函数名/签名保持不变，路由层无感
func NewTokenBucket(capacity int, limit float64) *rate.Limiter {
	return rate.NewLimiter(rate.Limit(limit), capacity)
}

// RateLimiter 限流中间件
func RateLimiter(l *rate.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
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

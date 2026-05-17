package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"golang.org/x/time/rate"
)

// GlobalRateLimitMiddleware 返回一个基于令牌桶的全局限流中间件
// limit: 每秒往桶里放多少个令牌 (比如 100 代表每秒允许 100 个请求)
// burst: 桶的最大容量，允许瞬间突发的最大并发量 (比如 200)
func GlobalRateLimitMiddleware(limit rate.Limit, burst int) app.HandlerFunc {
	// 初始化一个全局的令牌桶限流器
	limiter := rate.NewLimiter(limit, burst)

	return func(ctx context.Context, c *app.RequestContext) {
		// Allow() 会尝试从桶中取走 1 个令牌
		// 如果桶空了，说明请求过载，返回 false
		if !limiter.Allow() {
			c.JSON(429, map[string]interface{}{
				"code":    429,
				"message": "当前请求排队人数过多，请稍后再试 (Token Bucket 限流)",
			})
			c.Abort() // 终止后续的路由 Handler 执行
			return
		}

		// 取到令牌，放行！
		c.Next(ctx)
	}
}

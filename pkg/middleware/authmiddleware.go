package middleware

import (
	jwt2 "Goffer/pkg/jwt"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthMiddleware Hertz 版本的 JWT 鉴权与上下文透传中间件
func JWTAuthMiddleware(jwtManager *jwt2.JWTManager) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 1. 从 Hertz 的 RequestContext 获取 Header
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, utils.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, utils.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}
		tokenString := parts[1]

		// 2. 黑名单与降级逻辑（保留你优秀的原始逻辑）
		isBlacklisted, redisErr := jwtManager.IsTokenBlacklisted(ctx, tokenString)
		if redisErr != nil {
			slog.Warn("Redis unavailable, skipping blacklist check",
				"error", redisErr,
				"token", jwt2.GetTokenHash(tokenString))
		} else if isBlacklisted {
			c.AbortWithStatusJSON(http.StatusUnauthorized, utils.H{"error": "token has been revoked"})
			return
		}

		// 3. 校验 Token
		token, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, utils.H{"error": "token is expired"})
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, utils.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, err := jwt2.ExtractClaims(token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utils.H{"error": "extract claims failed"})
			c.Abort()
			return
		}

		// 4. 【关键修改】分布式的 Context 传递
		userID := claims["user_id"].(string)

		// 4.1 存入 Hertz 的请求上下文 (供网关层的其他 Handler 直接使用)
		c.Set("user_id", userID)
		c.Set("username", claims["username"])

		// 4.2 存入跨进程的 metainfo (随 RPC 请求透传给后端的 Kitex 服务)
		// 注意：metainfo 要求 key 和 value 都是 string 类型
		ctx = metainfo.WithValue(ctx, "user_id", userID)
		ctx = metainfo.WithValue(ctx, "username", claims["username"].(string))

		// 5. 传递携带了 metainfo 的标准 context 给下一个中间件或业务逻辑
		c.Next(ctx)
	}
}

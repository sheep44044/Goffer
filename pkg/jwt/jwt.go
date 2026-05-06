package jwt

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// JWTManager 封装了 JWT 相关的操作，避免使用全局变量
type JWTManager struct {
	secretKey []byte
	issuer    string
	redisCli  *redis.Client
}

// NewJWTManager 初始化一个 JWT 管理器
func NewJWTManager(SecretKey string, Issuer string, rdb *redis.Client) *JWTManager {
	return &JWTManager{
		secretKey: []byte(SecretKey),
		issuer:    Issuer,
		redisCli:  rdb,
	}
}

// GenerateToken 签发 Token (在 User 微服务登录逻辑中调用)
func (m *JWTManager) GenerateToken(userID string, username string, expire time.Duration) (string, error) {
	// 使用 UUID 作为唯一的 jti，避免并发冲突
	jti := uuid.New().String()

	claims := jwt.MapClaims{
		"user_id":  userID, // 建议统一用 string 处理 userID
		"username": username,
		"jti":      jti,
		"exp":      time.Now().Add(expire).Unix(),
		"iat":      time.Now().Unix(),
		"iss":      m.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// IsTokenBlacklisted 检查黑名单 (在 Hertz 网关中间件中调用，注意传入 ctx)
func (m *JWTManager) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false, nil
	}

	claims := jwt.MapClaims{}
	_, _, _ = jwt.NewParser().ParseUnverified(tokenString, claims)

	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return false, nil // 没有 jti 无法查黑名单
	}

	if m.redisCli == nil {
		return false, errors.New("cache client not initialized in jwt manager")
	}

	// 将 ctx 传入 Redis 操作，保障超时熔断和链路追踪生效
	key := "jwt:blacklist:" + jti
	_, err := m.redisCli.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return false, nil // 不在黑名单
	}
	if err != nil {
		return false, fmt.Errorf("cache error checking blacklist: %w", err)
	}

	return true, nil
}

// AddTokenToBlacklist 加入黑名单 (在用户登出逻辑中调用)
func (m *JWTManager) AddTokenToBlacklist(ctx context.Context, tokenString string, expiration time.Duration) error {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if jti, ok := claims["jti"].(string); ok {
		key := "jwt:blacklist:" + jti
		return m.redisCli.Set(ctx, key, "1", expiration).Err()
	}
	return nil
}

// ValidateToken 验证 Token 签名并解析 (在 Hertz 网关中调用)
func (m *JWTManager) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.secretKey, nil
	})
}

// ExtractClaims 提取负载
func ExtractClaims(token *jwt.Token) (jwt.MapClaims, error) {
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// GetTokenHash 仅做日志打印用的安全 Hash
func GetTokenHash(token string) string {
	if token == "" {
		return "empty"
	}
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash[:8])
}

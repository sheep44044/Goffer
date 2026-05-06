package cache

import (
	"Goffer/app/rpc/user/config"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Init 初始化 Redis 连接
func Init(cfg *config.Config) (*redis.Client, error) {
	// 拼接 Redis 的地址，例如 "localhost:6379"
	addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password, // 密码，没有则为空字符串
		DB:       cfg.Redis.DB,       // 使用的数据库，默认是 0

		// 以下是企业级项目中建议添加的连接池配置
		PoolSize:     100,             // 连接池最大连接数
		MinIdleConns: 10,              // 最小空闲连接数
		DialTimeout:  5 * time.Second, // 连接超时时间
		ReadTimeout:  3 * time.Second, // 读取超时时间
		WriteTimeout: 3 * time.Second, // 写入超时时间
	})

	// 测试连接是否成功 (go-cache v9 要求必须传入 context)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to cache: %w", err)
	}

	return rdb, nil
}

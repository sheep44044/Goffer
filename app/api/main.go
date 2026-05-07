package main

import (
	"Goffer/app/api/config"

	"Goffer/app/api/router"
	"Goffer/app/api/rpc"
	"Goffer/pkg/jwt"
	"fmt"
	"log"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 1. 加载网关层配置 (包含 Etcd 地址、RPC 客户端配置、JWT 密钥等)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load gateway config: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	// 2. 初始化下游 RPC 客户端 (User 服务, Interview 服务等)
	rpc.InitRpcClients(cfg)

	// 3. 初始化 JWT Manager
	jwtManager := jwt.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.Issuer, rdb)

	// 4. 初始化 Hertz 引擎
	// 监听 8080 端口，前端请求将发往这里
	h := server.Default(server.WithHostPorts("0.0.0.0:8080"))

	// 5. 注册路由并挂载中间件
	router.InitRouter(h, jwtManager)

	log.Println("Gateway server is running on http://0.0.0.0:8080")

	// 6. 启动 Hertz 网关服务 (阻塞等待)
	h.Spin()
}

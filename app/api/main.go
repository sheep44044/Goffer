package main

import (
	"Goffer/app/api/config"
	"Goffer/app/api/router"
	"Goffer/app/api/rpc"
	"Goffer/pkg/jwt"
	"Goffer/pkg/telemetry"
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/cors"
	"github.com/hertz-contrib/obs-opentelemetry/tracing"
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
	// 1. 初始化全局 OTel，指向本地 4317 端口
	shutdown, _ := telemetry.InitOTel("goffer-api-gateway", "localhost:4317")
	defer shutdown(context.Background())

	// 2. 注入 Hertz OTel Tracer
	tracer, config := tracing.NewServerTracer()
	h := server.Default(
		server.WithHostPorts("0.0.0.0:8080"),
		tracer,
	)
	h.Use(tracing.ServerMiddleware(config))
	h.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 生产环境请换成具体的前端域名
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	// 5. 注册路由并挂载中间件
	router.InitRouter(h, jwtManager)

	log.Println("Gateway server is running on http://0.0.0.0:8080")

	// 6. 启动 Hertz 网关服务 (阻塞等待)
	h.Spin()
}

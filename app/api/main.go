package main

import (
	"Goffer/app/api/config"
	"Goffer/app/api/router"
	"Goffer/app/api/rpc"
	"Goffer/pkg/jwt"
	"Goffer/pkg/logger"
	"Goffer/pkg/telemetry"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/cors"
	"github.com/hertz-contrib/obs-opentelemetry/tracing"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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
	shutdown, err := telemetry.InitOTel("goffer-api-gateway", "localhost:4317")
	if err != nil {
		log.Fatalf("Failed to initialize OTel: %v", err)
	}
	defer func() {
		// 如果 OTel Collector 网关此时断开了，直接调用 shutdown 可能会导致整个 main 进程永久卡死在退出阶段。
		// 因此必须构建一个带 Timeout 的 Context，规定其必须在 5 秒内平滑完成内存刷新。
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if shutdownErr := shutdown(ctx); shutdownErr != nil {
			logger.WarnCtx(ctx, "Warning: OpenTelemetry shutdown failed", zap.Error(shutdownErr))
		} else {
			logger.WarnCtx(ctx, "OpenTelemetry telemetry tracing flush and shutdown cleanly.")
		}
	}()

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

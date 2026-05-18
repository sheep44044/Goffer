.PHONY: all user knowledge agent interview earmouth media api clean

# 启动所有基础依赖 (前提是你写好了 docker-compose)
up:
	cd docker-compose && docker compose up -d

# 分别启动各个服务 (加上 & 让它们在后台运行)
run-all:
	@echo "Starting all Goffer services..."
	@go run app/rpc/user/main.go > user.log 2>&1 &
	@go run app/rpc/knowledge/main.go > knowledge.log 2>&1 &
	@go run app/rpc/agent/main.go > agent.log 2>&1 &
	@go run app/rpc/interview/main.go > interview.log 2>&1 &
	@go run app/rpc/earmouth/main.go > earmouth.log 2>&1 &
	@go run app/rpc/media/main.go > media.log 2>&1 &
	@sleep 2 # 等待 RPC 服务启动
	@go run app/api/main.go > api.log 2>&1 &
	@echo "All services started successfully! 🚀"

# 一键停止所有本地服务
stop:
	@echo "Stopping all Goffer services..."
	@pkill -f "go run app/rpc" || true
	@pkill -f "go run app/api" || true
	@echo "All services stopped."
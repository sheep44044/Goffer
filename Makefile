.PHONY: all user knowledge agent interview earmouth media api clean frontend

# 启动所有基础依赖
up:
	cd docker-compose && docker compose up -d

# 启动所有后端服务
run-all:
	@echo "Starting all Goffer services..."
	@go run app/rpc/user/main.go > user.log 2>&1 &
	@go run app/rpc/knowledge/main.go > knowledge.log 2>&1 &
	@go run app/rpc/agent/main.go > agent.log 2>&1 &
	@go run app/rpc/interview/main.go > interview.log 2>&1 &
	@go run app/rpc/earmouth/main.go > earmouth.log 2>&1 &
	@go run app/rpc/media/main.go > media.log 2>&1 &
	@sleep 2
	@go run app/api/main.go > api.log 2>&1 &
	@echo "All services started! 🚀"

# 启动前端开发服务器
frontend:
	@echo "Frontend: http://localhost:3000/candidate.html"
	@echo "Admin:   http://localhost:3000/admin.html"
	@python3 -m http.server 3000 -d frontend/

# 一键停止所有本地服务
stop:
	@echo "Stopping all Goffer services..."
	@pkill -f "go run app/rpc" || true
	@pkill -f "go run app/api" || true
	@pkill -f "python3 -m http.server 3000" || true
	@echo "All services stopped."

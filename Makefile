.PHONY: help build build-all up down logs clean test docker-test

# 默认目标
help:
	@echo "HLE Agent Docker 管理命令"
	@echo ""
	@echo "可用命令:"
	@echo "  make build          - 构建主应用镜像"
	@echo "  make build-all      - 构建所有镜像"
	@echo "  make up             - 启动所有服务"
	@echo "  make down           - 停止所有服务"
	@echo "  make logs           - 查看日志"
	@echo "  make clean          - 清理未使用的镜像和容器"
	@echo "  make test           - 运行单元测试"
	@echo "  make docker-test    - 在容器中运行测试"
	@echo "  make python-image   - 构建 Python 执行器镜像"
	@echo "  make sagemath-image - 构建 SageMath 执行器镜像"

# 构建主应用镜像
build:
	docker build -f docker/Dockerfile -t hle-agent:latest .

# 构建所有镜像
build-all:
	cd docker && docker-compose build

# 构建 Python 执行器镜像
python-image:
	docker build -f docker/python_executor/Dockerfile -t hle-agent-python:latest .

# 构建 SageMath 执行器镜像
sagemath-image:
	docker build -f docker/sagemath_executor/Dockerfile -t hle-agent-sagemath:latest .

# 启动服务
up:
	cd docker && docker-compose up -d

# 停止服务
down:
	cd docker && docker-compose down

# 查看日志
logs:
	cd docker && docker-compose logs -f

# 清理未使用的资源
clean:
	docker system prune -f
	docker image prune -f

# 运行单元测试
test:
	go test ./... -v

# 在容器中运行测试
docker-test:
	cd docker && docker-compose run --rm hle-agent go test ./... -v

# 进入容器
shell:
	cd docker && docker-compose exec hle-agent sh

# 查看服务状态
status:
	cd docker && docker-compose ps

# 查看资源使用
stats:
	docker stats

# 重启服务
restart:
	cd docker && docker-compose restart

# 更新并重启
update:
	cd docker && docker-compose pull && docker-compose up -d --build


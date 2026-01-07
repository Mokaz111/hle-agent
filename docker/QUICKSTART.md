# Docker 快速开始指南

## 快速启动

### 1. 构建镜像

```bash
# 使用 Makefile（推荐）
make build-all

# 或使用 docker-compose（从项目根目录）
cd docker
docker-compose build
```

### 2. 配置

编辑项目根目录的 `config.yaml`，设置 API Key 和执行模式：

```yaml
model:
  api_key: "your-api-key"
  
tools:
  python:
    enabled: true
    execution_mode: "docker"  # 使用 Docker 模式
    docker_image: "hle-agent-python:latest"
  sagemath:
    enabled: true
    execution_mode: "docker"  # 使用 Docker 模式
    docker_image: "hle-agent-sagemath:latest"
```

### 3. 启动服务

```bash
# 从 docker 目录启动（推荐）
cd docker
docker-compose up -d

# 或从项目根目录启动
docker-compose -f docker/docker-compose.yml up -d

# 查看日志
docker-compose logs -f hle-agent
```

### 4. 运行测试

```bash
# 进入容器
docker-compose exec hle-agent sh

# 运行应用
./hle-agent
```

## 常用命令

```bash
# 查看服务状态
docker-compose ps

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看资源使用
docker stats
```

## 执行模式说明

### Local 模式（本地执行）

在主容器内直接执行代码，需要主容器安装 Python/SageMath：

```yaml
tools:
  python:
    execution_mode: "local"
```

### Docker 模式（推荐）

在独立的 Docker 容器中执行，更安全：

```yaml
tools:
  python:
    execution_mode: "docker"
    docker_image: "hle-agent-python:latest"
```

## 故障排查

如果遇到问题，查看详细文档：`docs/docker.md`


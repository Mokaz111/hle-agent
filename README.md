# HLE Agent - 考试答题智能体系统

<div align="center">

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

**基于 CloudWeGo Eino ADK 的 HLE 考试答题智能体系统**

[快速开始](#快速开始) • [文档](#文档) • [特性](#核心特性) • [架构](#系统架构)

</div>

---

## 📖 项目简介

HLE Agent 是一个专为 **HLE（Human Last Exam）基准测试**设计的智能答题系统。HLE 是一项涵盖数学、人文科学和自然科学等数十个学科的综合性基准测试，包含 2500 道题目，由全球超过 50 个国家 500 所科研院校的约 1000 名各领域专家共同开发。

本系统基于 **CloudWeGo Eino ADK** 的 **Plan-Execute Agent** 模式构建，采用"规划-执行-反思"的智能体架构，能够理解复杂题目、进行多步推理、执行计算任务，并生成准确的答案。

## ✨ 核心特性

### 🧠 智能规划与执行
- **Plan-Execute 架构**：基于 Eino ADK 的规划-执行-重规划循环
- **多步推理**：支持复杂的多步骤问题求解
- **自动重规划**：当执行失败时自动调整策略

### 🔧 强大的工具支持
- **Python 执行器**：支持数据科学、密码学、机器学习等任务
- **SageMath 执行器**：支持符号计算、数论、代数等高级数学运算
- **Docker 隔离**：工具在隔离的容器中执行，确保安全性

### 📚 知识库系统
- **思维链检索**：从历史成功案例中检索相似解题思路
- **RAG 增强**：使用检索增强生成提升小模型性能
- **相似度匹配**：基于关键词、领域、复杂度等多维度匹配

### 🎯 感知与反馈
- **智能感知层**：自动识别题目类型、领域、复杂度
- **答案验证**：自动验证答案的正确性和完整性
- **置信度评估**：为每个答案提供置信度评分

### 🐳 容器化部署
- **多容器架构**：主应用、Python 执行器、SageMath 执行器独立部署
- **安全隔离**：工具执行在隔离的容器环境中
- **资源限制**：支持 CPU、内存等资源限制

### 📊 完善的日志与监控
- **结构化日志**：使用 zap 进行高性能结构化日志
- **性能指标**：记录执行时间、输出长度等关键指标
- **决策点追踪**：记录关键决策点，便于问题排查

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                     人机交互层                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│  │ 交互模式  │  │ 批处理   │  │ 标准输入  │                │
│  └──────────┘  └──────────┘  └──────────┘                │
├─────────────────────────────────────────────────────────────┤
│              Eino ADK Plan-Execute Agent                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│  │ Planner  │→ │ Executor  │→ │Replanner │                │
│  └──────────┘  └──────────┘  └──────────┘                │
├─────────────────────────────────────────────────────────────┤
│  感知层  │  记忆层  │  检索层  │  知识库  │  反馈层        │
├─────────────────────────────────────────────────────────────┤
│  工具层: Python Executor │ SageMath Executor                │
└─────────────────────────────────────────────────────────────┘
```

## 🛠️ 技术栈

- **开发语言**: Go 1.25+
- **Agent 框架**: CloudWeGo Eino ADK (Plan-Execute Agent)
- **LLM 支持**: OpenAI, MiniMax, DeepSeek, Claude 等
- **工具执行**: Python 3.11+, SageMath
- **存储**: SQLite (知识库), 内存 (短期记忆)
- **日志**: zap + lumberjack (日志轮转)
- **容器化**: Docker + Docker Compose
- **测试**: Go testing + testify

## 📁 项目结构

```
hle-agent/
├── cmd/
│   ├── agent/                    # 主程序入口
│   │   └── main.go              # 命令行工具
│   └── import-chains/           # 知识库导入工具
│       └── main.go
│
├── internal/
│   ├── agent/                    # Agent 核心实现
│   │   ├── agent.go             # HLEAgent 主类
│   │   ├── planner.go           # 规划器
│   │   ├── executor.go          # 执行器
│   │   ├── replanner.go         # 重规划器
│   │   └── *_test.go           # 单元测试
│   ├── config/                   # 配置管理
│   │   └── config.go
│   └── models/                   # 数据模型
│       └── question.go
│
├── pkg/
│   ├── llm/                      # LLM 客户端
│   ├── memory/                   # 记忆模块
│   ├── perception/               # 感知层
│   ├── retriever/                # 检索器
│   ├── knowledgebase/            # 知识库
│   ├── feedback/                 # 反馈层
│   ├── prompts/                  # Prompt 模板
│   ├── logging/                  # 日志系统
│   └── utils/                    # 工具函数
│
├── tools/
│   ├── python_executor/          # Python 执行器
│   │   ├── executor.go
│   │   ├── docker_executor.go
│   │   └── Dockerfile
│   └── sagemath_executor/        # SageMath 执行器
│       ├── executor.go
│       ├── docker_executor.go
│       └── Dockerfile
│
├── tests/
│   └── e2e/                      # 端到端测试
│
├── docs/                         # 文档
│   ├── architecture.md           # 架构设计
│   ├── detailed_design.md        # 详细设计
│   ├── docker.md                # Docker 部署指南
│   └── ...
│
├── docker/                       # Docker 相关文件
│   ├── Dockerfile               # 主应用镜像
│   ├── docker-compose.yml       # 服务编排
│   ├── .dockerignore           # Docker 忽略文件
│   ├── QUICKSTART.md           # Docker 快速开始
│   ├── python_executor/        # Python 执行器镜像
│   │   └── Dockerfile
│   └── sagemath_executor/      # SageMath 执行器镜像
│       └── Dockerfile
├── Makefile                      # 构建命令
├── config.yaml                   # 配置文件
└── README.md
```

## 🚀 快速开始

### 前置要求

- Go 1.25+ (本地运行)
- Docker 20.10+ & Docker Compose 2.0+ (容器部署)
- LLM API Key (OpenAI, MiniMax, DeepSeek 等)

### 方式一：Docker 部署（推荐）

#### 1. 克隆项目

```bash
git clone <repository-url>
cd hle-agent
```

#### 2. 配置

编辑 `config.yaml`，设置 API Key：

```yaml
model:
  provider: "openai"  # 或 "minimax", "deepseek" 等
  api_key: "your-api-key"
  model: "gpt-4"      # 或 "MiniMax-M2.1" 等
  base_url: "https://api.openai.com/v1"
```

配置工具执行模式（推荐使用 Docker 模式）：

```yaml
tools:
  python:
    enabled: true
    execution_mode: "docker"  # "local" 或 "docker"
    docker_image: "hle-agent-python:latest"
  sagemath:
    enabled: true
    execution_mode: "docker"
    docker_image: "hle-agent-sagemath:latest"
```

#### 3. 构建和启动

```bash
# 构建所有镜像
make build-all
# 或
cd docker && docker-compose build

# 启动服务
make up
# 或
cd docker && docker-compose up -d

# 查看日志
make logs
# 或
cd docker && docker-compose logs -f hle-agent
```

#### 4. 运行

```bash
# 进入容器
make shell
# 或
cd docker && docker-compose exec hle-agent sh

# 运行应用（交互模式）
./hle-agent

# 或批处理模式
./hle-agent --input questions.jsonl --output results/
```

### 方式二：本地运行

#### 1. 安装依赖

```bash
go mod tidy
```

#### 2. 安装工具依赖

```bash
# 安装 Python 3.11+
# 安装 SageMath（可选）
```

#### 3. 配置

编辑 `config.yaml`，设置 API Key 和执行模式：

```yaml
tools:
  python:
    execution_mode: "local"  # 本地模式
  sagemath:
    execution_mode: "local"
```

#### 4. 运行

```bash
# 交互模式
go run cmd/agent/main.go

# 批处理模式
go run cmd/agent/main.go --input questions.jsonl --output results/

# 标准输入模式
echo "What is 2+2?" | go run cmd/agent/main.go
```

## ⚙️ 配置说明

### 主要配置项

#### LLM 配置

```yaml
model:
  provider: "openai"           # 提供商: openai, minimax, deepseek, anthropic
  api_key: "your-api-key"      # API 密钥
  model: "gpt-4"               # 模型名称
  base_url: "https://api.openai.com/v1"
  temperature: 0.0             # 温度参数
  max_tokens: 4096             # 最大 token 数
  timeout: 60                  # 超时时间（秒）
```

#### 工具配置

```yaml
tools:
  python:
    enabled: true
    execution_mode: "docker"  # "local", "docker", "http"
    docker_image: "hle-agent-python:latest"
    timeout: 300               # 超时时间（秒）
    sandbox_enabled: true      # 启用沙箱模式
  sagemath:
    enabled: true
    execution_mode: "docker"
    docker_image: "hle-agent-sagemath:latest"
    timeout: 300
```

#### 知识库配置

```yaml
knowledge_base:
  enabled: true
  storage_path: "./data/knowledge_base.db"
  max_results: 3               # 最大检索结果数
  min_similarity: 0.5          # 最小相似度阈值
  weight_keywords: 0.4         # 关键词权重
  weight_domain: 0.4           # 领域权重
  weight_complexity: 0.2       # 复杂度权重
```

#### 日志配置

```yaml
logging:
  level: "info"                 # debug, info, warn, error
  output_path: "./logs/hle-agent.log"
  development: false
  rotation:
    enabled: true
    max_size: 100              # MB
    max_backups: 10
    max_age: 30                # 天
    compress: true
```

详细配置说明请参考 [配置文档](docs/detailed_design.md#配置管理)。

## 📝 使用示例

### 交互式模式

```bash
$ ./hle-agent
> What is the square root of 144?
[Agent] 分析题目...
[Agent] 生成解题计划...
[Agent] 执行计算...
[Agent] 答案: 12
置信度: 0.95
```

### 批处理模式

```bash
# 准备问题文件 questions.jsonl
echo '{"id": "q1", "question": "What is 2+2?"}' > questions.jsonl

# 运行批处理
./hle-agent --input questions.jsonl --output results/

# 查看结果
cat results/q1.json
```

### 导入知识库

```bash
# 导入思维链数据
go run cmd/import-chains/main.go --input examples/chains_example.json
```

## 🧪 测试

### 运行单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/agent/...

# 查看覆盖率
go test ./... -cover
```

### 运行端到端测试

```bash
# 运行 E2E 测试（需要配置 API Key）
go test ./tests/e2e/... -v

# 跳过 E2E 测试
go test ./tests/e2e/... -short
```

## 📚 文档

- [架构设计文档](docs/architecture.md) - 系统架构和设计理念
- [详细设计文档](docs/detailed_design.md) - 组件详细设计
- [需求文档](docs/requirements.md) - 功能需求和非功能需求
- [Docker 部署指南](docs/docker.md) - 容器化部署详细说明
- [Docker 快速开始](docker/QUICKSTART.md) - Docker 快速开始指南
- [知识库设计](docs/knowledge_base_design.md) - 知识库模块设计
- [TODO 列表](docs/todo.md) - 开发计划和任务

## 🔧 开发指南

### 环境设置

```bash
# 安装 Go 1.25+
# 安装依赖
go mod download

# 安装开发工具
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `golangci-lint` 进行代码检查
- 提交前运行测试确保通过

### 构建

```bash
# 构建二进制文件
go build -o hle-agent ./cmd/agent/main.go

# 构建 Docker 镜像
make build
# 或
docker build -t hle-agent:latest .
```

### 贡献

欢迎提交 Issue 和 Pull Request！

## 📊 项目状态

- ✅ 核心功能实现完成
- ✅ 知识库模块完成
- ✅ Docker 容器化完成
- ✅ 单元测试覆盖（核心组件 74.7%）
- ✅ 日志系统完善
- 🔄 端到端测试进行中
- 📋 性能优化待进行

详细开发计划请参考 [TODO 列表](docs/todo.md)。

## 🤝 致谢

- [CloudWeGo Eino ADK](https://www.cloudwego.io/zh/docs/eino/) - Agent 框架
- [HLE Benchmark](https://hle-benchmark.org/) - 基准测试数据集

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🔗 相关链接

- [Eino ADK 文档](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/)
- [Plan-Execute Agent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/plan_execute/)
- [HLE 基准测试](https://hle-benchmark.org/)

---

<div align="center">

**Made with ❤️ for HLE Benchmark**

[报告问题](https://github.com/your-repo/issues) • [功能请求](https://github.com/your-repo/issues) • [讨论](https://github.com/your-repo/discussions)

</div>


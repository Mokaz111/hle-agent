# HLE Agent - 考试答题智能体系统

## 项目介绍

HLE Agent 是一个基于 CloudWeGo Eino ADK 的考试答题智能体系统，专为 HLE（Human Last Exam）基准测试设计。

## 技术栈

- **开发语言**: Go
- **Agent框架**: CloudWeGo Eino ADK (Plan-Execute Agent)
- **工具语言**: Python (MCP Tools)
- **LLM**: OpenAI/Claude
- **容器化**: Docker (SageMath环境)

## 项目结构

```
hle-agent/
├── cmd/
│   └── agent/
│       └── main.go              # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go            # 配置管理
│   ├── models/
│   │   └── question.go          # 数据结构定义
│   ├── agent/
│   │   └── agent.go             # Agent核心实现
│   ├── tools/
│   │   └── ...                  # 工具实现
│   └── utils/
│       └── ...                  # 工具函数
├── pkg/
│   ├── llm/
│   │   └── client.go            # LLM客户端
│   └── prompts/
│       └── templates.go         # Prompt模板
├── tools/
│   ├── python_executor/         # Python MCP Tool
│   └── sagemath_executor/       # SageMath MCP Tool
├── scripts/                     # 脚本
├── testdata/                    # 测试数据
├── config.yaml                  # 配置文件
├── go.mod
└── README.md
```

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置

编辑 `config.yaml` 文件，设置 API Key：

```yaml
model:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"  # 或直接填入API Key
  model: "gpt-4"
```

### 3. 运行

```bash
# 交互模式
go run cmd/agent/main.go

# 批处理模式
go run cmd/agent/main.go --input questions.jsonl --output results/
```

## 开发计划

- [x] M1: 设计完成
- [ ] M2: Eino ADK 集成
- [ ] M3: 核心功能完成
- [ ] M4: 工具开发完成
- [ ] M5: 测试完成
- [ ] M6: 交付

## 参考文档

- [Eino ADK 快速开始](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_quickstart/)
- [Plan-Execute Agent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/plan_execute/)

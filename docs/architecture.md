# AI Agent 总体架构设计文档 - HLE 考试答题系统

## 文档信息

| 项目 | 内容 |
|------|------|
| **项目名称** | HLE 考试答题智能体系统 |
| **关联文档** | 需求文档 V2.0 |
| **版本** | V2.0 |
| **文档状态** | 已实现 |
| **创建日期** | 2026-01-05 |
| **更新日期** | 2026-01-XX |
| **技术框架** | CloudWeGo Eino ADK |

---

## 一、架构设计概述

### 1.1 设计理念

本架构设计基于 **CloudWeGo Eino ADK** 的 **Plan-Execute Agent** 模式进行设计，充分利用框架提供的"规划-执行-反思"范式，构建专注于 HLE 考试答题的专用智能体系统。

架构设计的核心目标是实现智能性与可控性的平衡。在智能性方面，系统需要具备理解复杂题目、进行多步推理、执行计算任务、生成准确答案的能力；在可控性方面，系统需要确保每一步决策都有明确的依据和日志记录，核心高风险操作需要经过验证，同时系统需要具备异常处理和自我纠错能力。

### 1.2 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          人机交互层 (Human-Interaction Layer)                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ 交互式模式  │  │ 批处理模式  │  │ 标准输入    │  │ 结果导出    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘          │
│                              (cmd/agent/main.go)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                            反馈层 (Feedback Layer)                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ 答案校验    │  │ 置信度评估  │  │ 质量评估    │  │ 经验沉淀    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘          │
│                        (pkg/feedback/feedback.go)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                    Eino ADK PlanExecuteAgent 核心                            │
│                    (internal/agent/agent.go)                                │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │                         Planner (规划器)                              │     │
│  │  - 理解题目意图                                                      │     │
│  │  - 检索知识库思维链                                                  │     │
│  │  - 检索历史上下文                                                    │     │
│  │  - 生成解题计划                                                      │     │
│  │  - 定义执行步骤                                                      │     │
│  │  (internal/agent/planner.go)                                        │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│                                       │                                      │
│                                       ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │                        Executor (执行器)                              │     │
│  │  - 执行解题步骤                                                      │     │
│  │  - 调用外部工具（支持重试和降级）                                    │     │
│  │  - 调用 LLM 推理                                                    │     │
│  │  - 返回执行结果                                                      │     │
│  │  (internal/agent/executor.go)                                      │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│                                       │                                      │
│                                       ▼                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐     │
│  │                        Replanner (重规划器)                           │     │
│  │  - 评估执行结果                                                      │     │
│  │  - 决定是否需要重规划                                                │     │
│  │  - 生成新计划或结束任务                                              │     │
│  │  (internal/agent/replanner.go)                                     │     │
│  └─────────────────────────────────────────────────────────────────────┘     │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                          执行层 (Execution Layer)                            │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐   │
│  │     Python执行器    │  │    SageMath执行器   │  │     工具注册表      │   │
│  │    (MCP Tool)       │  │    (MCP Tool)       │  │   (ToolRegistry)   │   │
│  │  tools/python_...   │  │  tools/sagemath_... │  │  internal/agent/   │   │
│  └─────────────────────┘  └─────────────────────┘  └─────────────────────┘   │
├─────────────────────────────────────────────────────────────────────────────┤
│                          记忆层 (Memory Layer)                               │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐   │
│  │     短期记忆        │  │     检索器          │  │     知识库          │   │
│  │  (ShortTermMemory)  │  │  (Retriever)        │  │  (KnowledgeBase)    │   │
│  │  pkg/memory/        │  │  pkg/retriever/     │  │  pkg/knowledgebase/ │   │
│  └─────────────────────┘  └─────────────────────┘  └─────────────────────┘   │
│  会话状态、历史记录       历史问题检索          GPT思维链存储与检索         │
├─────────────────────────────────────────────────────────────────────────────┤
│                          感知层 (Perception Layer)                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ 数据解析器  │  │ 领域识别器  │  │ 特征提取器  │  │ 格式规范化  │          │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘          │
│                        (pkg/perception/perception.go)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                          基础设施层 (Infrastructure Layer)                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ 日志系统    │  │ 配置管理    │  │ 错误处理    │  │ LLM客户端   │          │
│  │ (Logging)   │  │ (Config)     │  │ (Errors)    │  │ (LLM Client)│          │
│  │ pkg/logging │  │ internal/    │  │ internal/   │  │ pkg/llm/    │          │
│  │             │  │ config/      │  │ agent/      │  │             │          │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘          │
│  支持轮转、性能指标        YAML配置         自定义错误类型   多模型支持      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.3 Eino ADK Plan-Execute 模式

**Plan-Execute Agent** 是 Eino ADK 中一种基于"规划-执行-反思"范式的多智能体协作框架，包含三个核心智能体：

| 组件 | 职责 | 在本系统中的作用 | 实现位置 |
|------|------|------------------|----------|
| **Planner** | 生成解题计划 | 理解题目，检索知识库和历史，生成结构化解题步骤 | `internal/agent/planner.go` |
| **Executor** | 执行计划步骤 | 调用工具（支持重试和降级），执行推理，返回结果 | `internal/agent/executor.go` |
| **Replanner** | 评估并调整计划 | 评估结果，决定是否需要重规划 | `internal/agent/replanner.go` |

**工作流程：**
```
1. 用户输入题目
   ↓
2. Planner: 生成解题计划
   ├─ 感知层分析（领域识别）
   ├─ 检索知识库思维链（如启用）
   ├─ 检索历史上下文（Retriever）
   ├─ 调用LLM生成计划（HLEPlan）
   └─ 返回计划给 Eino ADK
   ↓
3. Executor: 执行每个计划步骤
   ├─ 对每个步骤：
   │   ├─ 如果是 "llm" → executeWithLLM()
   │   ├─ 如果是工具 → executeToolWithRetry()
   │   │   ├─ 重试机制（可配置，指数退避）
   │   │   └─ 失败后降级到 LLM（如启用）
   │   └─ 存储步骤结果到短期记忆
   └─ 返回执行结果
   ↓
4. Replanner: 评估结果，决定是否需要重新规划
   ├─ 分析步骤结果（成功率、置信度）
   ├─ 判断是否需要重规划
   └─ 如需要，生成新计划
   ↓
5. 回到步骤2 或 返回最终结果
```

### 1.4 架构设计原则

本架构设计遵循以下核心原则：

**目标唯一性原则**：整个系统围绕"HLE 考试答题"这一单一、明确的目标进行设计，不试图构建一个能够处理各种类型任务的通用系统。

**最小自主性原则**：自主性是一把双刃剑，自主性越强，系统智能化程度越高，但可控性越差。本架构将系统的自主权限严格限制在"低风险、高确定性"的环节，而将"高风险、高不确定性"的环节交给规则驱动或多模型校验机制。

**可解释性原则**：系统的每一个决策必须有明确的依据。感知层负责提取输入特征并记录原始数据，中枢大脑在做出决策时必须生成决策依据日志（关键决策点标记），执行层的每个操作都有完整的参数记录和结果反馈。

**鲁棒性原则**：确保系统能够应对各种异常情况。工具调用可能失败、模型返回可能异常、输入数据可能不完整——本架构在每个层级都设置了相应的容错机制（重试、降级、错误处理）。

**效率优先原则**：推理规划不追求"最优解"，而是追求"可行解"；任务拆解不追求"极致细粒度"，而是追求"能执行、无歧义"。

---

## 二、Eino ADK 技术架构

### 2.1 Eino ADK 组件集成

Eino ADK 提供了丰富的组件集成能力，本系统使用的核心组件包括：

| 组件类型 | 组件名称 | 用途 | 实现位置 |
|----------|----------|------|----------|
| ChatModel | OpenAI/Claude | LLM调用 | `pkg/llm/client.go` |
| Tool | Python Execute | Python代码执行 | `tools/python_executor/` |
| Tool | SageMath Execute | 数学计算 | `tools/sagemath_executor/` |
| ADK | PlanExecuteAgent | 多智能体协作 | `internal/agent/agent.go` |
| AsyncIterator | Event Stream | 异步事件处理 | `internal/agent/agent.go:Process()` |

### 2.2 Plan-Execute Agent 实现

**实际实现代码：**

```go
// internal/agent/agent.go

// 创建 Planner Agent
plannerAgent, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
    ChatModelWithFormattedOutput: chatModel,
    NewPlan:                      planner.CreatePlan,  // 自定义 Plan 生成函数
})

// 创建 Executor Agent
executorAgent, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
    Model: chatModel,
})

// 创建 Replanner Agent
replannerAgent, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
    ChatModel: chatModel,
    NewPlan:   replanner.CreatePlan,  // 自定义重规划函数
})

// 创建主 Agent
einoAgent, err := planexecute.New(ctx, &planexecute.Config{
    Planner:   plannerAgent,
    Executor:  executorAgent,
    Replanner: replannerAgent,
})
```

### 2.3 工具集成架构

```
工具系统
├── ToolRegistry (工具注册表)
│   └── internal/agent/executor.go
│
├── Python Executor (MCP Tool)
│   ├── 实现：tools/python_executor/
│   ├── 超时：300秒
│   └── 沙箱：支持
│
├── SageMath Executor (MCP Tool)
│   ├── 实现：tools/sagemath_executor/
│   ├── 超时：300秒
│   └── Docker：支持
│
└── 工具执行流程
    ├── Executor.Execute() 接收步骤
    ├── 判断 Action 类型
    ├── 如果是工具 → executeToolWithRetry()
    │   ├─ 重试机制（可配置：max_retries, retry_backoff）
    │   ├─ 指数退避
    │   └─ 失败后降级到 LLM（如启用）
    └── 返回执行结果
```

---

## 三、核心组件设计

### 3.1 Planner（规划器）

Planner 负责理解题目并生成解题计划，是系统的"大脑"入口。

**实现位置**：`internal/agent/planner.go`

**核心功能：**
- ✅ 理解题目意图，提取关键信息
- ✅ 感知层分析（领域识别、特征提取）
- ✅ 检索知识库思维链（如启用）
- ✅ 检索历史上下文（Retriever）
- ✅ 根据题目类型选择合适的解题策略
- ✅ 生成结构化的解题计划（HLEPlan）

**工作流程：**
```
1. 从上下文提取问题（GetQuestion(ctx)）
2. 感知层分析（perception.Analyze(question)）
3. 检索知识库（knowledgeBase.RetrieveSimilar()）
4. 检索历史上下文（retriever.BuildContextForPlanner()）
5. 选择 Prompt 模板（基于领域）
6. 构建完整 Prompt（包含思维链、历史上下文）
7. 调用 LLM 生成计划
8. 解析为 HLEPlan 结构
9. 返回 planexecute.Plan 接口
```

**Prompt设计：**
- 系统提示词：专业的 HLE 考试答题助手
- 上下文注入：知识库思维链（优先） + 历史上下文
- 输出格式：JSON 格式的 HLEPlan

### 3.2 Executor（执行器）

Executor 负责执行Planner生成的每个步骤，是系统的"手"。

**实现位置**：`internal/agent/executor.go`

**核心功能：**
- ✅ 根据Step的Action类型执行相应操作
- ✅ 调用外部工具（Python、SageMath等）
- ✅ 支持重试机制（可配置）
- ✅ 支持降级策略（工具失败降级到 LLM）
- ✅ 返回执行结果

**执行流程：**
```
Step.Action = "llm" → executeWithLLM()
    ├─ 构建 Prompt
    ├─ 调用 LLM
    └─ 返回结果

Step.Action = "tool" → executeToolWithRetry()
    ├─ 获取工具（ToolRegistry.Get()）
    ├─ 执行工具（最多重试 max_retries 次）
    │   ├─ 指数退避（retry_backoff）
    │   └─ 记录每次尝试的耗时
    ├─ 如果失败且启用降级 → executeWithLLM()
    └─ 返回结果
```

**配置项：**
- `max_retries`: 最大重试次数（默认 3）
- `retry_backoff`: 重试退避时间（秒，默认 1）
- `enable_fallback`: 是否启用降级到 LLM（默认 true）

### 3.3 Replanner（重规划器）

Replanner 负责评估执行结果并决定是否需要重规划，是系统的"反思"机制。

**实现位置**：`internal/agent/replanner.go`

**核心功能：**
- ✅ 分析执行结果，判断是否成功
- ✅ 决定是否需要重规划
- ✅ 生成新的计划或结束任务

**重规划策略：**
- 所有步骤成功 → 结束任务，返回结果
- 超过一半步骤失败 → 重新生成计划
- 最后一步失败 → 重试当前步骤

---

## 四、数据流设计

### 4.1 整体数据流

```
用户输入题目（JSONL 或交互式）
    │
    ▼
[main.go] 解析输入
    │
    ▼
[agent.go:Process] 开始处理
    ├─ 创建会话（sessionID）
    ├─ 感知层分析（perception.Analyze）
    ├─ 存储到短期记忆（shortTermMem.AddQuestion）
    ├─ 设置上下文（WithQuestion, WithSessionID）
    ├─ 创建 AgentInput（adk.AgentInput）
    │
    ▼
[Eino ADK Agent] agent.Run(ctx, input)
    │
    ├─ [Planner] CreatePlan()
    │   ├─ 提取问题（GetQuestion(ctx)）
    │   ├─ 感知层分析
    │   ├─ 检索知识库思维链
    │   ├─ 检索历史上下文
    │   ├─ 调用 LLM 生成计划
    │   └─ 返回 HLEPlan
    │
    ├─ [Executor] 执行步骤
    │   ├─ 对每个步骤：
    │   │   ├─ 判断 Action 类型
    │   │   ├─ 执行（LLM 或工具）
    │   │   ├─ 存储结果到记忆
    │   │   └─ 记录性能指标
    │   └─ 返回执行结果
    │
    └─ [Replanner] 评估结果
        ├─ 分析步骤结果
        ├─ 判断是否需要重规划
        └─ 返回新计划或结束
    │
    ▼
[agent.go:Process] 处理事件流
    ├─ 提取 assistant 消息
    ├─ 收集步骤结果
    └─ 如果无答案，从记忆合成
    │
    ▼
[feedback.go] 反馈处理
    ├─ 验证答案
    ├─ 评估置信度
    └─ 生成解释
    │
    ▼
返回 AnswerResult
    ├─ Answer（答案）
    ├─ Explanation（解释）
    ├─ Confidence（置信度）
    ├─ TotalDuration（总耗时）
    └─ Steps（步骤结果列表）
```

### 4.2 状态流转

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Pending │───▶│Executing │───▶│ Evaluating│───▶│Completed │
│  (待执行) │    │ (执行中) │    │ (评估中) │    │ (已完成) │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
                                       │
                                       │ Replan
                                       ▼
                                  ┌──────────┐
                                  │ Replanning│
                                  │ (重规划中) │
                                  └──────────┘
```

### 4.3 上下文传递

**类型安全的 Context Keys：**

```go
// internal/agent/context_keys.go

const (
    ContextKeyQuestion   contextKey = "hle_agent_question"
    ContextKeyUserMessage contextKey = "hle_agent_user_message"
    ContextKeyQuestionID contextKey = "hle_agent_question_id"
    ContextKeySessionID  contextKey = "hle_agent_session_id"
)

// 使用方式
ctx = WithQuestion(ctx, question)
ctx = WithSessionID(ctx, sessionID)
```

---

## 五、技术栈总结

### 5.1 核心技术栈

| 层级 | 技术 | 用途 | 版本/说明 |
|------|------|------|----------|
| **开发语言** | Go | Agent主体开发 | Go 1.25+ |
| **Agent框架** | CloudWeGo Eino ADK | Plan-Execute Agent | v0.7.18 |
| **工具语言** | Python | MCP Tools实现 | Python 3.x |
| **LLM** | OpenAI/Claude/DeepSeek | 推理与规划 | 支持多种兼容 API |
| **数据库** | SQLite | 知识库存储 | 通过 go-sqlite3 |
| **日志** | Zap + Lumberjack | 日志记录与轮转 | zap v1.26.0 |
| **容器化** | Docker | SageMath环境 | 可选 |
| **配置管理** | YAML | 配置文件 | gopkg.in/yaml.v3 |

### 5.2 框架优势

使用Eino ADK Plan-Execute Agent模式的优势：

1. **减少重复工作**：无需自己实现Planner、Executor、Replanner的循环逻辑
2. **标准化接口**：遵循Eino ADK的设计规范
3. **工具集成**：统一管理所有外部工具
4. **容错机制**：内置重试、回滚机制
5. **可观测性**：完整的异步事件流支持

---

## 六、安全与可靠性

### 6.1 安全措施

| 措施 | 说明 | 实现位置 |
|------|------|----------|
| 代码沙箱 | Python代码在隔离环境中执行 | `tools/python_executor/` |
| 资源限制 | CPU、内存、超时限制 | 工具配置（timeout） |
| 输入验证 | 题目数据格式验证 | `internal/config/config.go:Validate()` |
| 错误处理 | 完整的异常捕获与恢复 | `internal/agent/errors.go` |

### 6.2 可靠性保障

| 保障措施 | 说明 | 实现位置 |
|----------|------|----------|
| 重试机制 | 失败步骤自动重试（可配置） | `internal/agent/executor.go:executeToolWithRetry()` |
| 降级策略 | 工具不可用时的备选方案（降级到 LLM） | `internal/agent/executor.go` |
| 日志记录 | 全链路可追溯（支持轮转） | `pkg/logging/logger.go` |
| 性能监控 | 执行时间、输出长度、关键决策点 | 所有核心组件 |

### 6.3 错误处理

**自定义错误类型：**

```go
// internal/agent/errors.go

- AgentError（基础错误）
- ConfigurationError（配置错误）
- ToolNotFoundError（工具未找到）
- ToolExecutionError（工具执行错误）
- LLMInvocationError（LLM调用错误）
```

**错误恢复建议：**
- 每个错误类型都包含 `GetRecoverySuggestion()` 方法
- 提供明确的错误恢复指导

---

## 七、日志与监控

### 7.1 日志系统

**实现位置**：`pkg/logging/logger.go`

**功能特性：**
- ✅ 日志级别控制（debug/info/warn/error）
- ✅ 日志文件轮转（按大小、时间、压缩）
- ✅ 性能指标记录（执行时间、输出长度）
- ✅ 关键决策点标记（decision_point）

**配置项：**
```yaml
logging:
  level: "debug"
  output_path: "./logs/hle-agent.log"
  development: false
  rotation:
    enabled: true
    max_size: 100        # MB
    max_backups: 10
    max_age: 30          # days
    compress: true
    local_time: true
```

### 7.2 性能指标

**记录位置**：所有核心组件的关键方法

**指标类型：**
- 执行时间（duration, duration_seconds）
- 输出长度（output_length, content_length）
- 事件计数（total_events, total_messages）
- 步骤计数（total_steps, plan_steps_count）

### 7.3 关键决策点

**标记位置**：所有重要的决策和状态转换

**决策点示例：**
- `plan_generation_start`
- `knowledge_base_chains_retrieved`
- `tool_execution_success`
- `tool_fallback_to_llm`
- `process_complete`

---

## 八、知识库模块

### 8.1 知识库架构

**实现位置**：`pkg/knowledgebase/`

**核心组件：**
- `knowledgebase.go`: 知识库接口和实现
- `chain_store.go`: SQLite 存储实现
- 数据模型：`ChainOfThought`, `ReasoningStep`

**功能特性：**
- ✅ 持久化存储（SQLite）
- ✅ 相似度检索（关键词、领域、复杂度）
- ✅ 格式化输出（供 LLM 使用）
- ✅ 批量导入工具

### 8.2 集成点

**Planner 集成：**
- 在生成计划时检索相似思维链
- 将思维链作为上下文注入 Prompt
- 优先显示思维链，再显示历史上下文

---

## 九、总结

本架构设计基于 CloudWeGo Eino ADK 的 Plan-Execute Agent 模式，具有以下特点：

1. **框架驱动**：利用Eino ADK预构建框架，减少重复开发
2. **层次清晰**：感知、记忆、规划、执行、反馈，职责分明
3. **工具集成**：统一管理Python、SageMath等外部工具
4. **可扩展性**：支持添加新的工具和策略
5. **可控性强**：完整的容错和重试机制
6. **可观测性**：完整的日志、性能监控和关键决策点追踪
7. **知识增强**：通过知识库提升 LLM 推理能力

---

**文档版本**：V2.0  
**更新说明**：基于实际实现情况更新架构文档，反映知识库、日志系统等新增功能


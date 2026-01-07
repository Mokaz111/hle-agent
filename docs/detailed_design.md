# AI Agent 详细设计文档 - HLE 考试答题系统

## 文档信息

| 项目 | 内容 |
|------|------|
| **项目名称** | HLE 考试答题智能体系统 |
| **关联文档** | 需求文档 V2.0、架构设计文档 V2.0 |
| **版本** | V2.0 |
| **文档状态** | 已实现 |
| **创建日期** | 2026-01-05 |
| **更新日期** | 2026-01-XX |
| **技术框架** | CloudWeGo Eino ADK |

---

## 一、项目结构

### 1.1 目录结构

```
hle-agent/
├── cmd/
│   ├── agent/                    # 主程序入口
│   │   └── main.go               # 命令行工具（交互式/批处理/标准输入）
│   └── import-chains/            # 知识库导入工具
│       └── main.go               # 批量导入思维链
│
├── internal/
│   ├── agent/                    # Agent 核心实现
│   │   ├── agent.go              # HLEAgent 主类（Eino ADK 集成）
│   │   ├── planner.go            # Planner 实现（规划器）
│   │   ├── executor.go           # Executor 实现（执行器）
│   │   ├── replanner.go          # Replanner 实现（重规划器）
│   │   ├── context_keys.go       # 类型安全的 Context Keys
│   │   ├── errors.go             # 自定义错误类型
│   │   └── *_test.go             # 单元测试
│   │
│   ├── config/                   # 配置管理
│   │   └── config.go              # YAML 配置加载与验证
│   │
│   └── models/                    # 数据模型
│       └── question.go           # Question, Plan, Step, StepResult 等
│
├── pkg/
│   ├── llm/                      # LLM 客户端
│   │   └── client.go             # OpenAI/Claude 等模型封装
│   │
│   ├── memory/                   # 记忆模块
│   │   └── short_term.go         # 短期记忆实现
│   │
│   ├── retriever/                # 检索模块
│   │   └── retriever.go          # 历史上下文检索
│   │
│   ├── knowledgebase/            # 知识库模块
│   │   ├── knowledgebase.go      # 知识库接口与实现
│   │   └── chain_store.go        # SQLite 存储实现
│   │
│   ├── perception/               # 感知层
│   │   └── perception.go         # 题目解析与领域识别
│   │
│   ├── prompts/                  # Prompt 管理
│   │   └── templates.go           # Prompt 模板定义
│   │
│   ├── feedback/                 # 反馈层
│   │   └── feedback.go           # 答案验证与置信度评估
│   │
│   └── logging/                  # 日志系统
│       └── logger.go             # Zap + Lumberjack 日志实现
│
├── tools/                        # 外部工具（MCP）
│   ├── python_executor/          # Python 执行器
│   └── sagemath_executor/        # SageMath 执行器
│
├── config.yaml                   # 主配置文件
├── go.mod                        # Go 模块定义
└── docs/                         # 文档目录
    ├── requirements.md           # 需求文档
    ├── architecture.md           # 架构设计文档
    ├── detailed_design.md        # 详细设计文档（本文档）
    ├── eino.md                   # Eino ADK 知识文档
    ├── hle.md                    # HLE 考试说明
    └── HLE_text_only_10questions.jsonl  # 测试数据集
```

### 1.2 核心模块说明

| 模块 | 路径 | 职责 |
|------|------|------|
| **Agent 核心** | `internal/agent/` | Eino ADK Plan-Execute Agent 实现 |
| **LLM 客户端** | `pkg/llm/` | 统一 LLM 调用接口 |
| **记忆系统** | `pkg/memory/` | 短期记忆管理 |
| **检索系统** | `pkg/retriever/` | 历史上下文检索 |
| **知识库** | `pkg/knowledgebase/` | 思维链存储与检索 |
| **感知层** | `pkg/perception/` | 题目解析与领域识别 |
| **反馈层** | `pkg/feedback/` | 答案验证与评估 |
| **日志系统** | `pkg/logging/` | 日志记录与轮转 |

---

## 二、核心数据模型

### 2.1 问题模型（Question）

**定义位置**：`internal/models/question.go`

```go
type Question struct {
    ID         string                 `json:"id"`
    Category   string                 `json:"category"`
    Content    string                 `json:"content"`
    Options    []string               `json:"options,omitempty"`
    CorrectAns string                 `json:"correct_answer"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
```

**字段说明**：
- `ID`: 题目唯一标识
- `Category`: 题目类别（如 cryptography, programming, calculation）
- `Content`: 题目内容
- `Options`: 选项列表（选择题）
- `CorrectAns`: 正确答案
- `Metadata`: 扩展元数据

### 2.2 计划模型（HLEPlan）

**定义位置**：`internal/agent/planner.go`

```go
type HLEPlan struct {
    ID          string    `json:"id"`
    QuestionID  string    `json:"question_id,omitempty"`
    Steps       []HLEStep `json:"steps"`
    FirstStepID string    `json:"first_step_id"`
}

type HLEStep struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Action      string `json:"action"`        // "llm" 或工具名
    ToolName    string `json:"tool_name,omitempty"`
    Params      string `json:"parameters,omitempty"`
}
```

**实现接口**：`planexecute.Plan`

**关键方法**：
- `FirstStep() string`: 返回第一个步骤 ID（Eino ADK 要求）

### 2.3 步骤结果模型（HLEStepResult）

**定义位置**：`internal/agent/executor.go`

```go
type HLEStepResult struct {
    StepID     string  `json:"id"`
    Success    bool    `json:"success"`
    Error      string  `json:"error,omitempty"`
    Output     string  `json:"output"`
    Confidence float64 `json:"confidence"`
}
```

### 2.4 最终结果模型（AnswerResult）

**定义位置**：`internal/agent/agent.go`

```go
type AnswerResult struct {
    QuestionID    string           `json:"question_id"`
    Answer        string           `json:"answer"`
    Explanation   string           `json:"explanation"`
    Confidence    float64          `json:"confidence"`
    Steps         []*StepResult    `json:"steps"`
    TotalDuration float64          `json:"total_duration"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
}
```

### 2.5 知识库模型（ChainOfThought）

**定义位置**：`pkg/knowledgebase/knowledgebase.go`

```go
type ChainOfThought struct {
    ID          string          `json:"id"`
    Question    string          `json:"question"`
    Keywords    []string        `json:"keywords"`
    Domain      string          `json:"domain"`
    Complexity  int             `json:"complexity"`  // 1-5
    Steps       []ReasoningStep `json:"steps"`
    FinalAnswer string          `json:"final_answer"`
    Source      string          `json:"source"`     // 来源（如 "gpt-4"）
    CreatedAt   time.Time       `json:"created_at"`
}

type ReasoningStep struct {
    StepNumber int    `json:"step_number"`
    Thought    string `json:"thought"`
    Action     string `json:"action"`
    Result     string `json:"result"`
}
```

---

## 三、核心组件详细设计

### 3.1 HLEAgent（主 Agent）

**实现位置**：`internal/agent/agent.go`

#### 3.1.1 结构定义

```go
type HLEAgent struct {
    cfg          *config.AgentConfig
    llmClient    *llm.Client
    planner      *Planner
    executor     *Executor
    replanner    *Replanner
    agent        *planexecute.Agent  // Eino ADK Agent
    toolRegistry *ToolRegistry
    shortTermMem *memory.ShortTermMemory
    perception   *perception.Perception
    retriever    *retriever.Retriever
    feedback     *feedback.Feedback
    knowledgeBase knowledgebase.KnowledgeBase  // 知识库（可选）
    logger       *zap.Logger
    agentName    string
}
```

#### 3.1.2 初始化流程（NewHLEAgent）

```
1. 加载配置（config.AgentConfig）
2. 初始化 LLM 客户端（llm.NewClient）
3. 初始化短期记忆（memory.NewShortTermMemory）
4. 初始化检索器（retriever.NewRetriever）
5. 初始化知识库（如启用，knowledgebase.NewKnowledgeBase）
6. 初始化 Planner（NewPlanner）
7. 初始化 Executor（NewExecutor，包含配置）
8. 初始化 Replanner（NewReplanner）
9. 创建 Eino ADK Agent：
   - planexecute.NewPlanner（使用自定义 CreatePlan）
   - planexecute.NewExecutor
   - planexecute.NewReplanner（使用自定义 CreatePlan）
   - planexecute.New（组合三个组件）
10. 初始化感知层（perception.NewPerception）
11. 初始化反馈层（feedback.NewFeedback）
12. 注册工具（Python、SageMath）
```

#### 3.1.3 处理流程（Process）

**方法签名**：
```go
func (a *HLEAgent) Process(ctx context.Context, question string) (*AnswerResult, error)
```

**详细流程**：

```
1. 创建会话（sessionID = uuid.New().String()）
   - 记录开始时间（processStartTime）
   - 日志：process_start

2. 感知层分析
   - 调用 perception.Analyze(question)
   - 记录耗时（perceptionAnalysisDuration）
   - 日志：perception_analysis_complete

3. 存储到短期记忆
   - shortTermMem.AddQuestion(sessionID, questionID, question)
   - 日志：question_stored

4. 设置上下文
   - ctx = WithQuestion(ctx, question)
   - ctx = WithSessionID(ctx, sessionID)
   - ctx = WithQuestionID(ctx, questionID)

5. 创建 AgentInput
   - input := adk.AgentInput{UserMessage: question}

6. 运行 Eino ADK Agent
   - iterator := a.agent.Run(ctx, input)
   - 记录开始时间（agentRunStartTime）
   - 日志：eino_agent_started

7. 处理事件流（AsyncIterator）
   for {
       event, hasMore := iterator.Next()
       if !hasMore { break }
       
       switch event.Type {
       case adk.EventTypeAgentMessage:
           // 提取 assistant 消息作为答案
           msg := adk.GetMessage(event)
           if msg.Role == "assistant" {
               lastMessage = msg
           }
           // 日志：agent_message_received
           
       case adk.EventTypeStepResult:
           // 收集步骤结果
           // 存储到记忆
           
       case adk.EventTypeError:
           // 处理错误
       }
   }
   - 记录耗时（agentRunDuration）
   - 日志：event_processing_complete

8. 答案合成（降级处理）
   - 如果 lastMessage 为空：
     - 从记忆获取步骤结果
     - 调用 synthesizeAnswer() 生成答案
   - 记录耗时（answerSynthesisDuration）
   - 日志：answer_synthesis_start/success/failed

9. 反馈处理
   - feedback.Process(questionID, question, answer, stepResults)
   - 提取解释和置信度

10. 返回结果
    - 构建 AnswerResult
    - 记录总耗时（totalDuration）
    - 日志：process_complete
```

#### 3.1.4 批处理流程（ProcessBatch）

**方法签名**：
```go
func (a *HLEAgent) ProcessBatch(ctx context.Context, questions []map[string]interface{}) ([]*AnswerResult, error)
```

**特点**：
- 支持上下文取消（`ctx.Done()`）
- 每个问题独立处理
- 返回结果列表

---

### 3.2 Planner（规划器）

**实现位置**：`internal/agent/planner.go`

#### 3.2.1 结构定义

```go
type Planner struct {
    llmClient     *llm.Client
    prompts       *prompts.PromptManager
    retriever     *retriever.Retriever
    knowledgeBase knowledgebase.KnowledgeBase  // 知识库（可选）
    perception    *perception.Perception
    logger        *zap.Logger
}
```

#### 3.2.2 CreatePlan（Eino ADK 接口）

**方法签名**：
```go
func (p *Planner) CreatePlan(ctx context.Context) planexecute.Plan
```

**实现逻辑**：
1. 从上下文提取问题（`GetQuestion(ctx)`）
2. 如果问题为空，返回默认计划
3. 调用 `p.Plan(ctx, question)` 生成计划
4. 如果失败，返回默认计划（确保不为空）

#### 3.2.3 Plan（核心规划逻辑）

**方法签名**：
```go
func (p *Planner) Plan(ctx context.Context, question string) (*HLEPlan, error)
```

**详细流程**：

```
1. 记录开始时间（planStartTime）
   - 日志：plan_generation_start

2. 感知层分析
   - analysis := p.perception.Analyze(question)
   - 记录耗时（perceptionAnalysisDuration）
   - 日志：perception_analysis_complete

3. 检索知识库思维链（如启用）
   - chains := p.knowledgeBase.RetrieveSimilar(question, maxResults)
   - 记录耗时（knowledgeBaseRetrievalDuration）
   - 格式化思维链：p.knowledgeBase.FormatChainsForLLM(chains)
   - 日志：knowledge_base_chains_retrieved / no_knowledge_base_chains

4. 检索历史上下文
   - context := p.retriever.BuildContextForPlanner(question)
   - 记录耗时（historicalContextRetrievalDuration）
   - 日志：historical_context_retrieved / no_historical_context

5. 选择 Prompt 模板
   - domain := analysis.DomainRecognition.PrimaryDomain
   - template := p.prompts.GetTemplate("planner_" + domain)

6. 构建完整 Prompt
   - 优先注入知识库思维链（如有）
   - 然后注入历史上下文（如有）
   - 最后注入题目
   - 记录 Prompt 长度（promptLength）

7. 调用 LLM 生成计划
   - response := p.llmClient.GenerateWithSystemPrompt(ctx, systemPrompt, prompt)
   - 记录耗时（llmInvocationDuration）
   - 记录响应长度（responseLength）
   - 日志：llm_response_received

8. 解析计划
   - plan := p.parsePlan(response, question)
   - 记录耗时（planParsingDuration）
   - 日志：plan_generation_success / plan_generation_failed

9. 返回计划
   - 记录总耗时（totalPlanningDuration）
```

#### 3.2.4 parsePlan（计划解析）

**实现逻辑**：
1. 尝试 JSON 解析
2. 提取步骤列表
3. 验证步骤格式
4. 生成步骤 ID（如缺失）
5. 返回 `HLEPlan` 结构

---

### 3.3 Executor（执行器）

**实现位置**：`internal/agent/executor.go`

#### 3.3.1 结构定义

```go
type Executor struct {
    toolRegistry   *ToolRegistry
    llmClient      *llm.Client
    logger         *zap.Logger
    maxRetries     int              // 最大重试次数（可配置）
    retryBackoff   time.Duration    // 重试退避时间（可配置）
    enableFallback bool              // 是否启用降级（可配置）
}
```

#### 3.3.2 Execute（执行步骤）

**方法签名**：
```go
func (e *Executor) Execute(ctx context.Context, step planexecute.Step) (planexecute.StepResult, error)
```

**详细流程**：

```
1. 记录开始时间（stepExecutionStartTime）
   - 日志：step_execution_start

2. 判断 Action 类型
   - 如果是 "llm" → executeWithLLM()
   - 如果是工具名 → executeToolWithRetry()

3. 记录执行结果
   - 记录耗时（stepExecutionDuration）
   - 记录输出长度（outputLength）
   - 日志：step_execution_success / step_execution_failed
```

#### 3.3.3 executeWithLLM（LLM 执行）

**实现逻辑**：
1. 构建 Prompt（包含步骤描述和上下文）
2. 调用 LLM（`e.llmClient.GenerateWithSystemPrompt`）
3. 记录耗时和输出长度
4. 返回结果（置信度 = 0.8）

#### 3.3.4 executeToolWithRetry（工具执行，支持重试）

**实现逻辑**：

```
1. 获取工具（ToolRegistry.Get(toolName)）
   - 如果工具不存在，返回错误
   - 日志：tool_execution_prepare

2. 重试循环（最多 maxRetries 次）
   for attempt := 0; attempt <= e.maxRetries; attempt++ {
       if attempt > 0 {
           // 指数退避
           backoff := e.retryBackoff * time.Duration(1<<uint(attempt-1))
           time.Sleep(backoff)
           - 日志：tool_retry_start
       }
       
       // 执行工具
       result, err := tool.Execute(ctx, params)
       - 记录耗时（toolExecutionDuration）
       
       if err == nil {
           if attempt > 0 {
               - 日志：tool_retry_success
           } else {
               - 日志：tool_first_attempt_success
           }
           - 日志：tool_execution_success
           return result
       }
   }

3. 如果所有重试都失败
   - 日志：tool_execution_failed_after_retry

4. 降级处理（如启用）
   if e.enableFallback {
       - 日志：tool_fallback_to_llm
       return e.executeWithLLM(ctx, step)
   }

5. 返回错误
```

---

### 3.4 Replanner（重规划器）

**实现位置**：`internal/agent/replanner.go`

#### 3.4.1 结构定义

```go
type Replanner struct {
    llmClient *llm.Client
    prompts   *prompts.PromptManager
    logger    *zap.Logger
}
```

#### 3.4.2 CreatePlan（Eino ADK 接口）

**方法签名**：
```go
func (r *Replanner) CreatePlan(ctx context.Context) planexecute.Plan
```

**实现逻辑**：
1. 从上下文提取问题
2. 返回一个简单的重规划计划（实际重规划逻辑在 `Replan` 方法中）

#### 3.4.3 Replan（重规划逻辑）

**方法签名**：
```go
func (r *Replanner) Replan(ctx context.Context, plan *HLEPlan, stepResults []*HLEStepResult, question string) (*HLEPlan, error)
```

**详细流程**：

```
1. 分析步骤结果
   - analysis := r.analyzeStepResults(stepResults)
   - 计算平均置信度、失败步骤等

2. 判断是否需要重规划
   - 如果平均置信度 < 0.7 → 需要重规划
   - 如果所有步骤失败 → 需要重规划
   - 否则，返回原计划

3. 生成新计划（如需要）
   - 构建上下文（包含原计划、执行结果、分析）
   - 调用 LLM 生成新计划
   - 解析并返回新计划
```

#### 3.4.4 analyzeStepResults（结果分析）

**实现逻辑**：
1. 计算平均置信度
2. 统计失败步骤
3. 判断是否需要重规划（基于阈值）

---

### 3.5 知识库模块

**实现位置**：`pkg/knowledgebase/`

#### 3.5.1 接口定义

```go
type KnowledgeBase interface {
    SaveChain(ctx context.Context, chain *ChainOfThought) error
    RetrieveSimilar(ctx context.Context, question string, maxResults int) ([]*ChainOfThought, error)
    FormatChainsForLLM(chains []*ChainOfThought) string
    GetChainCount(ctx context.Context) (int, error)
}
```

#### 3.5.2 存储实现（SQLite）

**实现位置**：`pkg/knowledgebase/chain_store.go`

**核心方法**：
- `SaveChain`: 保存思维链到数据库
- `RetrieveSimilar`: 基于关键词、领域、复杂度检索相似思维链
- `FormatChainsForLLM`: 格式化思维链为 LLM 可读格式

**检索策略**：
1. 关键词匹配（权重：`weight_keywords`）
2. 领域匹配（权重：`weight_domain`）
3. 复杂度匹配（权重：`weight_complexity`）
4. 综合评分排序，返回 Top N

---

### 3.6 短期记忆模块

**实现位置**：`pkg/memory/short_term.go`

#### 3.6.1 结构定义

```go
type ShortTermMemory struct {
    sessions map[string]*Session
    mu       sync.RWMutex
    maxSize  int  // 最大会话数
}

type Session struct {
    ID          string
    Question    string
    Plan        *models.Plan
    StepResults []*models.StepResult
    Reasoning   string
    FinalAnswer string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### 3.6.2 核心方法

- `AddQuestion(sessionID, questionID, question)`: 添加问题
- `AddPlan(sessionID, plan)`: 添加计划
- `AddStepResult(sessionID, stepResult)`: 添加步骤结果
- `GetSession(sessionID)`: 获取会话
- `FormatContextForLLM(sessionID)`: 格式化上下文供 LLM 使用
- `Prune()`: 清理过期会话（防止内存泄漏）

---

### 3.7 检索器模块

**实现位置**：`pkg/retriever/retriever.go`

#### 3.7.1 核心方法

- `BuildContextForPlanner(question)`: 从短期记忆检索相似历史问题，构建上下文
- `FindSimilarQuestions(question, maxResults)`: 基于关键词和领域匹配查找相似问题

---

### 3.8 感知层模块

**实现位置**：`pkg/perception/perception.go`

#### 3.8.1 核心方法

- `Analyze(question)`: 完整分析题目
  - 解析题目（Parser）
  - 识别领域（Recognizer）
  - 返回 `AnalysisResult`

#### 3.8.2 领域识别

**支持的领域**：
- `DomainCryptography`: 密码学
- `DomainProgramming`: 编程
- `DomainCalculation`: 数学计算
- `DomainUnknown`: 未知

---

### 3.9 反馈层模块

**实现位置**：`pkg/feedback/feedback.go`

#### 3.9.1 核心方法

- `Process(questionID, question, answer, stepResults)`: 综合反馈处理
  - 验证答案（Validator）
  - 评估置信度（Classifier）
  - 返回 `FeedbackRecord`

---

### 3.10 日志系统

**实现位置**：`pkg/logging/logger.go`

#### 3.10.1 初始化

```go
func Init(cfg *config.LoggingConfig) error
```

**功能**：
- 支持日志级别（debug/info/warn/error）
- 支持文件输出（可配置路径）
- 支持日志轮转（Lumberjack）：
  - 按大小轮转（max_size）
  - 按时间轮转（max_age）
  - 压缩旧日志（compress）
  - 保留备份数（max_backups）

#### 3.10.2 性能指标记录

**记录位置**：所有核心组件的关键方法

**指标类型**：
- `duration` / `duration_seconds`: 执行时间
- `output_length` / `content_length`: 输出长度
- `prompt_length`: Prompt 长度
- `response_length`: 响应长度

#### 3.10.3 关键决策点标记

**标记位置**：所有重要的决策和状态转换

**决策点示例**：
- `process_start`
- `plan_generation_start`
- `knowledge_base_chains_retrieved`
- `tool_execution_success`
- `tool_fallback_to_llm`
- `process_complete`

---

## 四、工具集成

### 4.1 工具接口

**定义位置**：`internal/agent/executor.go`

```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, params string) (string, error)
}
```

### 4.2 工具注册表

**实现位置**：`internal/agent/executor.go`

```go
type ToolRegistry struct {
    tools  map[string]Tool
    logger *zap.Logger
}
```

**核心方法**：
- `Register(tool)`: 注册工具
- `Get(name)`: 获取工具
- `List()`: 列出所有工具

### 4.3 工具实现

#### 4.3.1 Python Executor

**位置**：`tools/python_executor/`

**功能**：
- 执行 Python 代码
- 支持超时（300秒）
- 支持沙箱隔离

#### 4.3.2 SageMath Executor

**位置**：`tools/sagemath_executor/`

**功能**：
- 执行 SageMath 代码
- 支持超时（300秒）
- Docker 容器化部署

---

## 五、配置管理

### 5.1 配置文件结构

**位置**：`config.yaml`

**主要配置项**：

```yaml
# LLM 配置
model:
  provider: "openai"
  model: "gpt-4"
  api_key: "..."
  base_url: "..."
  temperature: 0.7
  max_tokens: 2000

# Agent 配置
agent:
  executor:
    max_retries: 3
    retry_backoff: 1s
    enable_fallback: true

# 知识库配置
knowledge_base:
  enabled: true
  storage_path: "./data/knowledge_base.db"
  max_results: 5
  min_similarity: 0.3
  weight_keywords: 0.4
  weight_domain: 0.3
  weight_complexity: 0.3

# 日志配置
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

### 5.2 配置加载

**实现位置**：`internal/config/config.go`

**核心方法**：
- `Load(path)`: 加载配置文件
- `Validate()`: 验证配置有效性
- `postProcess()`: 后处理配置（设置默认值等）

---

## 六、错误处理

### 6.1 自定义错误类型

**定义位置**：`internal/agent/errors.go`

**错误类型**：
- `AgentError`: 基础错误类
- `PlanGenerationError`: 计划生成错误
- `StepExecutionError`: 步骤执行错误
- `ToolExecutionError`: 工具执行错误
- `LLMInvocationError`: LLM 调用错误
- `ReplanningError`: 重规划错误
- `ToolNotFoundError`: 工具未找到错误
- `ConfigurationError`: 配置错误

### 6.2 错误恢复建议

**方法**：`GetRecoverySuggestion()`

每个错误类型都提供明确的恢复建议，例如：
- 工具执行失败 → "建议：检查工具配置是否正确，或尝试使用 LLM 推理替代"
- LLM 调用失败 → "建议：检查 LLM 配置和网络连接，或稍后重试"

---

## 七、上下文传递

### 7.1 类型安全的 Context Keys

**定义位置**：`internal/agent/context_keys.go`

**Context Keys**：
- `ContextKeyQuestion`: 题目
- `ContextKeyUserMessage`: 用户消息
- `ContextKeyQuestionID`: 题目ID
- `ContextKeySessionID`: 会话ID

**辅助函数**：
- `WithQuestion(ctx, question)`: 设置题目
- `GetQuestion(ctx)`: 获取题目
- `WithSessionID(ctx, sessionID)`: 设置会话ID
- `GetSessionID(ctx)`: 获取会话ID

---

## 八、命令行工具

### 8.1 主程序（cmd/agent/main.go）

**运行模式**：

1. **交互式模式**（默认）
   ```bash
   ./hle-agent
   ```
   - 支持多行输入
   - 实时显示结果

2. **批处理模式**
   ```bash
   ./hle-agent --batch --input questions.jsonl --output results/
   ```
   - 从 JSONL 文件读取问题
   - 批量处理并保存结果

3. **标准输入模式**
   ```bash
   echo '{"id":"1","question":"..."}' | ./hle-agent --stdin
   ```
   - 从标准输入读取 JSON
   - 输出结果到标准输出

### 8.2 知识库导入工具（cmd/import-chains/main.go）

**功能**：
- 从 JSON/JSONL 文件导入思维链
- 批量导入到知识库

**使用示例**：
```bash
./import-chains --input chains.jsonl --config config.yaml
```

---

## 九、测试

### 9.1 单元测试

**测试文件**：
- `internal/agent/agent_test.go`
- `internal/agent/planner_test.go`
- `internal/agent/executor_test.go`
- `internal/agent/replanner_test.go`

### 9.2 集成测试

**测试文件**：
- `internal/agent/integration_test.go`

---

## 十、性能优化

### 10.1 已实现的优化

1. **短期记忆修剪**：防止内存泄漏
2. **日志轮转**：防止日志文件过大
3. **重试机制**：提高工具执行成功率
4. **降级策略**：工具失败时降级到 LLM

### 10.2 性能指标

**记录位置**：所有核心方法

**指标类型**：
- 执行时间（duration）
- 输出长度（output_length）
- Prompt 长度（prompt_length）
- 响应长度（response_length）

---

## 十一、总结

本详细设计文档基于实际代码实现，详细描述了：

1. **项目结构**：完整的目录组织和模块划分
2. **数据模型**：所有核心数据结构定义
3. **组件设计**：每个核心组件的详细实现逻辑
4. **工具集成**：外部工具的集成方式
5. **配置管理**：配置文件的加载和验证
6. **错误处理**：自定义错误类型和恢复建议
7. **上下文传递**：类型安全的 Context 使用
8. **命令行工具**：主程序和导入工具的使用方法

所有设计均基于实际代码实现，确保文档与代码的一致性。

---

**文档版本**：V2.0  
**更新说明**：基于实际实现情况更新详细设计文档，反映知识库、日志系统、配置管理等新增功能


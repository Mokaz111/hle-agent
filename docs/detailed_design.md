# AI Agent 详细设计文档 - HLE 考试答题系统

## 文档信息

| 项目 | 内容 |
|------|------|
| **项目名称** | HLE 考试答题智能体系统 |
| **关联文档** | 需求文档 V1.0、总体架构设计文档 V1.0 |
| **版本** | V2.0 |
| **文档状态** | 待评审 |
| **创建日期** | 2026-01-05 |
| **主要语言** | Go |
| **框架** | CloudWeGo Eino ADK |
| **工具语言** | Python |

---

## 框架选型说明

本系统基于 **CloudWeGo Eino ADK** 的 **Plan-Execute Agent** 模式进行设计，充分利用框架提供的能力：

- **PlanExecuteAgent**: 预构建的多智能体协作框架，实现"规划-执行-反思"范式
- **ToolsNode**: 工具调用节点，统一管理所有工具
- **Callback**: 回调机制，用于监控和调试
- **CheckPoint**: 检查点机制，支持中断和恢复

### Eino ADK Plan-Execute Agent 架构

```
┌─────────────────────────────────────────────────────────────────┐
│              Eino ADK PlanExecuteAgent                          │
│                                                                 │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│  │  Planner    │────▶│  Executor   │────▶│ Replanner   │       │
│  │  (规划器)   │     │  (执行器)   │     │  (重规划器) │       │
│  └─────────────┘     └─────────────┘     └─────────────┘       │
│         │                  │                   │                │
│         │                  ▼                   │                │
│         │           ┌─────────────┐            │                │
│         │           │   Tools     │◀───────────┘                │
│         │           │  (Python)   │                             │
│         │           └─────────────┘                             │
│         │                                                       │
└─────────┼───────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Eino ADK Core                                │
│  - ChatModel (LLM集成)                                          │
│  - ToolsNode & Tool (工具调用)                                  │
│  - Callback (回调机制)                                          │
│  - Interrupt & CheckPoint (中断与检查点)                        │
└─────────────────────────────────────────────────────────────────┘
```

### 使用 Eino ADK 的优势

1. **减少重复工作**: 无需自己实现Planner、Executor、Replanner
2. **标准化接口**: 遵循Eino ADK的设计规范
3. **工具集成**: 统一管理所有外部工具
4. **容错机制**: 内置重试、回滚机制
5. **可观测性**: 完整的Callback支持

---

## 一、项目结构设计

### 1.1 整体目录结构

```
hle-agent/
├── cmd/
│   └── agent/
│       └── main.go                    # 程序入口，使用Eino ADK
├── internal/
│   ├── config/
│   │   └── config.go                  # 配置管理
│   ├── models/
│   │   ├── question.go                # 题目数据结构
│   │   └── response.go                # 响应数据结构
│   ├── agent/
│   │   ├── planner.go                 # 自定义Planner（题目理解+策略规划）
│   │   ├── executor.go                # 自定义Executor（解题执行）
│   │   └── replanner.go               # 自定义Replanner（结果评估+重规划）
│   ├── tools/
│   │   ├── python.go                  # Python执行器（Tool包装）
│   │   ├── sagemath.go                # SageMath执行器（Tool包装）
│   │   └── retriever.go               # 知识检索器（Tool包装）
│   └── utils/
│       └── parser.go                  # JSONL题目解析
├── pkg/
│   ├── llm/
│   │   └── client.go                  # LLM客户端配置
│   └── prompts/
│       └── templates.go               # Prompt模板
├── tools/                             # Python MCP Tools
│   ├── python_executor/
│   │   ├── executor.py                # Python代码执行
│   │   └── requirements.txt
│   └── sagemath_executor/
│       ├── executor.py                # SageMath执行
│       └── Dockerfile
├── scripts/
│   └── run_eval.sh                    # 评估脚本
├── testdata/
│   └── questions.jsonl                # 测试题目
├── config.yaml                        # 配置文件
├── go.mod
├── go.sum
└── README.md
```

### 1.2 基于Eino ADK的核心组件依赖

```
                    ┌─────────────────────────────────────┐
                    │      cmd/main.go                    │
                    │  (创建PlanExecuteAgent)             │
                    └──────────────────┬──────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Eino ADK PlanExecuteAgent                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  Planner: 自定义题目理解 + 策略规划                                   │   │
│   │  - 接收题目，理解题意                                                │   │
│   │  - 规划解题步骤                                                      │   │
│   │  - 生成结构化Plan（Step + Action）                                   │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                       │                                      │
│                                       ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  Executor: 执行每个解题步骤                                          │   │
│   │  - 接收Plan中的Step                                                 │   │
│   │  - 根据Action调用对应Tool                                           │   │
│   │  - 返回执行结果                                                      │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                       │                                      │
│                                       ▼                                      │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │  Replanner: 评估结果 + 决定是否需要重规划                            │   │
│   │  - 评估执行结果                                                      │   │
│   │  - 判断是否成功/失败/需要重试                                        │   │
│   │  - 生成新Plan或结束任务                                              │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                              Tools (Eino Tools)                              │
│   ┌───────────────┐  ┌───────────────┐  ┌───────────────┐                   │
│   │ PythonExecute │  │ SageMathExec  │  │ KnowledgeBase │                   │
│   │  (Python)     │  │  (Python)     │  │    (Go/DB)    │                   │
│   └───────────────┘  └───────────────┘  └───────────────┘                   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 二、核心数据结构设计

### 2.1 题目相关结构

```go
package models

// Question represents a single HLE question
type Question struct {
    ID          string       `json:"id"`                    // 题目唯一标识
    Content     string       `json:"question"`              // 题目内容（原始）
    Answer      string       `json:"answer"`                 // 标准答案
    AnswerType  AnswerType   `json:"answer_type"`            // 答案类型
    Rationale   string       `json:"rationale"`              // 解题分析
    RawSubject  string       `json:"raw_subject"`            // 二级场景类型
    Category    string       `json:"category"`               // 一级场景类型
    
    // 衍生字段
    Fingerprint string       `json:"fingerprint,omitempty"`
    Difficulty  Difficulty   `json:"difficulty,omitempty"`
    Keywords    []string     `json:"keywords,omitempty"`
    Domain      Domain       `json:"domain,omitempty"`
}

// AnswerType defines the type of answer expected
type AnswerType string

const (
    AnswerTypeExactMatch    AnswerType = "exactMatch"
    AnswerTypeMultipleChoice AnswerType = "multipleChoice"
)

// Difficulty represents the estimated difficulty level
type Difficulty string

const (
    DifficultyEasy   Difficulty = "easy"
    DifficultyMedium Difficulty = "medium"
    DifficultyHard   Difficulty = "hard"
)

// Domain represents the subject domain of a question
type Domain string

const (
    DomainCybersecurity     Domain = "Cybersecurity"
    DomainCryptography      Domain = "Cryptography"
    DomainProgramming       Domain = "Programming"
    DomainRobotics          Domain = "Robotics"
    DomainArtificialIntelligence Domain = "Artificial Intelligence"
    DomainDataScience       Domain = "Data Science"
    DomainMachineLearning   Domain = "Machine Learning"
)
```

### 2.2 Plan结构（Eino ADK标准格式）

```go
package models

// Plan represents a解题 plan for Eino ADK PlanExecuteAgent
type Plan struct {
    ID          string      `json:"id"`            // Plan ID
    QuestionID  string      `json:"question_id"`   // 题目ID
    Steps       []Step      `json:"steps"`         // 解题步骤
    TotalSteps  int         `json:"total_steps"`   // 总步骤数
    CurrentStep int         `json:"current_step"`  // 当前步骤
    Status      PlanStatus  `json:"status"`        // Plan状态
}

// Step represents a single step in the plan
type Step struct {
    ID          int          `json:"id"`            // 步骤ID
    Description string       `json:"description"`   // 步骤描述
    Action      string       `json:"action"`        // 执行动作 (LLM/Tool)
    ToolName    string       `json:"tool_name,omitempty"` // 工具名称
    Parameters  map[string]interface{} `json:"parameters,omitempty"` // 参数
    InputFrom   []int        `json:"input_from,omitempty"` // 输入来源
    ExpectedOutput string    `json:"expected_output"` // 预期输出
    IsKeyPoint  bool         `json:"is_key_point"`  // 是否关键节点
    RetryOnFail bool         `json:"retry_on_fail"` // 失败时是否重试
}

// PlanStatus represents the status of a plan
type PlanStatus string

const (
    PlanStatusPending    PlanStatus = "pending"     // 待执行
    PlanStatusExecuting  PlanStatus = "executing"   // 执行中
    PlanStatusCompleted  PlanStatus = "completed"   // 已完成
    PlanStatusFailed     PlanStatus = "failed"      // 失败
    PlanStatusNeedReplan PlanStatus = "need_replan" // 需要重规划
)
```

### 2.3 StepResult结构

```go
package models

// StepResult represents the result of a step execution
type StepResult struct {
    StepID      int         `json:"step_id"`        // 步骤ID
    Success     bool        `json:"success"`        // 是否成功
    Output      interface{} `json:"output"`         // 输出结果
    Error       string      `json:"error,omitempty"` // 错误信息
    Confidence  float64     `json:"confidence"`     // 置信度
    Reasoning   string      `json:"reasoning"`      // 推理过程
    Duration    float64     `json:"duration"`       // 执行时间
    Timestamp   string      `json:"timestamp"`      // 时间戳
}

// FinalResult represents the final result of a question
type FinalResult struct {
    QuestionID    string      `json:"question_id"`
    Answer        string      `json:"answer"`
    Explanation   string      `json:"explanation"`
    Confidence    float64     `json:"confidence"`
    Steps         []StepResult `json:"steps"`
    TotalDuration float64     `json:"total_duration"`
    IsCorrect     bool        `json:"is_correct"`
}
```

---

## 三、Eino ADK Agent 设计

### 3.1 使用Eino ADK PlanExecuteAgent

```go
package agent

import (
    "context"
    "hle-agent/internal/models"
    "hle-agent/internal/tools"
    
    "github.com/cloudwego/eino/adk/prebuilt/planexecute"
    "github.com/cloudwego/eino/llm"
)

// HLEAgent 基于Eino ADK的HLE答题智能体
type HLEAgent struct {
    agent        *planexecute.Agent
    planner      *Planner
    executor     *Executor
    replanner    *Replanner
    llmClient    llm.ChatModel
    toolRegistry *ToolRegistry
}

// NewHLEAgent 创建HLE智能体
func NewHLEAgent(llmClient llm.ChatModel) (*HLEAgent, error) {
    // 创建工具注册表
    toolRegistry := NewToolRegistry()
    
    // 创建自定义Planner
    planner := NewPlanner(llmClient)
    
    // 创建自定义Executor
    executor := NewExecutor(toolRegistry)
    
    // 创建自定义Replanner
    replanner := NewReplanner(llmClient)
    
    // 使用Eino ADK创建PlanExecuteAgent
    agent, err := planexecute.NewAgent(
        planexecute.AgentConfig{
            Name:        "HLE-Agent",
            Description: "HLE考试答题智能体",
        },
        planner,    // Planner实现
        executor,   // Executor实现
        replanner,  // Replanner实现
    )
    
    if err != nil {
        return nil, err
    }
    
    return &HLEAgent{
        agent:        agent,
        planner:      planner,
        executor:     executor,
        replanner:    replanner,
        llmClient:    llmClient,
        toolRegistry: toolRegistry,
    }, nil
}

// Process 处理一道题目
func (a *HLEAgent) Process(ctx context.Context, question *models.Question) (*models.FinalResult, error) {
    // 调用Eino ADK Agent执行
    result, err := a.agent.Run(ctx, question.Content)
    if err != nil {
        return nil, err
    }
    
    // 转换为HLE结果格式
    return a.convertResult(question, result)
}

// ProcessBatch 批量处理题目
func (a *HLEAgent) ProcessBatch(ctx context.Context, questions []*models.Question) ([]*models.FinalResult, error) {
    results := make([]*models.FinalResult, 0, len(questions))
    
    for _, question := range questions {
        result, err := a.Process(ctx, question)
        if err != nil {
            // 记录错误但继续处理
            results = append(results, &models.FinalResult{
                QuestionID: question.ID,
                IsCorrect:  false,
            })
            continue
        }
        results = append(results, result)
    }
    
    return results, nil
}

// convertResult 转换Eino ADK结果为HLE格式
func (a *HLEAgent) convertResult(question *models.Question, result *planexecute.RunResult) (*models.FinalResult, error) {
    // 实现结果转换逻辑
    // ...
    return &models.FinalResult{
        QuestionID: question.ID,
        Answer:     result.Output,
    }, nil
}
```

### 3.2 Planner实现

```go
package agent

import (
    "context"
    "encoding/json"
    "hle-agent/internal/models"
    "hle-agent/pkg/prompts"
    
    "github.com/cloudwego/eino/llm"
    "github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// Planner 理解题目并生成解题计划
type Planner struct {
    llmClient llm.ChatModel
    promptTpl *prompts.PromptTemplate
}

// NewPlanner 创建Planner
func NewPlanner(llmClient llm.ChatModel) *Planner {
    return &Planner{
        llmClient: llmClient,
        promptTpl: prompts.NewPromptTemplate(),
    }
}

// Plan 生成解题计划
func (p *Planner) Plan(ctx context.Context, question string) (*models.Plan, error) {
    // 构建Prompt
    prompt := p.promptTpl.Render("planner", map[string]string{
        "question": question,
    })
    
    // 调用LLM
    response, err := p.llmClient.Generate(ctx, []llm.Message{
        {Role: "user", Content: prompt},
    })
    if err != nil {
        return nil, err
    }
    
    // 解析为Plan结构
    var plan models.Plan
    if err := json.Unmarshal([]byte(response), &plan); err != nil {
        // 如果JSON解析失败，尝试从文本中提取
        plan = p.parseFromText(response, question)
    }
    
    // 设置默认参数
    for i := range plan.Steps {
        plan.Steps[i].ID = i + 1
        if plan.Steps[i].RetryOnFail {
            plan.Steps[i].RetryOnFail = true
        }
    }
    plan.TotalSteps = len(plan.Steps)
    plan.CurrentStep = 1
    
    return &plan, nil
}

// parseFromText 从文本响应中解析Plan
func (p *Planner) parseFromText(response, question string) models.Plan {
    // 默认实现：简单解析
    return models.Plan{
        Steps: []models.Step{
            {ID: 1, Description: "理解题目", Action: "llm"},
            {ID: 2, Description: "分析解题策略", Action: "llm"},
            {ID: 3, Description: "执行计算", Action: "tool", ToolName: "python_executor"},
            {ID: 4, Description: "验证答案", Action: "llm"},
            {ID: 5, Description: "生成最终答案", Action: "llm"},
        },
        TotalSteps: 5,
    }
}

// 实现Eino ADK Planner接口
var _ planexecute.Planner = (*Planner)(nil)
```

### 3.3 Executor实现

```go
package agent

import (
    "context"
    "hle-agent/internal/models"
    
    "github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// Executor 执行解题步骤
type Executor struct {
    toolRegistry *ToolRegistry
}

// NewExecutor 创建Executor
func NewExecutor(registry *ToolRegistry) *Executor {
    return &Executor{
        toolRegistry: registry,
    }
}

// Execute 执行单个步骤
func (e *Executor) Execute(ctx context.Context, step *models.Step, input map[string]interface{}) (*models.StepResult, error) {
    startTime := time.Now()
    
    result := &models.StepResult{
        StepID:     step.ID,
        Success:    false,
        Timestamp:  time.Now().Format(time.RFC3339),
    }
    
    // 根据Action类型执行
    switch step.Action {
    case "llm":
        // LLM推理
        output, err := e.executeLLM(ctx, step, input)
        if err != nil {
            result.Error = err.Error()
            return result, nil
        }
        result.Output = output
        result.Success = true
        
    case "tool":
        // 工具调用
        output, err := e.executeTool(ctx, step, input)
        if err != nil {
            result.Error = err.Error()
            // 检查是否需要重试
            if step.RetryOnFail {
                // 重试逻辑
                output, err = e.retryTool(ctx, step, input)
                if err != nil {
                    result.Error = err.Error()
                    return result, nil
                }
            } else {
                return result, nil
            }
        }
        result.Output = output
        result.Success = true
        
    default:
        result.Output = "Unknown action type: " + step.Action
        result.Success = true
    }
    
    result.Duration = time.Since(startTime).Seconds()
    result.Confidence = e.calculateConfidence(result)
    
    return result, nil
}

// executeLLM 执行LLM推理
func (e *Executor) executeLLM(ctx context.Context, step *models.Step, input map[string]interface{}) (interface{}, error) {
    // 实现LLM调用逻辑
    // ...
    return nil, nil
}

// executeTool 执行工具调用
func (e *Executor) executeTool(ctx context.Context, step *models.Step, input map[string]interface{}) (interface{}, error) {
    tool, err := e.toolRegistry.GetTool(step.ToolName)
    if err != nil {
        return nil, err
    }
    
    params := step.Parameters
    if params == nil {
        params = make(map[string]interface{})
    }
    
    // 从input中提取参数
    if inputData, ok := input["input_data"].(map[string]interface{}); ok {
        for k, v := range inputData {
            params[k] = v
        }
    }
    
    return tool.Execute(ctx, params)
}

// retryTool 重试工具调用
func (e *Executor) retryTool(ctx context.Context, step *models.Step, input map[string]interface{}) (interface{}, error) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        output, err := e.executeTool(ctx, step, input)
        if err == nil {
            return output, nil
        }
        time.Sleep(time.Duration(i+1) * time.Second) // 指数退避
    }
    return nil, &ToolExecutionError{Tool: step.ToolName}
}

// calculateConfidence 计算置信度
func (e *Executor) calculateConfidence(result *models.StepResult) float64 {
    if !result.Success {
        return 0.0
    }
    
    // 基于执行时间和结果计算置信度
    if result.Duration < 5 {
        return 0.9
    } else if result.Duration < 10 {
        return 0.8
    }
    return 0.7
}

// 实现Eino ADK Executor接口
var _ planexecute.Executor = (*Executor)(nil)
```

### 3.4 Replanner实现

```go
package agent

import (
    "context"
    "encoding/json"
    "hle-agent/internal/models"
    "hle-agent/pkg/prompts"
    
    "github.com/cloudwego/eino/llm"
    "github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

// Replanner 评估结果并决定是否需要重规划
type Replanner struct {
    llmClient llm.ChatModel
    promptTpl *prompts.PromptTemplate
}

// NewReplanner 创建Replanner
func NewReplanner(llmClient llm.ChatModel) *Replanner {
    return &Replanner{
        llmClient: llmClient,
        promptTpl: prompts.NewPromptTemplate(),
    }
}

// Replan 评估并决定是否需要重规划
func (r *Replanner) Replan(ctx context.Context, plan *models.Plan, stepResults []*models.StepResult, question string) (*models.Plan, error) {
    // 分析执行结果
    successCount := 0
    failCount := 0
    lastError := ""
    
    for _, result := range stepResults {
        if result.Success {
            successCount++
        } else {
            failCount++
            lastError = result.Error
        }
    }
    
    // 检查是否完成
    if successCount == plan.TotalSteps {
        return nil, planexecute.NewFinalResult("completed")
    }
    
    // 检查是否需要重规划
    if failCount > plan.TotalSteps/2 {
        // 超过一半步骤失败，重新规划
        newPlan, err := r.generateNewPlan(ctx, question, stepResults)
        if err != nil {
            return nil, err
        }
        return newPlan, nil
    }
    
    // 检查最后一步是否失败
    if len(stepResults) > 0 {
        lastResult := stepResults[len(stepResults)-1]
        if !lastResult.Success && lastResult.Error != "" {
            // 尝试修复当前步骤
            fixedPlan := r.fixCurrentStep(plan, stepResults)
            return fixedPlan, nil
        }
    }
    
    // 继续执行当前计划
    return plan, nil
}

// generateNewPlan 生成新的计划
func (r *Replanner) generateNewPlan(ctx context.Context, question string, stepResults []*models.StepResult) (*models.Plan, error) {
    // 构建Prompt
    prompt := r.promptTpl.Render("replanner", map[string]string{
        "question":    question,
        "old_results": r.formatResults(stepResults),
    })
    
    // 调用LLM
    response, err := r.llmClient.Generate(ctx, []llm.Message{
        {Role: "user", Content: prompt},
    })
    if err != nil {
        return nil, err
    }
    
    // 解析新计划
    var newPlan models.Plan
    if err := json.Unmarshal([]byte(response), &newPlan); err != nil {
        // 解析失败，返回原计划
        return nil, planexecute.NewFinalResult("failed")
    }
    
    return &newPlan, nil
}

// fixCurrentStep 修复当前步骤
func (r *Replanner) fixCurrentStep(plan *models.Plan, stepResults []*models.StepResult) *models.Plan {
    // 复制原计划
    newPlan := *plan
    newPlan.Steps = make([]models.Step, len(plan.Steps))
    copy(newPlan.Steps, plan.Steps)
    
    // 修改当前步骤
    currentIdx := len(stepResults)
    if currentIdx < len(newPlan.Steps) {
        newPlan.Steps[currentIdx].RetryOnFail = true
        newPlan.Steps[currentIdx].Description += " (重试)"
    }
    
    return &newPlan
}

// formatResults 格式化结果用于Prompt
func (r *Replanner) formatResults(results []*models.StepResult) string {
    var sb strings.Builder
    for i, result := range results {
        sb.WriteString(fmt.Sprintf("步骤%d: %s\n", i+1, result.Description))
        sb.WriteString(fmt.Sprintf("  成功: %v\n", result.Success))
        if result.Error != "" {
            sb.WriteString(fmt.Sprintf("  错误: %s\n", result.Error))
        }
    }
    return sb.String()
}

// 实现Eino ADK Replanner接口
var _ planexecute.Replanner = (*Replanner)(nil)
```

---

## 四、工具注册与调用

### 4.1 工具注册表

```go
package agent

import (
    "context"
    "hle-agent/internal/models"
    
    "github.com/cloudwego/eino/tool"
)

// ToolRegistry 工具注册表
type ToolRegistry struct {
    tools map[string]Tool
}

// Tool 工具接口
type Tool interface {
    Name() string
    Description() string
    Parameters() *tool.Schema
    Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
    registry := &ToolRegistry{
        tools: make(map[string]Tool),
    }
    
    // 注册工具
    registry.Register(NewPythonTool())
    registry.Register(NewSageMathTool())
    registry.Register(NewKnowledgeRetrieverTool())
    
    return registry
}

// Register 注册工具
func (r *ToolRegistry) Register(t Tool) {
    r.tools[t.Name()] = t
}

// GetTool 获取工具
func (r *ToolRegistry) GetTool(name string) (Tool, error) {
    if t, ok := r.tools[name]; ok {
        return t, nil
    }
    return nil, &ToolNotFoundError{Name: name}
}

// GetAllTools 获取所有工具
func (r *ToolRegistry) GetAllTools() []Tool {
    tools := make([]Tool, 0, len(r.tools))
    for _, t := range r.tools {
        tools = append(tools, t)
    }
    return tools
}

// ToolNotFoundError 工具未找到错误
type ToolNotFoundError struct {
    Name string
}

func (e *ToolNotFoundError) Error() string {
    return "tool not found: " + e.Name
}
```

### 4.2 Python执行器工具

```go
package agent

import (
    "context"
    "encoding/json"
    "hle-agent/internal/models"
    
    "github.com/cloudwego/eino/tool"
)

// PythonTool Python执行器工具
type PythonTool struct {
    endpoint string
}

// NewPythonTool 创建Python工具
func NewPythonTool() *PythonTool {
    return &PythonTool{
        endpoint: "http://localhost:5000/python", // Python MCP服务地址
    }
}

// Name 返回工具名称
func (t *PythonTool) Name() string {
    return "python_executor"
}

// Description 返回工具描述
func (t *PythonTool) Description() string {
    return `执行Python代码，支持数学计算、数据处理、算法实现等。
    输入：Python代码字符串、超时时间
    输出：执行结果`
}

// Parameters 返回参数定义
func (t *PythonTool) Parameters() *tool.Schema {
    return &tool.Schema{
        Type: "object",
        Properties: map[string]*tool.Property{
            "code": {
                Type:        "string",
                Description: "要执行的Python代码",
            },
            "timeout": {
                Type:        "integer",
                Description: "超时时间(秒)",
                Default:     30,
            },
            "input_data": {
                Type:        "object",
                Description: "输入数据",
            },
        },
        Required: []string{"code"},
    }
}

// Execute 执行工具调用
func (t *PythonTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // 调用Python MCP服务
    reqBody, _ := json.Marshal(params)
    
    // HTTP调用Python服务
    // ...
    
    // 解析结果
    var result models.PythonResult
    json.Unmarshal([]byte(response), &result)
    
    return result, nil
}
```

---

## 五、Prompt模板管理

### 5.1 Planner Prompt

```go
package prompts

const (
    plannerTemplate = `你是一个专业的HLE考试答题助手。请分析以下题目并制定解题计划。

题目：
{question}

请制定一个详细的解题计划，包括：
1. 理解题目要求
2. 识别关键信息
3. 选择解题策略
4. 执行计算或推理
5. 验证答案

请按以下JSON格式输出计划：
{
    "id": "plan_001",
    "question_id": "从题目中提取或生成唯一ID",
    "steps": [
        {
            "id": 1,
            "description": "步骤描述",
            "action": "llm 或 tool",
            "tool_name": "工具名称（如需要）",
            "parameters": {"参数": "值"},
            "is_key_point": true/false,
            "retry_on_fail": true/false
        }
    ]
}

注意：
- action为"llm"表示需要LLM推理
- action为"tool"表示需要调用工具
- is_key_point标记关键步骤
- retry_on_fail标记失败时是否重试`
)
```

### 5.2 Replanner Prompt

```go
const (
    replannerTemplate = `请评估以下解题步骤的执行结果，决定是否需要重规划。

题目：
{question}

执行结果：
{old_results}

请评估：
1. 所有步骤是否都成功了？
2. 如果有失败的步骤，是否需要重规划整个方案？
3. 是否只需要重试当前失败的步骤？

请按以下JSON格式输出：
{
    "need_replan": true/false,
    "reason": "重规划或不重规划的理由",
    "new_steps": [
        // 如果需要重规划，提供新的步骤列表
    ]
}

如果不需要重规划，返回：
{"need_replan": false, "reason": "理由"}
`
)
```

---

## 六、总结

本文档基于Eino ADK的Plan-Execute Agent模式重新设计了HLE考试答题系统：

1. **利用Eino ADK预构建框架**：
   - 使用PlanExecuteAgent作为核心框架
   - 自定义Planner、Executor、Replanner
   - 减少大量重复代码

2. **核心组件**：
   - Planner：题目理解 + 策略规划
   - Executor：步骤执行 + 工具调用
   - Replanner：结果评估 + 动态重规划

3. **工具集成**：
   - Python执行器（Python MCP）
   - SageMath执行器（Python MCP）
   - 知识检索器

4. **优势**：
   - 遵循框架最佳实践
   - 代码更简洁
   - 维护性更好
   - 扩展性更强

---

**文档编写完成，等待评审确认。**

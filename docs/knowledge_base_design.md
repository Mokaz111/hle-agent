# 知识库模块设计文档

## 一、需求分析

### 1.1 核心需求

**问题：** LLM 模型能力有限，需要借助其他大模型（如 GPT）的解题思维链来提升准确率。

**解决方案：** 
- 手动收集 GPT 等大模型在 HLE 测试集上的解题思维链
- 存储到知识库中
- 在调用 LLM 时，检索匹配的思维链并作为上下文一起发送
- 通过 Few-Shot Learning 的方式提升小模型的推理能力

### 1.2 设计目标

1. **知识存储**：持久化存储高质量的解题思维链
2. **智能检索**：基于问题相似度检索最相关的思维链
3. **上下文注入**：在 Planner 和 Executor 调用 LLM 时注入思维链上下文
4. **性能优化**：快速检索，避免增加过多延迟

---

## 二、架构设计

### 2.1 模块定位

**建议：作为 LongTermMemory 的子模块，但也可以独立实现**

**理由：**
- ✅ **独立模块**：思维链库是专门的知识库，与用户历史对话不同
- ✅ **可复用**：可以被 Planner、Executor、Replanner 等多个组件使用
- ✅ **易扩展**：未来可以添加其他类型的知识库（如公式库、模式库等）

**推荐方案：独立的知识库模块 `pkg/knowledgebase/`**

```
pkg/
├── knowledgebase/          # 新增：知识库模块
│   ├── knowledgebase.go    # 知识库接口和实现
│   ├── chain_store.go      # 思维链存储
│   ├── chain_retriever.go  # 思维链检索
│   └── chain_formatter.go  # 思维链格式化
├── memory/
│   ├── short_term.go       # 短期记忆（会话状态）
│   └── long_term.go        # 长期记忆（用户历史，待实现）
└── retriever/
    └── retriever.go        # 检索器（可集成知识库）
```

### 2.2 数据模型

```go
// ChainOfThought 表示一个完整的解题思维链
type ChainOfThought struct {
    ID          string                 `json:"id"`
    Question    string                 `json:"question"`      // 原始问题
    QuestionID  string                 `json:"question_id"`   // HLE 测试集问题ID
    Domain      string                 `json:"domain"`        // 领域（数学、编程等）
    Complexity  string                 `json:"complexity"`    // 难度（low/medium/high）
    Keywords    []string               `json:"keywords"`     // 关键词
    Model       string                 `json:"model"`        // 来源模型（gpt-4, claude等）
    
    // 思维链内容
    Reasoning   string                 `json:"reasoning"`    // 完整推理过程
    Steps       []ReasoningStep        `json:"steps"`        // 结构化步骤
    Answer      string                 `json:"answer"`        // 最终答案
    Confidence  float64                `json:"confidence"`   // 置信度
    
    // 元数据
    Embedding   []float32              `json:"embedding"`     // 向量嵌入（用于语义检索）
    Metadata    map[string]interface{} `json:"metadata"`     // 扩展元数据
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

// ReasoningStep 表示思维链中的一个步骤
type ReasoningStep struct {
    StepID      int    `json:"step_id"`
    Description string `json:"description"`  // 步骤描述
    Action      string `json:"action"`        // 操作类型（think, calculate, reason等）
    Content     string `json:"content"`       // 步骤内容
    Result      string `json:"result"`        // 步骤结果
}

// KnowledgeBase 知识库接口
type KnowledgeBase interface {
    // 存储思维链
    AddChain(chain *ChainOfThought) error
    
    // 检索相似思维链
    RetrieveSimilar(question string, domain string, maxResults int) ([]*ChainOfThought, error)
    
    // 格式化思维链为 LLM 上下文
    FormatChainsForLLM(chains []*ChainOfThought) string
    
    // 批量导入
    ImportChains(chains []*ChainOfThought) error
}
```

### 2.3 存储方案

**方案选择：**

| 方案 | 优点 | 缺点 | 推荐度 |
|------|------|------|--------|
| **向量数据库（Milvus/QDrant）** | 语义检索准确，支持大规模 | 需要额外服务，复杂度高 | ⭐⭐⭐⭐ |
| **SQLite + 向量扩展** | 简单，无需额外服务 | 性能有限，扩展性差 | ⭐⭐⭐ |
| **JSON 文件 + 简单匹配** | 最简单，快速实现 | 检索能力弱，性能差 | ⭐⭐ |
| **混合方案（SQLite + 向量库）** | 平衡性能和复杂度 | 实现复杂 | ⭐⭐⭐⭐ |

**推荐：分阶段实现**
- **Phase 1（快速实现）**：SQLite + 关键词/领域匹配
- **Phase 2（优化）**：集成向量数据库（Milvus/QDrant）进行语义检索

---

## 三、集成方案

### 3.1 在 Planner 中集成

**位置：** `internal/agent/planner.go:Plan()`

```go
func (p *Planner) Plan(ctx context.Context, question string) (*HLEPlan, error) {
    // ... 现有代码 ...
    
    // 从知识库检索相似思维链
    var chainContext string
    if p.knowledgeBase != nil {
        similarChains, err := p.knowledgeBase.RetrieveSimilar(
            question,
            analysis.Domain,
            3, // 最多3条
        )
        if err == nil && len(similarChains) > 0 {
            chainContext = p.knowledgeBase.FormatChainsForLLM(similarChains)
            logger.Debug("检索到相似思维链",
                zap.Int("count", len(similarChains)))
        }
    }
    
    // 构建 Prompt，包含思维链上下文
    var promptBuilder strings.Builder
    promptBuilder.WriteString(template)
    
    if chainContext != "" {
        promptBuilder.WriteString("\n\n=== 参考解题思路 ===\n")
        promptBuilder.WriteString(chainContext)
        promptBuilder.WriteString("\n\n请参考以上解题思路，为当前问题制定解题计划。\n")
    }
    
    promptBuilder.WriteString("\n\n当前问题:\n")
    promptBuilder.WriteString(question)
    
    // ... 调用 LLM ...
}
```

### 3.2 在 Executor 中集成

**位置：** `internal/agent/executor.go:executeWithLLM()`

```go
func (e *Executor) executeWithLLM(ctx context.Context, step *HLEStep) (string, error) {
    // 如果步骤描述包含问题信息，可以检索相关思维链
    var chainContext string
    if e.knowledgeBase != nil && step.Description != "" {
        // 从步骤描述中提取问题关键词
        similarChains, err := e.knowledgeBase.RetrieveSimilar(
            step.Description,
            "", // 可以从上下文获取领域
            2,  // 最多2条
        )
        if err == nil && len(similarChains) > 0 {
            chainContext = e.knowledgeBase.FormatChainsForLLM(similarChains)
        }
    }
    
    // 构建 Prompt
    prompt := fmt.Sprintf("请执行以下步骤：\n%s", step.Description)
    
    if chainContext != "" {
        prompt = fmt.Sprintf("=== 参考思路 ===\n%s\n\n%s", chainContext, prompt)
    }
    
    // ... 调用 LLM ...
}
```

### 3.3 格式化思维链

```go
// FormatChainsForLLM 将思维链格式化为 LLM 可用的上下文
func (kb *KnowledgeBase) FormatChainsForLLM(chains []*ChainOfThought) string {
    var builder strings.Builder
    
    for i, chain := range chains {
        builder.WriteString(fmt.Sprintf("========== 参考示例 %d ==========\n", i+1))
        builder.WriteString(fmt.Sprintf("问题：%s\n\n", chain.Question))
        
        // 格式化推理过程
        if chain.Reasoning != "" {
            builder.WriteString("推理过程：\n")
            builder.WriteString(chain.Reasoning)
            builder.WriteString("\n\n")
        }
        
        // 格式化结构化步骤
        if len(chain.Steps) > 0 {
            builder.WriteString("解题步骤：\n")
            for j, step := range chain.Steps {
                builder.WriteString(fmt.Sprintf("步骤 %d: %s\n", j+1, step.Description))
                if step.Content != "" {
                    builder.WriteString(fmt.Sprintf("  内容: %s\n", step.Content))
                }
                if step.Result != "" {
                    builder.WriteString(fmt.Sprintf("  结果: %s\n", step.Result))
                }
            }
            builder.WriteString("\n")
        }
        
        builder.WriteString(fmt.Sprintf("最终答案：%s\n", chain.Answer))
        builder.WriteString("\n")
    }
    
    return builder.String()
}
```

---

## 四、实现细节

### 4.1 检索策略

**Phase 1：简单匹配（快速实现）**

```go
func (kb *KnowledgeBase) RetrieveSimilar(question string, domain string, maxResults int) ([]*ChainOfThought, error) {
    // 1. 领域匹配
    // 2. 关键词匹配（Jaccard相似度）
    // 3. 文本相似度（简单的字符串匹配或编辑距离）
    // 4. 按相似度排序，返回 Top K
}
```

**Phase 2：语义检索（优化）**

```go
func (kb *KnowledgeBase) RetrieveSimilar(question string, domain string, maxResults int) ([]*ChainOfThought, error) {
    // 1. 将问题向量化
    questionEmbedding := kb.embedder.Embed(question)
    
    // 2. 向量数据库检索（Milvus/QDrant）
    similarVectors := kb.vectorStore.Search(questionEmbedding, maxResults)
    
    // 3. 过滤和排序
    // 4. 返回结果
}
```

### 4.2 数据导入

**支持格式：**

1. **JSON 格式**
```json
{
  "question_id": "hle_001",
  "question": "计算 1+1",
  "domain": "math",
  "complexity": "low",
  "model": "gpt-4",
  "reasoning": "这是一个简单的加法问题...",
  "steps": [
    {
      "step_id": 1,
      "description": "识别问题类型",
      "content": "这是一个基础算术问题",
      "result": "确认是加法运算"
    }
  ],
  "answer": "2",
  "confidence": 0.99
}
```

2. **CSV 格式**（批量导入）

3. **命令行工具导入**
```bash
./hle-agent import-chains --file chains.json --model gpt-4
```

### 4.3 配置

```yaml
knowledge_base:
  enabled: true
  storage:
    type: "sqlite"  # sqlite, milvus, qdrant
    path: "./data/knowledge_base.db"
  
  retrieval:
    max_results: 3
    min_similarity: 0.6
    use_semantic_search: false  # Phase 1 为 false，Phase 2 为 true
  
  embedding:
    provider: "openai"  # openai, local
    model: "text-embedding-ada-002"
```

---

## 五、设计合理性分析

### 5.1 ✅ 优点

1. **提升准确率**：通过 Few-Shot Learning 提升小模型能力
2. **可扩展性**：可以持续添加高质量的思维链
3. **灵活性**：可以针对不同领域使用不同的思维链
4. **成本效益**：利用已有的大模型输出，无需额外训练

### 5.2 ⚠️ 注意事项

1. **思维链质量**：需要确保存储的思维链是高质量的
2. **匹配准确性**：检索到的思维链必须与当前问题相关
3. **Token 限制**：思维链可能很长，需要注意 LLM 的 token 限制
4. **过拟合风险**：如果思维链过于具体，可能导致模型过度依赖

### 5.3 🔧 优化建议

1. **思维链压缩**：对于过长的思维链，可以提取关键步骤
2. **动态选择**：根据问题复杂度选择不同数量的思维链
3. **质量评分**：为思维链添加质量评分，优先使用高质量链
4. **A/B 测试**：对比使用和不使用思维链的效果

---

## 六、实施路线图

### Phase 1：基础实现（1-2周）

- [ ] 设计数据模型
- [ ] 实现 SQLite 存储
- [ ] 实现简单检索（关键词+领域匹配）
- [ ] 实现格式化功能
- [ ] 集成到 Planner
- [ ] 实现数据导入工具

### Phase 2：优化（1-2周）

- [ ] 集成向量数据库
- [ ] 实现语义检索
- [ ] 添加思维链压缩
- [ ] 性能优化
- [ ] 添加监控指标

### Phase 3：高级功能（可选）

- [ ] 自动质量评估
- [ ] 思维链自动提取（从执行结果）
- [ ] 多模型思维链融合
- [ ] 思维链版本管理

---

## 七、总结

### 7.1 推荐方案

**独立的知识库模块 `pkg/knowledgebase/`**

**理由：**
1. ✅ 职责清晰：专门存储和检索思维链
2. ✅ 易于维护：与 LongTermMemory 解耦
3. ✅ 可扩展：未来可以添加其他类型的知识库
4. ✅ 可复用：可以被多个组件使用

### 7.2 设计合理性

**✅ 设计合理，建议采用**

这个设计符合 RAG（Retrieval-Augmented Generation）的最佳实践，可以有效提升小模型的推理能力。关键是要：
1. 确保思维链质量
2. 实现准确的检索匹配
3. 控制上下文长度
4. 持续优化和迭代


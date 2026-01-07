# 知识库使用指南

## 一、快速开始

### 1.1 启用知识库

在 `config.yaml` 中配置：

```yaml
knowledge_base:
  enabled: true
  storage_path: "./data/knowledge_base.db"
  max_results: 3
  min_similarity: 0.5
  weight_keywords: 0.4
  weight_domain: 0.4
  weight_complexity: 0.2
```

### 1.2 导入思维链

准备 JSON 格式的思维链数据，然后导入：

```bash
# 导入 JSON 数组格式
./import-chains -input chains.json -db ./data/knowledge_base.db

# 导入 JSONL 格式（每行一个 JSON）
./import-chains -input chains.jsonl -db ./data/knowledge_base.db
```

---

## 二、数据格式

### 2.1 JSON 格式示例

```json
[
  {
    "id": "chain_001",
    "question": "计算 1+1 等于多少？",
    "question_id": "hle_001",
    "domain": "math",
    "complexity": "low",
    "keywords": ["计算", "加法", "基础"],
    "model": "gpt-4",
    "reasoning": "这是一个简单的加法问题。1+1 是最基础的算术运算，答案是 2。",
    "steps": [
      {
        "step_id": 1,
        "description": "识别问题类型",
        "action": "think",
        "content": "这是一个基础算术问题",
        "result": "确认是加法运算"
      },
      {
        "step_id": 2,
        "description": "执行计算",
        "action": "calculate",
        "content": "1 + 1",
        "result": "2"
      }
    ],
    "answer": "2",
    "confidence": 0.99
  }
]
```

### 2.2 JSONL 格式示例

```jsonl
{"id":"chain_001","question":"计算 1+1","domain":"math","reasoning":"...","answer":"2"}
{"id":"chain_002","question":"求解方程 x+1=2","domain":"math","reasoning":"...","answer":"x=1"}
```

---

## 三、使用场景

### 3.1 自动检索

知识库会在 Planner 生成计划时自动检索相似思维链，并作为上下文发送给 LLM。

### 3.2 手动查询（未来功能）

```go
// 检索相似思维链
chains, err := kb.RetrieveSimilar(ctx, question, "math", 3)

// 格式化供 LLM 使用
context := kb.FormatChainsForLLM(chains)
```

---

## 四、最佳实践

### 4.1 思维链质量

- ✅ 使用 GPT-4 等高质量模型的输出
- ✅ 确保推理过程完整清晰
- ✅ 包含结构化的步骤信息
- ✅ 标注领域和复杂度

### 4.2 数据组织

- 按领域分类存储
- 为每个思维链添加关键词
- 标注来源模型和置信度
- 定期更新和维护

### 4.3 性能优化

- 控制思维链数量（建议 3-5 条）
- 设置合理的相似度阈值（0.5-0.7）
- 定期清理低质量数据

---

## 五、故障排查

### 5.1 知识库未启用

检查 `config.yaml` 中的 `knowledge_base.enabled` 是否为 `true`

### 5.2 检索不到结果

- 检查数据库中是否有数据：`SELECT COUNT(*) FROM chains`
- 降低 `min_similarity` 阈值
- 检查领域匹配是否正确

### 5.3 导入失败

- 检查 JSON 格式是否正确
- 确保数据库文件路径可写
- 查看日志获取详细错误信息


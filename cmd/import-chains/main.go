package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hle-agent/hle-agent/pkg/knowledgebase"
)

var (
	inputFile = flag.String("input", "", "输入文件路径（JSON 或 JSONL 格式）")
	dbPath    = flag.String("db", "./data/knowledge_base.db", "知识库数据库路径")
)

func main() {
	flag.Parse()

	if *inputFile == "" {
		log.Fatal("请指定输入文件: -input <file>")
	}

	// 读取输入文件
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}

	// 解析 JSON
	var chains []*knowledgebase.ChainOfThought
	
	// 尝试解析为 JSON 数组
	if err := json.Unmarshal(data, &chains); err != nil {
		// 如果不是数组，尝试解析为 JSONL（每行一个 JSON）
		lines := strings.Split(string(data), "\n")
		chains = make([]*knowledgebase.ChainOfThought, 0)
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var chain knowledgebase.ChainOfThought
			if err := json.Unmarshal([]byte(line), &chain); err != nil {
				log.Printf("警告: 跳过第 %d 行（解析失败）: %v", i+1, err)
				continue
			}
			chains = append(chains, &chain)
		}
	}

	if len(chains) == 0 {
		log.Fatal("未找到有效的思维链数据")
	}

	fmt.Printf("读取到 %d 条思维链\n", len(chains))

	// 创建知识库
	cfg := &knowledgebase.KnowledgeBaseConfig{
		StoragePath: *dbPath,
	}
	kb, err := knowledgebase.NewKnowledgeBase(cfg)
	if err != nil {
		log.Fatalf("创建知识库失败: %v", err)
	}
	defer kb.Close()

	// 导入思维链
	ctx := context.Background()
	if err := kb.ImportChains(ctx, chains); err != nil {
		log.Fatalf("导入失败: %v", err)
	}

	// 显示统计信息
	count, err := kb.GetChainCount(ctx)
	if err == nil {
		fmt.Printf("导入成功！知识库中共有 %d 条思维链\n", count)
	} else {
		fmt.Println("导入成功！")
	}
}


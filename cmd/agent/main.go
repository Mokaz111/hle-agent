package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hle-agent/hle-agent/internal/agent"
	"github.com/hle-agent/hle-agent/internal/config"
)

var (
	configFile   = flag.String("config", "config.yaml", "Path to configuration file")
	questionFile = flag.String("input", "", "Path to input question file (JSONL format)")
	outputDir    = flag.String("output", "./results", "Output directory for results")
	version      = "1.0.0"
)

func main() {
	flag.Parse()

	// Print version
	fmt.Printf("HLE Agent v%s\n", version)
	fmt.Println("==================")

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Configuration loaded from: %s\n", *configFile)
	fmt.Printf("Model: %s (%s)\n", cfg.Model.Model, cfg.Model.Provider)
	fmt.Println()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		cancel()
	}()

	// Create HLE Agent
	hleAgent, err := agent.NewHLEAgent(cfg)
	if err != nil {
		log.Fatalf("Failed to create HLE Agent: %v", err)
	}

	fmt.Println("HLE Agent initialized successfully!")
	fmt.Println()

	// Check if running in interactive mode, batch mode, or stdin mode
	if *questionFile == "stdin" {
		// Stdin mode: process questions from standard input
		runStdin(ctx, hleAgent)
	} else if *questionFile != "" {
		// Batch mode: process questions from file
		processBatch(ctx, hleAgent, *questionFile, *outputDir)
	} else {
		// Interactive mode
		runInteractive(ctx, hleAgent)
	}

	fmt.Println("\nHLE Agent shutdown complete.")
}

func runInteractive(ctx context.Context, hleAgent *agent.HLEAgent) {
	fmt.Println("Running in interactive mode...")
	fmt.Println("Enter your question (or 'quit' to exit):")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Print("\nQuestion: ")

			var question string
			fmt.Scanln(&question)

			if question == "quit" || question == "exit" {
				fmt.Println("Goodbye!")
				return
			}

			if question == "" {
				continue
			}

			// Process the question
			fmt.Println("\nProcessing...")

			result, err := hleAgent.Process(ctx, question)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			// Output result
			fmt.Printf("\nAnswer: %s\n", result.Answer)
			fmt.Printf("Confidence: %.2f%%\n", result.Confidence*100)
			fmt.Printf("Duration: %.2fs\n", result.TotalDuration)
		}
	}
}

func processBatch(ctx context.Context, hleAgent *agent.HLEAgent, inputFile, outputDir string) {
	fmt.Printf("Processing questions from: %s\n", inputFile)
	fmt.Printf("Output directory: %s\n", outputDir)
	fmt.Println()

	// 读取输入文件
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer file.Close()

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// 读取并解析问题
	var questions []map[string]interface{}
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "{}" {
			continue
		}

		var question map[string]interface{}
		if err := json.Unmarshal([]byte(line), &question); err != nil {
			fmt.Printf("Warning: Failed to parse line %d: %v\n", lineNum, err)
			continue
		}
		questions = append(questions, question)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}

	fmt.Printf("Loaded %d questions\n\n", len(questions))

	// 创建结果文件
	timestamp := time.Now().Format("20060102_150405")
	resultFile := fmt.Sprintf("%s/results_%s.jsonl", outputDir, timestamp)
	output, err := os.Create(resultFile)
	if err != nil {
		log.Fatalf("Failed to create result file: %v", err)
	}
	defer output.Close()

	// 统计变量
	correct := 0
	errors := 0
	startTime := time.Now()

	// 处理每个问题
	for i, q := range questions {
		questionID, _ := q["id"].(string)
		questionText, _ := q["question"].(string)
		expectedAnswer, _ := q["answer"].(string)

		// 截取问题文本用于显示
		displayQuestion := questionText
		if len(displayQuestion) > 60 {
			displayQuestion = displayQuestion[:60] + "..."
		}

		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(questions), displayQuestion)

		// 处理问题
		result, err := hleAgent.Process(ctx, questionText)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			errors++

			// 记录错误结果
			resultRecord := map[string]interface{}{
				"id":                      questionID,
				"status":                  "error",
				"error":                   err.Error(),
				"expected":                expectedAnswer,
				"processing_time_seconds": time.Since(startTime).Seconds(),
			}
			recordJSON, err := json.Marshal(resultRecord)
			if err == nil {
				output.Write(recordJSON)
				output.WriteString("\n")
			}
		} else {
			fmt.Printf("  ✅ Answer: %.80s...\n", result.Answer)
			fmt.Printf("     Confidence: %.2f%%\n", result.Confidence*100)
			fmt.Printf("     Duration: %.2fs\n", result.TotalDuration)

			// 简单匹配检查
			resultAnswer := strings.ToLower(strings.TrimSpace(result.Answer))
			expectedLower := strings.ToLower(strings.TrimSpace(expectedAnswer))

			isMatch := strings.Contains(resultAnswer, expectedLower) ||
				strings.Contains(expectedLower, resultAnswer)

			if isMatch {
				fmt.Printf("     ✅ Match: Correct!\n")
				correct++
			} else {
				fmt.Printf("     ❌ No match with expected\n")
				fmt.Printf("     Expected: %.80s...\n", expectedAnswer)
			}

			// 记录成功结果
			resultRecord := map[string]interface{}{
				"id":                      questionID,
				"status":                  "success",
				"answer":                  result.Answer,
				"confidence":              result.Confidence,
				"duration":                result.TotalDuration,
				"expected":                expectedAnswer,
				"is_correct":              isMatch,
				"steps_count":             len(result.Steps),
				"processing_time_seconds": time.Since(startTime).Seconds(),
			}
			recordJSON, err := json.Marshal(resultRecord)
			if err == nil {
				output.Write(recordJSON)
				output.WriteString("\n")
			}
		}

		fmt.Println()
	}

	// 打印统计信息
	totalTime := time.Since(startTime)
	separator := strings.Repeat("=", 60)
	fmt.Println(separator)
	fmt.Println("Batch Processing Complete!")
	fmt.Println(separator)
	fmt.Printf("Total questions: %d\n", len(questions))
	fmt.Printf("Correct answers: %d\n", correct)
	fmt.Printf("Errors: %d\n", errors)
	if len(questions)-errors > 0 {
		fmt.Printf("Accuracy: %.1f%%\n", float64(correct)/float64(len(questions)-errors)*100)
	}
	fmt.Printf("Total time: %.2fs\n", totalTime.Seconds())
	fmt.Printf("Average time per question: %.2fs\n", totalTime.Seconds()/float64(len(questions)))
	fmt.Printf("Results saved to: %s\n", resultFile)
	fmt.Println(separator)
}

func runStdin(ctx context.Context, hleAgent *agent.HLEAgent) {
	fmt.Println("Running in stdin mode...")
	fmt.Println("Enter questions (one per line, Ctrl+D or Ctrl+Z to finish):")

	scanner := bufio.NewScanner(os.Stdin)
	lineNum := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !scanner.Scan() {
				// EOF or error
				if err := scanner.Err(); err != nil {
					fmt.Printf("Error reading stdin: %v\n", err)
				}
				return
			}

			lineNum++
			question := strings.TrimSpace(scanner.Text())

			// Skip empty lines
			if question == "" {
				continue
			}

			fmt.Printf("\n[%d] Processing: %s\n", lineNum, question)

			result, err := hleAgent.Process(ctx, question)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("\nAnswer: %s\n", result.Answer)
				fmt.Printf("Confidence: %.2f%%\n", result.Confidence*100)
				fmt.Printf("Duration: %.2fs\n", result.TotalDuration)
			}
			fmt.Println(strings.Repeat("-", 50))
		}
	}
}

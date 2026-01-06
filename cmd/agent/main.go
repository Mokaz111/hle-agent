package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hle-agent/hle-agent/internal/config"
	"github.com/hle-agent/hle-agent/internal/agent"
)

var (
	configFile = flag.String("config", "config.yaml", "Path to configuration file")
	questionFile = flag.String("input", "", "Path to input question file (JSONL format)")
	outputDir = flag.String("output", "./results", "Output directory for results")
	version = "1.0.0"
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
	
	// Check if running in interactive mode or batch mode
	if *questionFile != "" {
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
	
	// TODO: Implement batch processing
	fmt.Println("\nBatch processing mode - TODO: Implement")
	
	_ = hleAgent
	_ = outputDir
}

func init() {
	// Set random seed for reproducibility
	// In production, you might want to use a fixed seed
}

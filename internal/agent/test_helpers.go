package agent

import (
	"context"

	"github.com/hle-agent/hle-agent/pkg/knowledgebase"
	"github.com/hle-agent/hle-agent/pkg/llm"
	"github.com/hle-agent/hle-agent/pkg/memory"
	"github.com/hle-agent/hle-agent/pkg/retriever"
)

// MockLLMClientForTesting is a test helper that creates a mockable LLM client
// Since llm.Client is a concrete type, we use a wrapper approach
type MockLLMClientForTesting struct {
	GenerateFunc              func(ctx context.Context, messages []llm.Message) (string, error)
	GenerateWithSystemPromptFunc func(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	realClient                *llm.Client
}

// NewTestPlanner creates a Planner for testing
func NewTestPlanner(llmClient *llm.Client, withKB bool) *Planner {
	mem := memory.NewDefaultMemory()
	mem.StartSession("test-session")
	r := retriever.NewRetriever(mem, nil)
	
	var kb knowledgebase.KnowledgeBase
	if withKB {
		kb = &MockKnowledgeBase{}
	}
	
	return NewPlanner(llmClient, r, kb)
}

// NewTestExecutor creates an Executor for testing
func NewTestExecutor(llmClient *llm.Client, cfg *ExecutorConfig) *Executor {
	registry := NewToolRegistry()
	return NewExecutor(registry, llmClient, cfg)
}

// NewTestReplanner creates a Replanner for testing
func NewTestReplanner(llmClient *llm.Client) *Replanner {
	return NewReplanner(llmClient)
}

// MockKnowledgeBase is a mock implementation for testing
type MockKnowledgeBase struct {
	chains []*knowledgebase.ChainOfThought
}

func (m *MockKnowledgeBase) AddChain(ctx context.Context, chain *knowledgebase.ChainOfThought) error {
	if m.chains == nil {
		m.chains = []*knowledgebase.ChainOfThought{}
	}
	m.chains = append(m.chains, chain)
	return nil
}

func (m *MockKnowledgeBase) RetrieveSimilar(ctx context.Context, question string, domain string, maxResults int) ([]*knowledgebase.ChainOfThought, error) {
	if m.chains == nil {
		return []*knowledgebase.ChainOfThought{}, nil
	}
	if maxResults > len(m.chains) {
		maxResults = len(m.chains)
	}
	return m.chains[:maxResults], nil
}

func (m *MockKnowledgeBase) FormatChainsForLLM(chains []*knowledgebase.ChainOfThought) string {
	if len(chains) == 0 {
		return ""
	}
	return "参考思维链：\n" + chains[0].Reasoning
}

func (m *MockKnowledgeBase) ImportChains(ctx context.Context, chains []*knowledgebase.ChainOfThought) error {
	m.chains = chains
	return nil
}

func (m *MockKnowledgeBase) GetChainCount(ctx context.Context) (int, error) {
	return len(m.chains), nil
}

func (m *MockKnowledgeBase) Close() error {
	return nil
}


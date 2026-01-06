package perception

import (
	"regexp"
	"strings"
	"sync"
)

// Perception combines Parser and Recognizer for unified perception
type Perception struct {
	parser    *Parser
	recognizer *Recognizer
	logger    interface{}
	mu        sync.RWMutex
	history   []*QuestionInfo
}

// NewPerception creates a new Perception instance
func NewPerception() *Perception {
	return &Perception{
		parser:    NewParser(),
		recognizer: NewRecognizer(),
		history:   make([]*QuestionInfo, 0),
	}
}

// Analyze performs full analysis on a question
func (p *Perception) Analyze(question string) *AnalysisResult {
	result := &AnalysisResult{
		QuestionInfo:      p.parser.ParseQuestion(question),
		DomainRecognition: p.recognizer.Recognize(question),
	}

	// Store in history
	p.mu.Lock()
	p.history = append(p.history, result.QuestionInfo)
	if len(p.history) > 100 {
		p.history = p.history[len(p.history)-100:]
	}
	p.mu.Unlock()

	return result
}

// AnalysisResult represents the full analysis result
type AnalysisResult struct {
	*QuestionInfo
	*DomainRecognition
}

// GetRecommendedTool returns the recommended tool for the question
func (p *Perception) GetRecommendedTool(question string) string {
	domain := p.recognizer.Recognize(question).PrimaryDomain
	tools := p.recognizer.GetRecommendedTools(domain)
	if len(tools) > 0 {
		return tools[0]
	}
	return "python_executor"
}

// GetHistory returns the analysis history
func (p *Perception) GetHistory() []*QuestionInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*QuestionInfo, len(p.history))
	copy(result, p.history)
	return result
}

// GetDomainStats returns statistics about analyzed domains
func (p *Perception) GetDomainStats() map[Domain]int {
	stats := make(map[Domain]int)

	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, info := range p.history {
		if info.Domain != DomainUnknown {
			stats[info.Domain]++
		}
	}

	return stats
}

// Preprocess performs preprocessing on the input
func (p *Perception) Preprocess(input string) string {
	// Clean up the input
	result := strings.TrimSpace(input)

	// Remove excessive whitespace
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	// Normalize newlines
	result = strings.ReplaceAll(result, "\r\n", "\n")

	return result
}

// ExtractCodeBlocks extracts code blocks from the input
func (p *Perception) ExtractCodeBlocks(input string) []CodeBlock {
	blocks := make([]CodeBlock, 0)

	// Simple code block detection
	// Look for ```code blocks```
	codeBlockRegex := regexp.MustCompile("```(\\w*)\\n([\\s\\S]*?)```")
	matches := codeBlockRegex.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			blocks = append(blocks, CodeBlock{
				Language: match[1],
				Code:     match[2],
			})
		}
	}

	return blocks
}

// CodeBlock represents a code block in the input
type CodeBlock struct {
	Language string
	Code     string
}

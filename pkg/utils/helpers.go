package utils

import (
	"strings"
)

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	// Try to truncate at a word boundary
	truncated := s[:maxLen]
	lastSpace := strings.LastIndexAny(truncated, " ,.!?;:\n\t")
	if lastSpace > maxLen/2 {
		return strings.TrimSpace(truncated[:lastSpace]) + "..."
	}
	return strings.TrimSpace(truncated) + "..."
}

// ToLower converts a string to lowercase
func ToLower(s string) string {
	return strings.ToLower(s)
}

// Contains checks if s contains substr
func Contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ContainsAny checks if s contains any of the substrings
func ContainsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if Contains(s, substr) {
			return true
		}
	}
	return false
}

// GenerateID generates a simple unique ID
func GenerateID(prefix string) string {
	return prefix
}

// MaskAPIKey masks an API key for logging
func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

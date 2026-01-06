package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateStringNormal(t *testing.T) {
	result := TruncateString("Hello, World!", 20)

	assert.Equal(t, "Hello, World!", result)
}

func TestTruncateStringAtBoundary(t *testing.T) {
	result := TruncateString("Hello, World!", 5)

	assert.Equal(t, "Hello...", result)
}

func TestTruncateStringWithWordBoundary(t *testing.T) {
	result := TruncateString("Hello, World! How are you?", 10)

	// Should truncate at word boundary near space
	assert.Contains(t, result, "...")
}

func TestTruncateStringEmpty(t *testing.T) {
	result := TruncateString("", 10)

	assert.Equal(t, "", result)
}

func TestTruncateStringZeroMaxLen(t *testing.T) {
	result := TruncateString("Hello", 0)

	assert.Equal(t, "", result)
}

func TestTruncateStringNegativeMaxLen(t *testing.T) {
	result := TruncateString("Hello", -1)

	assert.Equal(t, "", result)
}

func TestTruncateStringLongText(t *testing.T) {
	longText := "This is a very long text that should be truncated at some point to fit the maximum length"
	result := TruncateString(longText, 20)

	assert.LessOrEqual(t, len(result), 23) // 20 chars + "..."
	assert.Contains(t, result, "...")
}

func TestTruncateStringWithNewlines(t *testing.T) {
	result := TruncateString("Hello\nWorld\nTest", 8)

	assert.Contains(t, result, "...")
}

func TestToLowerBasic(t *testing.T) {
	result := ToLower("HELLO")

	assert.Equal(t, "hello", result)
}

func TestToLowerMixed(t *testing.T) {
	result := ToLower("HeLLo WoRLd")

	assert.Equal(t, "hello world", result)
}

func TestToLowerEmpty(t *testing.T) {
	result := ToLower("")

	assert.Equal(t, "", result)
}

func TestContainsBasic(t *testing.T) {
	assert.True(t, Contains("Hello World", "World"))
	assert.False(t, Contains("Hello World", "Python"))
}

func TestContainsAtStart(t *testing.T) {
	assert.True(t, Contains("Hello World", "Hello"))
}

func TestContainsAtEnd(t *testing.T) {
	assert.True(t, Contains("Hello World", "World"))
}

func TestContainsEmptySubstr(t *testing.T) {
	assert.True(t, Contains("Hello", ""))
}

func TestContainsSubstrLonger(t *testing.T) {
	assert.False(t, Contains("Hi", "Hello"))
}

func TestContainsAnyTrue(t *testing.T) {
	assert.True(t, ContainsAny("Hello World", "Python", "World", "Java"))
}

func TestContainsAnyFalse(t *testing.T) {
	assert.False(t, ContainsAny("Hello World", "Python", "Java", "C++"))
}

func TestContainsAnyEmpty(t *testing.T) {
	assert.False(t, ContainsAny("Hello World"))
}

func TestContainsAnyWithEmptyList(t *testing.T) {
	assert.False(t, ContainsAny("Hello World"))
}

func TestGenerateID(t *testing.T) {
	result := GenerateID("test")

	assert.Equal(t, "test", result)
}

func TestGenerateIDDifferentPrefixes(t *testing.T) {
	result1 := GenerateID("prefix1")
	result2 := GenerateID("prefix2")

	assert.Equal(t, "prefix1", result1)
	assert.Equal(t, "prefix2", result2)
}

func TestMaskAPIKeyNormal(t *testing.T) {
	result := MaskAPIKey("sk-1234567890abcdef")

	assert.Equal(t, "sk-1****cdef", result)
}

func TestMaskAPIKeyShort(t *testing.T) {
	result := MaskAPIKey("short")

	assert.Equal(t, "****", result)
}

func TestMaskAPIKeyVeryShort(t *testing.T) {
	result := MaskAPIKey("ab")

	assert.Equal(t, "****", result)
}

func TestMaskAPIKeyExactly8(t *testing.T) {
	result := MaskAPIKey("12345678")

	assert.Equal(t, "****", result)
}

func TestMaskAPIKeyEmpty(t *testing.T) {
	result := MaskAPIKey("")

	assert.Equal(t, "****", result)
}

func TestMaskAPIKey9Chars(t *testing.T) {
	result := MaskAPIKey("123456789")

	assert.Equal(t, "1234****6789", result)
}

func TestTruncateStringWithPunctuation(t *testing.T) {
	result := TruncateString("Hello, World! This is a test.", 10)

	// Should try to truncate at punctuation
	assert.Contains(t, result, "...")
}

func TestContainsCaseSensitive(t *testing.T) {
	assert.False(t, Contains("Hello World", "hello"))
	assert.True(t, Contains("Hello World", "Hello"))
}

func TestContainsAnyMultipleMatches(t *testing.T) {
	// Should return true on first match
	assert.True(t, ContainsAny("Hello", "He", "ll", "lo"))
}

func TestTruncateStringPreservesContent(t *testing.T) {
	result := TruncateString("Test", 10)

	assert.Equal(t, "Test", result)
}

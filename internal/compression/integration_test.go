package compression

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompress_Integration_CodePreservation verifies that code structure is preserved
func TestCompress_Integration_CodePreservation(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmExtractive,
		TargetRatio:      2.0,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	// Sample Go code with multiple functions
	codeContent := `package main

import "fmt"

func hello() {
    fmt.Println("hello")
    return
}

func world() {
    fmt.Println("world")
    return
}

func goodbye() {
    fmt.Println("goodbye")
    return
}`

	result, err := service.Compress(context.Background(), codeContent, AlgorithmExtractive, 2.0)
	require.NoError(t, err)

	// Verify compression happened
	assert.Less(t, result.Metadata.CompressedSize, result.Metadata.OriginalSize)
	assert.Greater(t, result.Metadata.CompressionRatio, 1.0)

	// Verify at least one complete function remains
	functionCount := strings.Count(result.Content, "func ")
	assert.Greater(t, functionCount, 0, "should preserve at least one complete function")
}

// TestCompress_Integration_MarkdownPreservation verifies markdown structure is preserved
func TestCompress_Integration_MarkdownPreservation(t *testing.T) {
	config := Config{
		DefaultAlgorithm: AlgorithmExtractive,
		TargetRatio:      2.0,
	}

	service, err := NewService(config)
	require.NoError(t, err)

	markdownContent := `# Main Header

This is the introduction section with some content.

## Section 1

Content for section 1 goes here.

## Section 2

Content for section 2 goes here.

## Section 3

Content for section 3 goes here.`

	result, err := service.Compress(context.Background(), markdownContent, AlgorithmExtractive, 2.0)
	require.NoError(t, err)

	// Verify compression happened
	assert.Less(t, result.Metadata.CompressedSize, result.Metadata.OriginalSize)

	// Verify at least one section header remains
	headerCount := strings.Count(result.Content, "#")
	assert.Greater(t, headerCount, 0, "should preserve at least one header")
}

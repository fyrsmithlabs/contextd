package compression_test

import (
	"context"
	"fmt"
	"os"

	"github.com/fyrsmithlabs/contextd/internal/compression"
)

// ExamplePreview demonstrates the preview functionality
func ExamplePreview() {
	// Sample content to compress
	original := `The Go programming language is a statically typed, compiled language.
It was designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson.
Go is syntactically similar to C, but with memory safety and garbage collection.
The language was announced in November 2009 and version 1.0 was released in March 2012.
Go is widely used for building web servers, data pipelines, and cloud-native applications.
It has a rich standard library and excellent concurrency support via goroutines.`

	// Create compression service
	config := compression.Config{
		DefaultAlgorithm: compression.AlgorithmExtractive,
		TargetRatio:      2.0,
	}

	service, err := compression.NewService(config)
	if err != nil {
		panic(err)
	}

	// Compress the content
	ctx := context.Background()
	result, err := service.Compress(ctx, original, compression.AlgorithmExtractive, 2.0)
	if err != nil {
		panic(err)
	}

	// Preview the compression results
	opts := compression.PreviewOptions{
		Width:       80,
		ShowMetrics: true,
		ColorOutput: false, // Disable colors for example output
	}

	err = compression.Preview(os.Stdout, original, result, opts)
	if err != nil {
		panic(err)
	}

	fmt.Println("\n✓ Compression preview generated successfully")
}

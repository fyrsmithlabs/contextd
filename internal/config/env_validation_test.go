package config

import (
	"os"
	"testing"
)

func TestLoad_ValidatesQdrantHost(t *testing.T) {
	defer os.Unsetenv("QDRANT_HOST")
	
	// Invalid hostnames with command injection attempts
	invalidHosts := []string{
		"localhost; rm -rf /",
		"localhost\nmalicious",
		"localhost$(whoami)",
	}

	for _, host := range invalidHosts {
		t.Run(host, func(t *testing.T) {
			os.Setenv("QDRANT_HOST", host)
			cfg := Load()
			
			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for malicious host: %s", host)
			}
		})
	}
}

func TestLoad_ValidatesDataPath(t *testing.T) {
	defer os.Unsetenv("CONTEXTD_DATA_PATH")
	
	// Paths with traversal attempts
	invalidPaths := []string{
		"../../../etc/passwd",
		"/data/../../../etc/passwd",
	}

	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			os.Setenv("CONTEXTD_DATA_PATH", path)
			cfg := Load()
			
			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for path traversal: %s", path)
			}
		})
	}
}

func TestLoad_ValidatesEmbeddingBaseURL(t *testing.T) {
	defer os.Unsetenv("EMBEDDING_BASE_URL")
	
	// Invalid URLs
	invalidURLs := []string{
		"javascript:alert(1)",
		"file:///etc/passwd",
		"ftp://malicious.com",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			os.Setenv("EMBEDDING_BASE_URL", url)
			cfg := Load()
			
			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for invalid URL: %s", url)
			}
		})
	}
}

func TestLoad_AllowsValidConfig(t *testing.T) {
	defer os.Unsetenv("QDRANT_HOST")
	defer os.Unsetenv("CONTEXTD_DATA_PATH")
	defer os.Unsetenv("EMBEDDING_BASE_URL")
	
	os.Setenv("QDRANT_HOST", "localhost")
	os.Setenv("CONTEXTD_DATA_PATH", "/data")
	os.Setenv("EMBEDDING_BASE_URL", "http://localhost:8080")
	
	cfg := Load()
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Valid configuration rejected: %v", err)
	}
}

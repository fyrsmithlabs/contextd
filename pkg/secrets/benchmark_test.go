package secrets

import (
	"os"
	"testing"
)

func BenchmarkLoadAllowlists_SmallFile(b *testing.B) {
	tmpDir := b.TempDir()
	allowlistPath := tmpDir + "/.gitleaks.toml"

	// Create small allowlist (10 patterns)
	content := `[allowlist]
paths = [
  '''pattern1.*''',
  '''pattern2.*''',
  '''pattern3.*''',
  '''pattern4.*''',
  '''pattern5.*''',
  '''pattern6.*''',
  '''pattern7.*''',
  '''pattern8.*''',
  '''pattern9.*''',
  '''pattern10.*'''
]
regexes = []
`
	if err := os.WriteFile(allowlistPath, []byte(content), 0600); err != nil {
		b.Fatalf("Failed to create allowlist: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadAllowlists(tmpDir, "")
		if err != nil {
			b.Fatalf("LoadAllowlists() error = %v", err)
		}
	}
}

func BenchmarkLoadAllowlists_MediumFile(b *testing.B) {
	tmpDir := b.TempDir()
	allowlistPath := tmpDir + "/.gitleaks.toml"

	// Create medium allowlist (100 patterns)
	var content string
	content += "[allowlist]\npaths = [\n"
	for i := 0; i < 100; i++ {
		content += "  '''pattern" + string(rune('0'+i%10)) + ".*''',\n"
	}
	content += "]\nregexes = []\n"

	if err := os.WriteFile(allowlistPath, []byte(content), 0600); err != nil {
		b.Fatalf("Failed to create allowlist: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadAllowlists(tmpDir, "")
		if err != nil {
			b.Fatalf("LoadAllowlists() error = %v", err)
		}
	}
}

func BenchmarkLoadAllowlists_LargeFile(b *testing.B) {
	tmpDir := b.TempDir()
	allowlistPath := tmpDir + "/.gitleaks.toml"

	// Create large allowlist (1000 patterns)
	var content string
	content += "[allowlist]\npaths = [\n"
	for i := 0; i < 1000; i++ {
		content += "  '''pattern" + string(rune('0'+i%10)) + ".*''',\n"
	}
	content += "]\nregexes = []\n"

	if err := os.WriteFile(allowlistPath, []byte(content), 0600); err != nil {
		b.Fatalf("Failed to create allowlist: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadAllowlists(tmpDir, "")
		if err != nil {
			b.Fatalf("LoadAllowlists() error = %v", err)
		}
	}
}

func BenchmarkRedact_NoSecrets(b *testing.B) {
	content := `
package main

func main() {
	println("Hello World")
}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Redact(content, RedactOptions{})
		if err != nil {
			b.Fatalf("Redact() error = %v", err)
		}
	}
}

func BenchmarkRedact_SingleSecret(b *testing.B) {
	content := `const key = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890123456"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Redact(content, RedactOptions{})
		if err != nil {
			b.Fatalf("Redact() error = %v", err)
		}
	}
}

func BenchmarkRedact_LargeFile(b *testing.B) {
	// Generate 10KB file
	var content string
	for i := 0; i < 500; i++ {
		content += "line " + string(rune('0'+i%10)) + " with some content\n"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Redact(content, RedactOptions{})
		if err != nil {
			b.Fatalf("Redact() error = %v", err)
		}
	}
}

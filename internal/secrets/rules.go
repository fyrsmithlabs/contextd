package secrets

// DefaultRules returns the default set of secret detection rules.
// Based on common secret patterns from gitleaks and industry standards.
func DefaultRules() []Rule {
	return []Rule{
		// AWS
		{
			ID:          "aws-access-key-id",
			Description: "AWS Access Key ID",
			Pattern:     `(?i)(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`,
			Keywords:    []string{"aws", "access", "key"},
			Severity:    "high",
		},
		{
			ID:          "aws-secret-access-key",
			Description: "AWS Secret Access Key",
			Pattern:     `(?i)(?:aws_secret_access_key|aws_secret_key|secret_access_key)\s*[:=]\s*['"]?([A-Za-z0-9/+=]{40})['"]?`,
			Keywords:    []string{"aws", "secret"},
			Severity:    "high",
		},

		// Generic API Keys
		{
			ID:          "generic-api-key",
			Description: "Generic API Key",
			Pattern:     `(?i)(?:api[_-]?key|apikey)\s*[:=]\s*['"]?([A-Za-z0-9_\-]{16,64})['"]?`,
			Keywords:    []string{"api", "key"},
			Severity:    "high",
		},
		{
			ID:          "generic-secret",
			Description: "Generic Secret",
			Pattern:     `(?i)(?:secret|password|passwd|pwd)\s*[:=]\s*['"]?([^\s'"]{8,})['"]?`,
			Keywords:    []string{"secret", "password"},
			Severity:    "high",
		},

		// Private Keys
		{
			ID:          "private-key",
			Description: "Private Key",
			Pattern:     `-----BEGIN (?:RSA |DSA |EC |OPENSSH |PGP )?PRIVATE KEY(?:[- ]BLOCK)?-----`,
			Severity:    "high",
		},

		// GitHub (prefixes are self-identifying)
		{
			ID:          "github-token",
			Description: "GitHub Personal Access Token",
			Pattern:     `ghp_[A-Za-z0-9]{36}`,
			Severity:    "high",
		},
		{
			ID:          "github-oauth",
			Description: "GitHub OAuth Access Token",
			Pattern:     `gho_[A-Za-z0-9]{36}`,
			Severity:    "high",
		},
		{
			ID:          "github-app",
			Description: "GitHub App Token",
			Pattern:     `(?:ghu|ghs)_[A-Za-z0-9]{36}`,
			Severity:    "high",
		},
		{
			ID:          "github-fine-grained",
			Description: "GitHub Fine-grained Personal Access Token",
			Pattern:     `github_pat_[A-Za-z0-9_]{22,}`,
			Severity:    "high",
		},

		// GitLab (prefix is self-identifying)
		{
			ID:          "gitlab-token",
			Description: "GitLab Personal Access Token",
			Pattern:     `glpat-[A-Za-z0-9\-]{20,}`,
			Severity:    "high",
		},

		// Slack (prefix is self-identifying)
		{
			ID:          "slack-token",
			Description: "Slack Token",
			Pattern:     `xox[baprs]-[A-Za-z0-9\-]{10,}`,
			Severity:    "high",
		},

		// Stripe (prefix is self-identifying)
		{
			ID:          "stripe-key",
			Description: "Stripe API Key",
			Pattern:     `(?:sk|pk)_(?:live|test)_[A-Za-z0-9]{24,}`,
			Severity:    "high",
		},

		// Database URLs
		{
			ID:          "database-url",
			Description: "Database Connection URL with credentials",
			Pattern:     `(?i)(?:postgres|mysql|mongodb|redis|amqp)://[^:]+:[^@]+@[^\s]+`,
			Keywords:    []string{"database", "db", "connection"},
			Severity:    "high",
		},

		// JWT (eyJ prefix is self-identifying - base64 encoded JSON header)
		{
			ID:          "jwt",
			Description: "JSON Web Token",
			Pattern:     `eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`,
			Severity:    "medium",
		},

		// Google
		{
			ID:          "google-api-key",
			Description: "Google API Key",
			Pattern:     `AIza[A-Za-z0-9_\-]{35}`,
			Keywords:    []string{"google"},
			Severity:    "high",
		},
		{
			ID:          "google-oauth",
			Description: "Google OAuth Client Secret",
			Pattern:     `(?i)client_secret['":\s]+[A-Za-z0-9_\-]{24}`,
			Keywords:    []string{"google", "oauth"},
			Severity:    "high",
		},

		// Azure
		{
			ID:          "azure-storage-key",
			Description: "Azure Storage Account Key",
			Pattern:     `(?i)(?:account_?key|storage_?key)\s*[:=]\s*['"]?([A-Za-z0-9+/]{86}==)['"]?`,
			Keywords:    []string{"azure", "storage"},
			Severity:    "high",
		},

		// Anthropic
		{
			ID:          "anthropic-api-key",
			Description: "Anthropic API Key",
			Pattern:     `sk-ant-[A-Za-z0-9_\-]{90,}`,
			Keywords:    []string{"anthropic", "claude"},
			Severity:    "high",
		},

		// OpenAI
		{
			ID:          "openai-api-key",
			Description: "OpenAI API Key",
			Pattern:     `sk-[A-Za-z0-9]{48,}`,
			Keywords:    []string{"openai"},
			Severity:    "high",
		},

		// SendGrid (SG. prefix is self-identifying)
		{
			ID:          "sendgrid-api-key",
			Description: "SendGrid API Key",
			Pattern:     `SG\.[A-Za-z0-9_\-]{22,}\.[A-Za-z0-9_\-]{43,}`,
			Severity:    "high",
		},

		// Twilio (SK prefix with 32 chars - keep keywords due to common prefix)
		{
			ID:          "twilio-api-key",
			Description: "Twilio API Key",
			Pattern:     `SK[A-Za-z0-9]{32}`,
			Keywords:    []string{"twilio"},
			Severity:    "high",
		},

		// npm (npm_ prefix is self-identifying)
		{
			ID:          "npm-token",
			Description: "npm Access Token",
			Pattern:     `npm_[A-Za-z0-9]{36}`,
			Severity:    "high",
		},

		// Heroku
		{
			ID:          "heroku-api-key",
			Description: "Heroku API Key",
			Pattern:     `(?i)heroku[_-]?api[_-]?key\s*[:=]\s*[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}`,
			Keywords:    []string{"heroku"},
			Severity:    "high",
		},

		// Generic Bearer Token
		{
			ID:          "bearer-token",
			Description: "Bearer Token in Authorization Header",
			Pattern:     `(?i)(?:authorization|bearer)\s*[:=]\s*['"]?bearer\s+([A-Za-z0-9_\-\.]{20,})['"]?`,
			Keywords:    []string{"authorization", "bearer"},
			Severity:    "medium",
		},

		// SSH Key
		{
			ID:          "ssh-key",
			Description: "SSH Private Key",
			Pattern:     `-----BEGIN (?:OPENSSH |RSA |DSA |EC )?PRIVATE KEY-----`,
			Severity:    "high",
		},

		// Environment Variables with sensitive names
		{
			ID:          "env-credential",
			Description: "Environment Variable with Credential",
			Pattern:     `(?i)(?:^|[^A-Za-z0-9_])(?:DB_PASSWORD|DATABASE_PASSWORD|MYSQL_PASSWORD|POSTGRES_PASSWORD|REDIS_PASSWORD|MONGO_PASSWORD|API_SECRET|APP_SECRET|SECRET_KEY|ENCRYPTION_KEY|PRIVATE_KEY|AUTH_TOKEN|ACCESS_TOKEN|REFRESH_TOKEN)\s*[:=]\s*['"]?([^\s'"]{8,})['"]?`,
			Severity:    "high",
		},
	}
}

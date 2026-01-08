# LLM Integration for Extraction

Implement actual Claude/OpenAI API calls in internal/extraction/llm.go to enable intelligent extraction of facts, patterns, and insights from conversations. Currently contains TODO stubs.

## Rationale
The extraction package has TODO comments for LLM integration. Without real LLM calls, automatic memory extraction can't identify what's worth remembering. This enables auto-memory capture (feature-9) and distiller (feature-2). Addresses the known gap in current state.

## User Stories
- As a developer, I want automatic extraction of useful facts from my conversations so that I don't have to manually record memories
- As a privacy-conscious user, I want the option to use local LLMs so that my data never leaves my machine
- As a team lead, I want consistent extraction quality so that team knowledge is reliably captured

## Acceptance Criteria
- [ ] ClaudeExtractor makes real API calls to Claude API
- [ ] OpenAIExtractor makes real API calls to OpenAI API
- [ ] API keys are loaded from config/environment securely
- [ ] Extraction prompts are configurable and well-tested
- [ ] Rate limiting and error handling are implemented
- [ ] Unit tests mock the API calls for CI

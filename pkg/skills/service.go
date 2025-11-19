package skills

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
)

var (
	// tracer for OpenTelemetry instrumentation
	tracer = otel.Tracer("contextd.skills")
)

// VectorStore defines the interface for vector storage operations.
//
// This interface allows for testing and decouples the service from
// the specific vector store implementation.
type VectorStore interface {
	AddDocuments(ctx context.Context, docs []vectorstore.Document) error
	SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

// Service provides skills management with vector storage and semantic search.
//
// Skills are stored in the "shared" database for global access across all projects.
// The service uses a vector store for persistence and enables semantic search
// using embeddings.
type Service struct {
	vectorStore VectorStore
}

// NewService creates a new skills service.
//
// The vector store must be configured with an embedder for automatic
// embedding generation. Skills are stored in a shared collection accessible
// to all projects.
func NewService(vs VectorStore) *Service {
	return &Service{
		vectorStore: vs,
	}
}

// Save stores a skill with automatic embedding generation.
//
// The skill is validated before saving. A unique ID is generated if not
// provided. Timestamps are set automatically. The skill name, description,
// and content are embedded for semantic search.
//
// Returns the skill ID on success, or an error if validation or storage fails.
func (s *Service) Save(ctx context.Context, skill *Skill) (string, error) {
	ctx, span := tracer.Start(ctx, "skills.Save")
	defer span.End()

	// Validate skill
	if skill == nil {
		return "", errors.New("skill cannot be nil")
	}

	if err := skill.Validate(); err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("validating skill: %w", err)
	}

	// Generate ID if not provided
	if skill.ID == "" {
		skill.ID = "skill_" + uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	skill.UpdatedAt = now

	// Build content for embedding (name + description + content)
	embedContent := fmt.Sprintf("%s\n\n%s\n\n%s", skill.Name, skill.Description, skill.Content)

	// Prepare metadata for vector store
	metadata := map[string]interface{}{
		"id":          skill.ID,
		"name":        skill.Name,
		"description": skill.Description,
		"content":     skill.Content,
		"created_at":  skill.CreatedAt.Format(time.RFC3339),
		"updated_at":  skill.UpdatedAt.Format(time.RFC3339),
	}

	// Add tags
	if len(skill.Tags) > 0 {
		metadata["tags"] = skill.Tags
	}

	// Add user metadata
	if skill.Metadata != nil {
		for k, v := range skill.Metadata {
			metadata[k] = v
		}
	}

	// Create vector store document
	doc := vectorstore.Document{
		ID:       skill.ID,
		Content:  embedContent,
		Metadata: metadata,
	}

	// Store in vector database (embedding generated automatically)
	if err := s.vectorStore.AddDocuments(ctx, []vectorstore.Document{doc}); err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("storing skill: %w", err)
	}

	return skill.ID, nil
}

// Search finds semantically similar skills using vector similarity.
//
// The search query is embedded and compared against all skills in the
// shared database. Results are ordered by similarity score (highest first).
//
// Parameters:
//   - ctx: Context for cancellation
//   - query: Search query text
//   - limit: Maximum number of results to return
//
// Returns skills ordered by similarity score (highest first).
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*SearchResult, error) {
	ctx, span := tracer.Start(ctx, "skills.Search")
	defer span.End()

	if query == "" {
		err := errors.New("query cannot be empty")
		span.RecordError(err)
		return nil, err
	}

	if limit <= 0 {
		err := errors.New("limit must be positive")
		span.RecordError(err)
		return nil, err
	}

	// Skills are stored in shared database - no filters needed
	// (all skills are globally accessible)
	filters := map[string]interface{}{}

	// Execute semantic search
	results, err := s.vectorStore.SearchWithFilters(ctx, query, limit, filters)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("searching skills: %w", err)
	}

	// Convert results to skills
	searchResults := make([]*SearchResult, 0, len(results))
	for _, result := range results {
		skill, err := resultToSkill(result)
		if err != nil {
			// Log warning but continue processing other results
			continue
		}

		searchResults = append(searchResults, &SearchResult{
			Skill: skill,
			Score: result.Score,
		})
	}

	return searchResults, nil
}

// Create creates a new skill with automatic embedding.
//
// This is an alias for Save() for API consistency.
func (s *Service) Create(ctx context.Context, skill *Skill) (string, error) {
	return s.Save(ctx, skill)
}

// Get retrieves a specific skill by ID.
//
// Returns ErrSkillNotFound if the skill doesn't exist.
func (s *Service) Get(ctx context.Context, id string) (*Skill, error) {
	ctx, span := tracer.Start(ctx, "skills.Get")
	defer span.End()

	if id == "" {
		err := errors.New("skill ID is required")
		span.RecordError(err)
		return nil, err
	}

	// Search by ID using metadata filter
	filters := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key": "id",
				"match": map[string]interface{}{
					"value": id,
				},
			},
		},
	}

	results, err := s.vectorStore.SearchWithFilters(ctx, id, 1, filters)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("getting skill: %w", err)
	}

	if len(results) == 0 {
		return nil, ErrSkillNotFound
	}

	skill, err := resultToSkill(results[0])
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("converting skill: %w", err)
	}

	return skill, nil
}

// Helper functions

// resultToSkill converts a vector store search result to a Skill.
func resultToSkill(result vectorstore.SearchResult) (*Skill, error) {
	skill := &Skill{
		Metadata: make(map[string]interface{}),
	}

	// Extract required fields
	id, ok := result.Metadata["id"].(string)
	if !ok {
		return nil, errors.New("missing or invalid ID in result")
	}
	skill.ID = id

	name, ok := result.Metadata["name"].(string)
	if !ok {
		return nil, errors.New("missing or invalid name in result")
	}
	skill.Name = name

	description, ok := result.Metadata["description"].(string)
	if !ok {
		return nil, errors.New("missing or invalid description in result")
	}
	skill.Description = description

	content, ok := result.Metadata["content"].(string)
	if !ok {
		return nil, errors.New("missing or invalid content in result")
	}
	skill.Content = content

	// Parse timestamps
	if createdStr, ok := result.Metadata["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			skill.CreatedAt = t
		}
	}
	if updatedStr, ok := result.Metadata["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			skill.UpdatedAt = t
		}
	}

	// Extract tags
	if tags, ok := result.Metadata["tags"].([]interface{}); ok {
		skill.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				skill.Tags = append(skill.Tags, tagStr)
			}
		}
	} else if tags, ok := result.Metadata["tags"].([]string); ok {
		skill.Tags = tags
	}

	// Copy all metadata (excluding system fields)
	for k, v := range result.Metadata {
		if k != "id" && k != "name" && k != "description" && k != "content" &&
			k != "created_at" && k != "updated_at" && k != "tags" {
			skill.Metadata[k] = v
		}
	}

	return skill, nil
}

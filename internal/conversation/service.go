package conversation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
)

// Scrubber is an interface for secret scrubbing.
type Scrubber interface {
	Scrub(content string) ScrubResult
}

// ScrubResult holds the result of secret scrubbing.
type ScrubResult interface {
	GetScrubbed() string
}

// Service implements ConversationService for indexing and searching conversations.
type Service struct {
	parser    *Parser
	extractor *Extractor
	store     vectorstore.Store
	scrubber  Scrubber
	logger    *zap.Logger

	// Configuration
	conversationsPath string
}

// ServiceConfig holds configuration for the conversation service.
type ServiceConfig struct {
	ConversationsPath string // Base path for conversation files
}

// NewService creates a new conversation service.
func NewService(
	store vectorstore.Store,
	scrubber Scrubber,
	logger *zap.Logger,
	cfg ServiceConfig,
) *Service {
	conversationsPath := cfg.ConversationsPath
	if conversationsPath == "" {
		// Default Claude Code location
		home, _ := os.UserHomeDir()
		conversationsPath = filepath.Join(home, ".claude", "projects")
	}

	return &Service{
		parser:            NewParser(),
		extractor:         NewExtractor(),
		store:             store,
		scrubber:          scrubber,
		logger:            logger,
		conversationsPath: conversationsPath,
	}
}

// collectionName returns the collection name for a tenant/project.
func (s *Service) collectionName(tenantID, projectPath string) string {
	// Sanitize tenant ID and project path for collection name
	tenantID = sanitizeForCollectionName(tenantID)
	projectName := sanitizeForCollectionName(filepath.Base(projectPath))

	return fmt.Sprintf("%s_%s_conversations", tenantID, projectName)
}

// sanitizeForCollectionName ensures a string is safe for use in collection names.
// Only allows alphanumeric characters and underscores.
// Uses a hash prefix when sanitization produces no alphanumeric characters to avoid collisions.
//
// Hash collision note: When using hash fallback, we use first 8 bytes of SHA-256 (2^64 space).
// This provides sufficient collision resistance for typical tenant/project naming scenarios.
// For untrusted input with adversarial collision attempts, consider using full hash or
// adding collision detection in CreateCollection.
func sanitizeForCollectionName(s string) string {
	original := s
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r)
		case r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == '-' || r == ' ' || r == '_' || r == '.':
			result.WriteRune('_')
		// Skip other characters
		}
	}
	// Use hash prefix when result is empty to avoid collisions between
	// different all-unicode strings that would otherwise all become "default"
	if result.Len() == 0 {
		hash := sha256.Sum256([]byte(original))
		return "h_" + hex.EncodeToString(hash[:8]) // 16-char hex prefix
	}
	return result.String()
}

// Index processes and stores conversations for a project.
func (s *Service) Index(ctx context.Context, opts IndexOptions) (*IndexResult, error) {
	startTime := time.Now()

	s.logger.Info("starting conversation indexing",
		zap.String("project_path", opts.ProjectPath),
		zap.String("tenant_id", opts.TenantID),
		zap.Bool("force", opts.Force),
	)

	// Determine conversation directory
	convDir := s.getConversationDir(opts.ProjectPath)
	if _, err := os.Stat(convDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("conversation directory not found: %s", convDir)
	}

	// Parse all conversations with error tracking
	parseResult, err := s.parser.ParseAllWithErrors(convDir)
	if err != nil {
		return nil, fmt.Errorf("parsing conversations: %w", err)
	}
	sessionMessages := parseResult.Messages

	// Log parse errors (they're non-fatal)
	if parseResult.ErrorCount > 0 {
		s.logger.Warn("encountered parse errors during indexing",
			zap.Int("error_count", parseResult.ErrorCount),
			zap.Int("files_with_errors", len(parseResult.Errors)),
		)
	}

	// Filter sessions if specified
	if len(opts.SessionIDs) > 0 {
		filtered := make(map[string][]RawMessage)
		for _, sid := range opts.SessionIDs {
			if msgs, ok := sessionMessages[sid]; ok {
				filtered[sid] = msgs
			}
		}
		sessionMessages = filtered
	}

	// Create or get collection
	collName := s.collectionName(opts.TenantID, opts.ProjectPath)

	// Add tenant context for vectorstore operations
	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  opts.TenantID,
		ProjectID: filepath.Base(opts.ProjectPath),
	})

	// Ensure collection exists
	exists, err := s.store.CollectionExists(ctx, collName)
	if err != nil {
		return nil, fmt.Errorf("checking collection: %w", err)
	}
	if !exists {
		if err := s.store.CreateCollection(ctx, collName, 0); err != nil {
			return nil, fmt.Errorf("creating collection: %w", err)
		}
	}

	result := &IndexResult{
		FilesReferenced: []string{},
	}
	var indexErrors []error

	// Include parse errors in result
	for _, pe := range parseResult.Errors {
		indexErrors = append(indexErrors, fmt.Errorf("%s", pe.Error))
	}

	filesSet := make(map[string]bool)

	// Process each session
	for sessionID, messages := range sessionMessages {
		s.logger.Debug("indexing session",
			zap.String("session_id", sessionID),
			zap.Int("message_count", len(messages)),
		)

		// Sort messages by timestamp
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Timestamp.Before(messages[j].Timestamp)
		})

		// Convert messages to documents
		for idx, msg := range messages {
			doc, err := s.messageToDocument(msg, idx, sessionID)
			if err != nil {
				indexErrors = append(indexErrors, err)
				continue
			}

			// Extract and track file references
			for _, ref := range doc.FilesDiscussed {
				filesSet[ref.Path] = true
			}

			// Convert to vectorstore document
			vsDoc := s.toVectorstoreDocument(doc)

			// Add to store - set collection in document metadata
			vsDoc.Metadata["collection"] = collName
			if _, err := s.store.AddDocuments(ctx, []vectorstore.Document{vsDoc}); err != nil {
				indexErrors = append(indexErrors, fmt.Errorf("adding message %s: %w", doc.ID, err))
				continue
			}

			result.MessagesIndexed++
		}

		result.SessionsIndexed++
	}

	// Collect referenced files
	for path := range filesSet {
		result.FilesReferenced = append(result.FilesReferenced, path)
	}
	result.Errors = indexErrors

	s.logger.Info("conversation indexing complete",
		zap.Int("sessions", result.SessionsIndexed),
		zap.Int("messages", result.MessagesIndexed),
		zap.Int("decisions", result.DecisionsExtracted),
		zap.Int("files", len(result.FilesReferenced)),
		zap.Int("errors", len(result.Errors)),
		zap.Duration("duration", time.Since(startTime)),
	)

	return result, nil
}

// getConversationDir determines the conversation directory for a project.
func (s *Service) getConversationDir(projectPath string) string {
	// Claude Code stores conversations in ~/.claude/projects/{project-hash}/
	// We need to find the right directory based on project path

	// For now, use a simplified approach - look for project-specific subdirectory
	// or fall back to the base conversations path
	projectName := filepath.Base(projectPath)
	projectDir := filepath.Join(s.conversationsPath, projectName)

	if _, err := os.Stat(projectDir); err == nil {
		return projectDir
	}

	// Try with path hash (Claude's actual behavior)
	// For MVP, just return base path
	return s.conversationsPath
}

// messageToDocument converts a RawMessage to a MessageDocument.
func (s *Service) messageToDocument(msg RawMessage, index int, sessionID string) (*MessageDocument, error) {
	// Scrub content
	scrubbedContent := msg.Content
	if s.scrubber != nil {
		result := s.scrubber.Scrub(msg.Content)
		scrubbedContent = result.GetScrubbed()
	}

	// Extract metadata
	files, commits := s.extractor.ExtractMetadata(msg)

	doc := &MessageDocument{
		ConversationDocument: ConversationDocument{
			ID:               uuid.New().String(),
			SessionID:        sessionID,
			Type:             TypeMessage,
			Timestamp:        msg.Timestamp,
			Content:          scrubbedContent,
			Tags:             []string{}, // Will be populated by extraction package
			FilesDiscussed:   files,
			CommitsMade:      commits,
			IndexedAt:        time.Now(),
			ExtractionMethod: "heuristic",
		},
		Role:         msg.Role,
		MessageUUID:  msg.UUID,
		MessageIndex: index,
	}

	return doc, nil
}

// toVectorstoreDocument converts a MessageDocument to a vectorstore.Document.
func (s *Service) toVectorstoreDocument(doc *MessageDocument) vectorstore.Document {
	metadata := map[string]interface{}{
		"session_id":        doc.SessionID,
		"type":              string(doc.Type),
		"timestamp":         doc.Timestamp.Unix(),
		"role":              string(doc.Role),
		"message_uuid":      doc.MessageUUID,
		"message_index":     doc.MessageIndex,
		"indexed_at":        doc.IndexedAt.Unix(),
		"extraction_method": doc.ExtractionMethod,
	}

	// Add tags
	if len(doc.Tags) > 0 {
		metadata["tags"] = doc.Tags
	}
	if doc.Domain != "" {
		metadata["domain"] = doc.Domain
	}

	// Add file references
	if len(doc.FilesDiscussed) > 0 {
		files := make([]string, len(doc.FilesDiscussed))
		for i, f := range doc.FilesDiscussed {
			files[i] = f.Path
		}
		metadata["files_discussed"] = files
	}

	// Add commit references
	if len(doc.CommitsMade) > 0 {
		commits := make([]string, len(doc.CommitsMade))
		for i, c := range doc.CommitsMade {
			commits[i] = c.SHA
		}
		metadata["commits_made"] = commits
	}

	return vectorstore.Document{
		ID:       doc.ID,
		Content:  doc.Content,
		Metadata: metadata,
	}
}

// Search finds relevant conversations.
func (s *Service) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	startTime := time.Now()

	// Set default limit
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	collName := s.collectionName(opts.TenantID, opts.ProjectPath)

	// Add tenant context
	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  opts.TenantID,
		ProjectID: filepath.Base(opts.ProjectPath),
	})

	// Build filters
	filters := make(map[string]interface{})
	if len(opts.Types) > 0 {
		filters["type"] = opts.Types
	}
	if len(opts.Tags) > 0 {
		filters["tags"] = opts.Tags
	}
	if opts.FilePath != "" {
		filters["files_discussed"] = opts.FilePath
	}
	if opts.Domain != "" {
		filters["domain"] = opts.Domain
	}

	// Search vectorstore
	results, err := s.store.SearchInCollection(ctx, collName, opts.Query, limit, filters)
	if err != nil {
		return nil, fmt.Errorf("searching conversations: %w", err)
	}

	// Convert results
	hits := make([]SearchHit, len(results))
	for i, r := range results {
		doc := s.resultToDocument(r)
		hits[i] = SearchHit{
			Document: doc,
			Score:    float64(r.Score),
		}
	}

	return &SearchResult{
		Query:   opts.Query,
		Results: hits,
		Total:   len(hits),
		Took:    time.Since(startTime),
	}, nil
}

// resultToDocument converts a vectorstore result to a ConversationDocument.
func (s *Service) resultToDocument(r vectorstore.SearchResult) ConversationDocument {
	doc := ConversationDocument{
		ID:      r.ID,
		Content: r.Content,
	}

	// Extract metadata
	if sessionID, ok := r.Metadata["session_id"].(string); ok {
		doc.SessionID = sessionID
	}
	if docType, ok := r.Metadata["type"].(string); ok {
		doc.Type = DocumentType(docType)
	}
	if ts, ok := r.Metadata["timestamp"].(float64); ok {
		doc.Timestamp = time.Unix(int64(ts), 0)
	}
	if tags, ok := r.Metadata["tags"].([]interface{}); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				doc.Tags = append(doc.Tags, s)
			}
		}
	}
	if domain, ok := r.Metadata["domain"].(string); ok {
		doc.Domain = domain
	}

	return doc
}

// Ensure Service implements ConversationService.
var _ ConversationService = (*Service)(nil)

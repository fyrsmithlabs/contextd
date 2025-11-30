package remediation

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const instrumentationName = "github.com/fyrsmithlabs/contextd/internal/remediation"

// Service provides remediation management operations.
type Service interface {
	// Search finds remediations by semantic similarity to error message/pattern.
	Search(ctx context.Context, req *SearchRequest) ([]*ScoredRemediation, error)

	// Record creates a new remediation.
	Record(ctx context.Context, req *RecordRequest) (*Remediation, error)

	// Get retrieves a remediation by ID.
	Get(ctx context.Context, tenantID, remediationID string) (*Remediation, error)

	// Feedback records feedback on a remediation, adjusting confidence.
	Feedback(ctx context.Context, req *FeedbackRequest) error

	// Delete removes a remediation.
	Delete(ctx context.Context, tenantID, remediationID string) error

	// Close closes the service.
	Close() error
}

// Config configures the remediation service.
type Config struct {
	// CollectionPrefix is the prefix for remediation collections (default: remediations)
	CollectionPrefix string

	// VectorSize is the dimension of embedding vectors (default: 1536)
	VectorSize uint64

	// DefaultConfidence is the initial confidence for new remediations (default: 0.5)
	DefaultConfidence float64

	// FeedbackDelta is how much feedback changes confidence (default: 0.1)
	FeedbackDelta float64

	// MinConfidence is the minimum confidence threshold (default: 0.1)
	MinConfidence float64

	// MaxConfidence is the maximum confidence (default: 1.0)
	MaxConfidence float64
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *Config {
	return &Config{
		CollectionPrefix:  "remediations",
		VectorSize:        1536,
		DefaultConfidence: 0.5,
		FeedbackDelta:     0.1,
		MinConfidence:     0.1,
		MaxConfidence:     1.0,
	}
}

// service implements the Service interface.
type service struct {
	config   *Config
	qdrant   QdrantClient
	embedder Embedder
	logger   *zap.Logger

	// Telemetry
	tracer          trace.Tracer
	meter           metric.Meter
	searchCounter   metric.Int64Counter
	recordCounter   metric.Int64Counter
	feedbackCounter metric.Int64Counter

	mu     sync.RWMutex
	closed bool
}

// NewService creates a new remediation service.
func NewService(cfg *Config, qc QdrantClient, embedder Embedder, logger *zap.Logger) (Service, error) {
	if cfg == nil {
		cfg = DefaultServiceConfig()
	}
	if qc == nil {
		return nil, errors.New("qdrant client is required")
	}
	if embedder == nil {
		return nil, errors.New("embedder is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	s := &service{
		config:   cfg,
		qdrant:   qc,
		embedder: embedder,
		logger:   logger,
		tracer:   otel.Tracer(instrumentationName),
		meter:    otel.Meter(instrumentationName),
	}

	s.initMetrics()

	return s, nil
}

// initMetrics initializes OpenTelemetry metrics.
func (s *service) initMetrics() {
	var err error

	s.searchCounter, err = s.meter.Int64Counter(
		"contextd.remediation.searches_total",
		metric.WithDescription("Total number of remediation searches"),
		metric.WithUnit("{search}"),
	)
	if err != nil {
		s.logger.Warn("failed to create search counter", zap.Error(err))
	}

	s.recordCounter, err = s.meter.Int64Counter(
		"contextd.remediation.records_total",
		metric.WithDescription("Total number of remediations recorded"),
		metric.WithUnit("{record}"),
	)
	if err != nil {
		s.logger.Warn("failed to create record counter", zap.Error(err))
	}

	s.feedbackCounter, err = s.meter.Int64Counter(
		"contextd.remediation.feedbacks_total",
		metric.WithDescription("Total number of feedback events"),
		metric.WithUnit("{feedback}"),
	)
	if err != nil {
		s.logger.Warn("failed to create feedback counter", zap.Error(err))
	}
}

// collectionName returns the collection name for a given tenant and scope.
func (s *service) collectionName(tenantID string, scope Scope, teamID, projectPath string) string {
	switch scope {
	case ScopeOrg:
		return fmt.Sprintf("%s_org_%s", s.config.CollectionPrefix, tenantID)
	case ScopeTeam:
		return fmt.Sprintf("%s_team_%s_%s", s.config.CollectionPrefix, tenantID, teamID)
	case ScopeProject:
		return fmt.Sprintf("%s_project_%s_%s", s.config.CollectionPrefix, tenantID, sanitizePath(projectPath))
	default:
		return fmt.Sprintf("%s_org_%s", s.config.CollectionPrefix, tenantID)
	}
}

// sanitizePath converts a path to a safe collection name suffix.
func sanitizePath(path string) string {
	// Simple sanitization - replace / with _
	result := ""
	for _, c := range path {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result += string(c)
		} else {
			result += "_"
		}
	}
	return result
}

// Search finds remediations by semantic similarity.
func (s *service) Search(ctx context.Context, req *SearchRequest) ([]*ScoredRemediation, error) {
	ctx, span := s.tracer.Start(ctx, "remediation.search")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("scope", string(req.Scope)),
		attribute.String("category", string(req.Category)),
		attribute.Int("limit", req.Limit),
		attribute.Float64("min_confidence", req.MinConfidence),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	// Generate embedding if not provided
	var vector []float32
	if len(req.Vector) > 0 {
		vector = req.Vector
	} else if req.Query != "" {
		var err error
		vector, err = s.embedder.Embed(ctx, req.Query)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("failed to embed query: %w", err)
		}
	} else {
		return nil, errors.New("either query or vector is required")
	}

	limit := uint64(req.Limit)
	if limit == 0 {
		limit = 10
	}

	// Build filter
	filter := s.buildSearchFilter(req)

	// Determine which collections to search
	collections := s.getSearchCollections(req)

	var allResults []*ScoredRemediation

	for _, collection := range collections {
		// Check if collection exists
		exists, err := s.qdrant.CollectionExists(ctx, collection)
		if err != nil {
			s.logger.Warn("failed to check collection", zap.String("collection", collection), zap.Error(err))
			continue
		}
		if !exists {
			continue
		}

		results, err := s.qdrant.Search(ctx, collection, vector, limit, filter)
		if err != nil {
			s.logger.Warn("search failed", zap.String("collection", collection), zap.Error(err))
			continue
		}

		for _, r := range results {
			rem := payloadToRemediation(r.Payload)
			if rem != nil {
				rem.ID = r.ID
				allResults = append(allResults, &ScoredRemediation{
					Remediation: *rem,
					Score:       float64(r.Score),
				})
			}
		}
	}

	// Sort by score and limit
	allResults = sortAndLimit(allResults, int(limit))

	// Record metrics
	if s.searchCounter != nil {
		s.searchCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("scope", string(req.Scope)),
			attribute.Int("result_count", len(allResults)),
		))
	}

	span.SetAttributes(attribute.Int("result_count", len(allResults)))
	return allResults, nil
}

// buildSearchFilter creates a Qdrant filter from search request.
func (s *service) buildSearchFilter(req *SearchRequest) *QdrantFilter {
	filter := &QdrantFilter{}

	// Confidence filter
	if req.MinConfidence > 0 {
		filter.Must = append(filter.Must, QdrantCondition{
			Field: "confidence",
			Range: &QdrantRangeCondition{Gte: &req.MinConfidence},
		})
	}

	// Category filter
	if req.Category != "" {
		filter.Must = append(filter.Must, QdrantCondition{
			Field: "category",
			Match: string(req.Category),
		})
	}

	if len(filter.Must) == 0 && len(filter.Should) == 0 && len(filter.MustNot) == 0 {
		return nil
	}

	return filter
}

// getSearchCollections returns the collections to search based on scope and hierarchy.
func (s *service) getSearchCollections(req *SearchRequest) []string {
	var collections []string

	switch req.Scope {
	case ScopeProject:
		collections = append(collections, s.collectionName(req.TenantID, ScopeProject, req.TeamID, req.ProjectPath))
		if req.IncludeHierarchy {
			collections = append(collections, s.collectionName(req.TenantID, ScopeTeam, req.TeamID, ""))
			collections = append(collections, s.collectionName(req.TenantID, ScopeOrg, "", ""))
		}
	case ScopeTeam:
		collections = append(collections, s.collectionName(req.TenantID, ScopeTeam, req.TeamID, ""))
		if req.IncludeHierarchy {
			collections = append(collections, s.collectionName(req.TenantID, ScopeOrg, "", ""))
		}
	case ScopeOrg:
		collections = append(collections, s.collectionName(req.TenantID, ScopeOrg, "", ""))
	default:
		// Search all scopes if not specified
		if req.ProjectPath != "" {
			collections = append(collections, s.collectionName(req.TenantID, ScopeProject, req.TeamID, req.ProjectPath))
		}
		if req.TeamID != "" {
			collections = append(collections, s.collectionName(req.TenantID, ScopeTeam, req.TeamID, ""))
		}
		collections = append(collections, s.collectionName(req.TenantID, ScopeOrg, "", ""))
	}

	return collections
}

// Record creates a new remediation.
func (s *service) Record(ctx context.Context, req *RecordRequest) (*Remediation, error) {
	ctx, span := s.tracer.Start(ctx, "remediation.record")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("scope", string(req.Scope)),
		attribute.String("category", string(req.Category)),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	// Generate embedding from problem + symptoms + root_cause
	text := req.Title + " " + req.Problem + " " + req.RootCause
	if len(req.Symptoms) > 0 {
		for _, symptom := range req.Symptoms {
			text += " " + symptom
		}
	}

	vector, err := s.embedder.Embed(ctx, text)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to embed remediation: %w", err)
	}

	// Create remediation
	now := time.Now()
	confidence := req.Confidence
	if confidence == 0 {
		confidence = s.config.DefaultConfidence
	}

	rem := &Remediation{
		ID:            uuid.New().String(),
		Title:         req.Title,
		Problem:       req.Problem,
		Symptoms:      req.Symptoms,
		RootCause:     req.RootCause,
		Solution:      req.Solution,
		CodeDiff:      req.CodeDiff,
		AffectedFiles: req.AffectedFiles,
		Category:      req.Category,
		Confidence:    confidence,
		UsageCount:    0,
		Tags:          req.Tags,
		Scope:         req.Scope,
		TenantID:      req.TenantID,
		TeamID:        req.TeamID,
		ProjectPath:   req.ProjectPath,
		SessionID:     req.SessionID,
		CreatedAt:     now,
		UpdatedAt:     now,
		Vector:        vector,
	}

	// Get collection name
	collection := s.collectionName(req.TenantID, req.Scope, req.TeamID, req.ProjectPath)

	// Ensure collection exists
	exists, err := s.qdrant.CollectionExists(ctx, collection)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to check collection: %w", err)
	}
	if !exists {
		if err := s.qdrant.CreateCollection(ctx, collection, s.config.VectorSize); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// Store in Qdrant
	point := &QdrantPoint{
		ID:      rem.ID,
		Vector:  vector,
		Payload: remediationToPayload(rem),
	}

	if err := s.qdrant.Upsert(ctx, collection, []*QdrantPoint{point}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to store remediation: %w", err)
	}

	// Record metrics
	if s.recordCounter != nil {
		s.recordCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("scope", string(req.Scope)),
			attribute.String("category", string(req.Category)),
		))
	}

	s.logger.Info("recorded remediation",
		zap.String("id", rem.ID),
		zap.String("title", rem.Title),
		zap.String("category", string(rem.Category)),
		zap.String("scope", string(rem.Scope)),
	)

	span.SetAttributes(attribute.String("remediation_id", rem.ID))
	return rem, nil
}

// Get retrieves a remediation by ID.
func (s *service) Get(ctx context.Context, tenantID, remediationID string) (*Remediation, error) {
	ctx, span := s.tracer.Start(ctx, "remediation.get")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", tenantID),
		attribute.String("remediation_id", remediationID),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	// Try each scope level
	scopes := []Scope{ScopeProject, ScopeTeam, ScopeOrg}
	for _, scope := range scopes {
		collection := s.collectionName(tenantID, scope, "", "")
		exists, err := s.qdrant.CollectionExists(ctx, collection)
		if err != nil || !exists {
			continue
		}

		points, err := s.qdrant.Get(ctx, collection, []string{remediationID})
		if err != nil {
			continue
		}
		if len(points) > 0 {
			rem := payloadToRemediation(points[0].Payload)
			if rem != nil {
				rem.ID = points[0].ID
				rem.Vector = points[0].Vector
				return rem, nil
			}
		}
	}

	return nil, fmt.Errorf("remediation not found: %s", remediationID)
}

// Feedback records feedback on a remediation.
func (s *service) Feedback(ctx context.Context, req *FeedbackRequest) error {
	ctx, span := s.tracer.Start(ctx, "remediation.feedback")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", req.TenantID),
		attribute.String("remediation_id", req.RemediationID),
		attribute.String("rating", string(req.Rating)),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return errors.New("service is closed")
	}
	s.mu.RUnlock()

	// Get the remediation
	rem, err := s.Get(ctx, req.TenantID, req.RemediationID)
	if err != nil {
		span.RecordError(err)
		return err
	}

	// Adjust confidence based on feedback
	switch req.Rating {
	case RatingHelpful:
		rem.Confidence = min(rem.Confidence+s.config.FeedbackDelta, s.config.MaxConfidence)
	case RatingNotHelpful:
		rem.Confidence = max(rem.Confidence-s.config.FeedbackDelta, s.config.MinConfidence)
	case RatingOutdated:
		rem.Confidence = max(rem.Confidence-s.config.FeedbackDelta*2, s.config.MinConfidence)
	}

	rem.UsageCount++
	rem.UpdatedAt = time.Now()

	// Update in storage
	collection := s.collectionName(req.TenantID, rem.Scope, rem.TeamID, rem.ProjectPath)
	point := &QdrantPoint{
		ID:      rem.ID,
		Vector:  rem.Vector,
		Payload: remediationToPayload(rem),
	}

	if err := s.qdrant.Upsert(ctx, collection, []*QdrantPoint{point}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to update remediation: %w", err)
	}

	// Record metrics
	if s.feedbackCounter != nil {
		s.feedbackCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("rating", string(req.Rating)),
		))
	}

	s.logger.Info("recorded feedback",
		zap.String("remediation_id", req.RemediationID),
		zap.String("rating", string(req.Rating)),
		zap.Float64("new_confidence", rem.Confidence),
	)

	return nil
}

// Delete removes a remediation.
func (s *service) Delete(ctx context.Context, tenantID, remediationID string) error {
	ctx, span := s.tracer.Start(ctx, "remediation.delete")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", tenantID),
		attribute.String("remediation_id", remediationID),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return errors.New("service is closed")
	}
	s.mu.RUnlock()

	// Try each scope level
	scopes := []Scope{ScopeProject, ScopeTeam, ScopeOrg}
	for _, scope := range scopes {
		collection := s.collectionName(tenantID, scope, "", "")
		exists, err := s.qdrant.CollectionExists(ctx, collection)
		if err != nil || !exists {
			continue
		}

		if err := s.qdrant.Delete(ctx, collection, []string{remediationID}); err == nil {
			s.logger.Info("deleted remediation", zap.String("id", remediationID))
			return nil
		}
	}

	return fmt.Errorf("remediation not found: %s", remediationID)
}

// Close closes the service.
func (s *service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	return nil
}

// Helper functions

func remediationToPayload(r *Remediation) map[string]interface{} {
	payload := map[string]interface{}{
		"title":        r.Title,
		"problem":      r.Problem,
		"root_cause":   r.RootCause,
		"solution":     r.Solution,
		"category":     string(r.Category),
		"confidence":   r.Confidence,
		"usage_count":  r.UsageCount,
		"scope":        string(r.Scope),
		"tenant_id":    r.TenantID,
		"team_id":      r.TeamID,
		"project_path": r.ProjectPath,
		"session_id":   r.SessionID,
		"created_at":   r.CreatedAt.Unix(),
		"updated_at":   r.UpdatedAt.Unix(),
	}

	if r.CodeDiff != "" {
		payload["code_diff"] = r.CodeDiff
	}

	// Note: Symptoms and AffectedFiles are slices, we'll store them as JSON strings or comma-separated
	// For now, store as comma-separated strings for simple Qdrant payload compatibility
	if len(r.Symptoms) > 0 {
		symptomsStr := ""
		for i, s := range r.Symptoms {
			if i > 0 {
				symptomsStr += "||"
			}
			symptomsStr += s
		}
		payload["symptoms"] = symptomsStr
	}

	if len(r.AffectedFiles) > 0 {
		filesStr := ""
		for i, f := range r.AffectedFiles {
			if i > 0 {
				filesStr += "||"
			}
			filesStr += f
		}
		payload["affected_files"] = filesStr
	}

	if len(r.Tags) > 0 {
		tagsStr := ""
		for i, t := range r.Tags {
			if i > 0 {
				tagsStr += "||"
			}
			tagsStr += t
		}
		payload["tags"] = tagsStr
	}

	return payload
}

func payloadToRemediation(payload map[string]interface{}) *Remediation {
	if payload == nil {
		return nil
	}

	r := &Remediation{}

	if v, ok := payload["title"].(string); ok {
		r.Title = v
	}
	if v, ok := payload["problem"].(string); ok {
		r.Problem = v
	}
	if v, ok := payload["root_cause"].(string); ok {
		r.RootCause = v
	}
	if v, ok := payload["solution"].(string); ok {
		r.Solution = v
	}
	if v, ok := payload["code_diff"].(string); ok {
		r.CodeDiff = v
	}
	if v, ok := payload["category"].(string); ok {
		r.Category = ErrorCategory(v)
	}
	if v, ok := payload["confidence"].(float64); ok {
		r.Confidence = v
	}
	if v, ok := payload["usage_count"].(int64); ok {
		r.UsageCount = v
	}
	if v, ok := payload["scope"].(string); ok {
		r.Scope = Scope(v)
	}
	if v, ok := payload["tenant_id"].(string); ok {
		r.TenantID = v
	}
	if v, ok := payload["team_id"].(string); ok {
		r.TeamID = v
	}
	if v, ok := payload["project_path"].(string); ok {
		r.ProjectPath = v
	}
	if v, ok := payload["session_id"].(string); ok {
		r.SessionID = v
	}
	if v, ok := payload["created_at"].(int64); ok {
		r.CreatedAt = time.Unix(v, 0)
	}
	if v, ok := payload["updated_at"].(int64); ok {
		r.UpdatedAt = time.Unix(v, 0)
	}

	// Parse symptoms from comma-separated string
	if v, ok := payload["symptoms"].(string); ok && v != "" {
		r.Symptoms = []string{}
		for _, s := range splitByDelimiter(v, "||") {
			if s != "" {
				r.Symptoms = append(r.Symptoms, s)
			}
		}
	}

	// Parse affected_files from comma-separated string
	if v, ok := payload["affected_files"].(string); ok && v != "" {
		r.AffectedFiles = []string{}
		for _, f := range splitByDelimiter(v, "||") {
			if f != "" {
				r.AffectedFiles = append(r.AffectedFiles, f)
			}
		}
	}

	// Parse tags from comma-separated string
	if v, ok := payload["tags"].(string); ok && v != "" {
		r.Tags = []string{}
		for _, t := range splitByDelimiter(v, "||") {
			if t != "" {
				r.Tags = append(r.Tags, t)
			}
		}
	}

	return r
}

func splitByDelimiter(s, delimiter string) []string {
	result := []string{}
	current := ""
	delimLen := len(delimiter)

	for i := 0; i < len(s); i++ {
		if i+delimLen <= len(s) && s[i:i+delimLen] == delimiter {
			result = append(result, current)
			current = ""
			i += delimLen - 1
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func sortAndLimit(remediations []*ScoredRemediation, limit int) []*ScoredRemediation {
	sort.Slice(remediations, func(i, j int) bool {
		return remediations[i].Score > remediations[j].Score
	})

	if len(remediations) > limit {
		return remediations[:limit]
	}
	return remediations
}

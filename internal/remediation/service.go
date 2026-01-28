package remediation

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/vectorstore"
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
	config *Config
	store  vectorstore.Store         // Legacy single-store mode
	stores vectorstore.StoreProvider // Database-per-project isolation mode
	logger *zap.Logger

	// Telemetry
	tracer           trace.Tracer
	meter            metric.Meter
	searchCounter    metric.Int64Counter
	searchDuration   metric.Float64Histogram
	recordCounter    metric.Int64Counter
	feedbackCounter  metric.Int64Counter
	errorCounter     metric.Int64Counter

	mu     sync.RWMutex
	closed bool
}

// NewService creates a new remediation service.
func NewService(cfg *Config, store vectorstore.Store, logger *zap.Logger) (Service, error) {
	if cfg == nil {
		cfg = DefaultServiceConfig()
	}
	if store == nil {
		return nil, errors.New("vector store is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required for remediation service")
	}

	s := &service{
		config: cfg,
		store:  store,
		logger: logger,
		tracer: otel.Tracer(instrumentationName),
		meter:  otel.Meter(instrumentationName),
	}

	s.initMetrics()

	return s, nil
}

// NewServiceWithStoreProvider creates a remediation service using StoreProvider
// for database-per-scope isolation.
//
// With StoreProvider, each scope level (org, team, project) gets its own
// chromem.DB instance at a unique filesystem path, providing physical isolation.
//
// The collection naming within each store is simplified to just "remediations"
// since isolation is handled at the store/directory level.
func NewServiceWithStoreProvider(cfg *Config, stores vectorstore.StoreProvider, logger *zap.Logger) (Service, error) {
	if cfg == nil {
		cfg = DefaultServiceConfig()
	}
	if stores == nil {
		return nil, errors.New("store provider is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required for remediation service")
	}

	s := &service{
		config: cfg,
		stores: stores,
		logger: logger,
		tracer: otel.Tracer(instrumentationName),
		meter:  otel.Meter(instrumentationName),
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

	s.searchDuration, err = s.meter.Float64Histogram(
		"contextd.remediation.search_duration_seconds",
		metric.WithDescription("Duration of remediation searches"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0),
	)
	if err != nil {
		s.logger.Warn("failed to create search duration histogram", zap.Error(err))
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

	s.errorCounter, err = s.meter.Int64Counter(
		"contextd.remediation.errors_total",
		metric.WithDescription("Total number of remediation errors by operation"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		s.logger.Warn("failed to create error counter", zap.Error(err))
	}
}

// recordError records an error metric with operation and reason labels.
func (s *service) recordError(ctx context.Context, operation, reason string) {
	if s.errorCounter != nil {
		s.errorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("reason", reason),
		))
	}
}

// getStore returns the appropriate store and collection name based on scope.
// With StoreProvider, each scope level gets its own chromem.DB instance.
// With legacy Store, collection names include scope prefixes.
func (s *service) getStore(ctx context.Context, tenantID string, scope Scope, teamID, projectPath string) (vectorstore.Store, string, error) {
	if s.stores != nil {
		// Use StoreProvider for database-per-scope isolation
		var store vectorstore.Store
		var err error

		switch scope {
		case ScopeProject:
			store, err = s.stores.GetProjectStore(ctx, tenantID, teamID, projectPath)
		case ScopeTeam:
			store, err = s.stores.GetTeamStore(ctx, tenantID, teamID)
		case ScopeOrg:
			store, err = s.stores.GetOrgStore(ctx, tenantID)
		default:
			store, err = s.stores.GetOrgStore(ctx, tenantID)
		}

		if err != nil {
			return nil, "", fmt.Errorf("getting store for scope %s: %w", scope, err)
		}

		// With StoreProvider, use simple collection name (isolation is at store level)
		return store, s.config.CollectionPrefix, nil
	}

	// Legacy: single store with prefixed collection names
	if s.store == nil {
		return nil, "", errors.New("no store configured")
	}
	return s.store, s.collectionName(tenantID, scope, teamID, projectPath), nil
}

// collectionName returns the collection name for a given tenant and scope.
// Used in legacy single-store mode.
func (s *service) collectionName(tenantID string, scope Scope, teamID, projectPath string) string {
	// Sanitize tenant and team IDs to ensure valid collection names
	sanitizedTenant := sanitizePath(tenantID)
	sanitizedTeam := sanitizePath(teamID)

	switch scope {
	case ScopeOrg:
		return fmt.Sprintf("%s_org_%s", s.config.CollectionPrefix, sanitizedTenant)
	case ScopeTeam:
		return fmt.Sprintf("%s_team_%s_%s", s.config.CollectionPrefix, sanitizedTenant, sanitizedTeam)
	case ScopeProject:
		return fmt.Sprintf("%s_project_%s_%s", s.config.CollectionPrefix, sanitizedTenant, sanitizePath(projectPath))
	default:
		return fmt.Sprintf("%s_org_%s", s.config.CollectionPrefix, sanitizedTenant)
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
	start := time.Now()
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

	if req.Query == "" {
		return nil, errors.New("query is required")
	}

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	// Build metadata filters (excludes confidence - that's post-filtered)
	filters := s.buildSearchFilters(req)

	// Determine which scopes to search
	scopes := s.getSearchScopes(req)

	// Fetch extra results to account for confidence post-filtering
	// Use 3x multiplier to ensure enough results after filtering, with bounds
	searchLimit := limit * 3
	if searchLimit < 30 {
		searchLimit = 30
	}
	if searchLimit > 200 {
		searchLimit = 200 // Cap to prevent excessive fetching
	}

	var allResults []*ScoredRemediation
	var storesAccessed int
	var lastStoreErr error

	for _, scopeInfo := range scopes {
		// Get store for this scope
		store, collection, err := s.getStore(ctx, req.TenantID, scopeInfo.scope, scopeInfo.teamID, scopeInfo.projectPath)
		if err != nil {
			s.logger.Warn("failed to get store", zap.String("scope", string(scopeInfo.scope)), zap.Error(err))
			lastStoreErr = err
			continue
		}

		// Inject tenant context for payload-based isolation
		scopedCtx := vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
			TenantID:  req.TenantID,
			TeamID:    scopeInfo.teamID,
			ProjectID: scopeInfo.projectPath,
		})

		// Check if collection exists
		exists, err := store.CollectionExists(scopedCtx, collection)
		if err != nil {
			s.logger.Warn("failed to check collection", zap.String("collection", collection), zap.Error(err))
			lastStoreErr = err
			continue
		}
		if !exists {
			// Collection doesn't exist yet - not an error, just no data
			// Note: We intentionally don't count this as "accessed" for error tracking
			// because we want to know if ANY data was successfully retrieved
			continue
		}

		results, err := store.SearchInCollection(scopedCtx, collection, req.Query, searchLimit, filters)
		if err != nil {
			s.logger.Warn("search failed", zap.String("collection", collection), zap.Error(err))
			lastStoreErr = err
			continue
		}

		// Only increment after successful search (with or without results)
		storesAccessed++

		for _, r := range results {
			rem := s.resultToRemediation(r)
			if rem == nil {
				continue
			}

			// Post-filter: skip remediations below confidence threshold
			if req.MinConfidence > 0 && rem.Confidence < req.MinConfidence {
				s.logger.Debug("skipping low-confidence remediation",
					zap.String("id", rem.ID),
					zap.Float64("confidence", rem.Confidence),
					zap.Float64("min_confidence", req.MinConfidence))
				continue
			}

			// Post-filter: skip remediations that don't match any requested tags
			if len(req.Tags) > 0 {
				matchesTag := false
				for _, reqTag := range req.Tags {
					for _, remTag := range rem.Tags {
						if reqTag == remTag {
							matchesTag = true
							break
						}
					}
					if matchesTag {
						break
					}
				}
				if !matchesTag {
					s.logger.Debug("skipping remediation without matching tags",
						zap.String("id", rem.ID),
						zap.Strings("required_tags", req.Tags),
						zap.Strings("remediation_tags", rem.Tags))
					continue
				}
			}

			allResults = append(allResults, &ScoredRemediation{
				Remediation: *rem,
				Score:       float64(r.Score),
			})
		}
	}

	// If no stores were accessible, return error instead of empty results
	if storesAccessed == 0 && lastStoreErr != nil {
		s.recordError(ctx, "search", "no_stores_accessible")
		return nil, fmt.Errorf("failed to access any stores: %w", lastStoreErr)
	}

	// Sort by score and limit
	allResults = sortAndLimit(allResults, limit)

	// Record metrics
	duration := time.Since(start)
	if s.searchCounter != nil {
		s.searchCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("scope", string(req.Scope)),
			attribute.String("project_id", req.ProjectPath),
			attribute.Int("result_count", len(allResults)),
		))
	}
	if s.searchDuration != nil {
		s.searchDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
			attribute.String("scope", string(req.Scope)),
		))
	}

	span.SetAttributes(attribute.Int("result_count", len(allResults)))
	return allResults, nil
}

// scopeInfo holds scope information for searching.
type scopeInfo struct {
	scope       Scope
	teamID      string
	projectPath string
}

// getSearchScopes returns the scopes to search based on request and hierarchy.
// This replaces getSearchCollections to work with StoreProvider.
func (s *service) getSearchScopes(req *SearchRequest) []scopeInfo {
	var scopes []scopeInfo

	switch req.Scope {
	case ScopeProject:
		scopes = append(scopes, scopeInfo{scope: ScopeProject, teamID: req.TeamID, projectPath: req.ProjectPath})
		if req.IncludeHierarchy {
			scopes = append(scopes, scopeInfo{scope: ScopeTeam, teamID: req.TeamID})
			scopes = append(scopes, scopeInfo{scope: ScopeOrg})
		}
	case ScopeTeam:
		scopes = append(scopes, scopeInfo{scope: ScopeTeam, teamID: req.TeamID})
		if req.IncludeHierarchy {
			scopes = append(scopes, scopeInfo{scope: ScopeOrg})
		}
	case ScopeOrg:
		scopes = append(scopes, scopeInfo{scope: ScopeOrg})
	default:
		// Search all applicable scopes if not specified
		if req.ProjectPath != "" {
			scopes = append(scopes, scopeInfo{scope: ScopeProject, teamID: req.TeamID, projectPath: req.ProjectPath})
		}
		if req.TeamID != "" {
			scopes = append(scopes, scopeInfo{scope: ScopeTeam, teamID: req.TeamID})
		}
		scopes = append(scopes, scopeInfo{scope: ScopeOrg})
	}

	return scopes
}

// buildSearchFilters creates metadata filters from search request.
// NOTE: Confidence filtering is done post-search in the service layer to remain
// store-agnostic (not all vectorstores support $gte operators).
func (s *service) buildSearchFilters(req *SearchRequest) map[string]interface{} {
	filters := make(map[string]interface{})

	// Category filter (exact match - supported by all stores)
	if req.Category != "" {
		filters["category"] = string(req.Category)
	}

	if len(filters) == 0 {
		return nil
	}

	return filters
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
	}

	// Get store and collection name
	store, collection, err := s.getStore(ctx, req.TenantID, req.Scope, req.TeamID, req.ProjectPath)
	if err != nil {
		span.RecordError(err)
		s.recordError(ctx, "record", "get_store_failed")
		return nil, err
	}

	// Inject tenant context for payload-based isolation
	ctx = vectorstore.ContextWithTenant(ctx, &vectorstore.TenantInfo{
		TenantID:  req.TenantID,
		TeamID:    req.TeamID,
		ProjectID: req.ProjectPath,
	})

	// Ensure collection exists
	exists, err := store.CollectionExists(ctx, collection)
	if err != nil {
		span.RecordError(err)
		s.recordError(ctx, "record", "check_collection_failed")
		return nil, fmt.Errorf("failed to check collection: %w", err)
	}
	if !exists {
		// Use store's configured vector size (0 = use default from embedder)
		if err := store.CreateCollection(ctx, collection, 0); err != nil {
			span.RecordError(err)
			s.recordError(ctx, "record", "create_collection_failed")
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	// Convert to document for storage (Store handles embedding internally)
	doc := s.remediationToDocument(rem, collection)

	if _, err := store.AddDocuments(ctx, []vectorstore.Document{doc}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.recordError(ctx, "record", "store_failed")
		return nil, fmt.Errorf("failed to store remediation: %w", err)
	}

	// Record metrics
	if s.recordCounter != nil {
		s.recordCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("scope", string(req.Scope)),
			attribute.String("project_id", req.ProjectPath),
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
//
// Note: This method searches org-level scope only. For project or team-scoped
// remediations, use GetByScope which accepts full scope parameters.
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

	// With StoreProvider, we can only search org scope without teamID/projectPath
	// For legacy store, try all scopes
	if s.stores != nil {
		// StoreProvider mode: only search org scope
		store, collection, err := s.getStore(ctx, tenantID, ScopeOrg, "", "")
		if err != nil {
			return nil, err
		}

		exists, err := store.CollectionExists(ctx, collection)
		if err != nil || !exists {
			return nil, fmt.Errorf("remediation not found: %s", remediationID)
		}

		filters := map[string]interface{}{
			"id": remediationID,
		}

		results, err := store.SearchInCollection(ctx, collection, "remediation", 1, filters)
		if err != nil {
			return nil, fmt.Errorf("searching for remediation: %w", err)
		}
		if len(results) > 0 {
			rem := s.resultToRemediation(results[0])
			if rem != nil {
				return rem, nil
			}
		}
		return nil, fmt.Errorf("remediation not found: %s", remediationID)
	}

	// Legacy mode: try each scope level (can enumerate collections)
	scopes := []Scope{ScopeProject, ScopeTeam, ScopeOrg}
	for _, scope := range scopes {
		collection := s.collectionName(tenantID, scope, "", "")
		exists, err := s.store.CollectionExists(ctx, collection)
		if err != nil || !exists {
			continue
		}

		// Search by ID using filter
		filters := map[string]interface{}{
			"id": remediationID,
		}

		results, err := s.store.SearchInCollection(ctx, collection, "remediation", 1, filters)
		if err != nil {
			continue
		}
		if len(results) > 0 {
			rem := s.resultToRemediation(results[0])
			if rem != nil {
				return rem, nil
			}
		}
	}

	return nil, fmt.Errorf("remediation not found: %s", remediationID)
}

// GetByScope retrieves a remediation by ID within a specific scope.
// This method works with both legacy Store and StoreProvider modes.
func (s *service) GetByScope(ctx context.Context, tenantID, remediationID string, scope Scope, teamID, projectPath string) (*Remediation, error) {
	ctx, span := s.tracer.Start(ctx, "remediation.get_by_scope")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", tenantID),
		attribute.String("remediation_id", remediationID),
		attribute.String("scope", string(scope)),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, errors.New("service is closed")
	}
	s.mu.RUnlock()

	store, collection, err := s.getStore(ctx, tenantID, scope, teamID, projectPath)
	if err != nil {
		return nil, err
	}

	exists, err := store.CollectionExists(ctx, collection)
	if err != nil || !exists {
		return nil, fmt.Errorf("remediation not found: %s", remediationID)
	}

	filters := map[string]interface{}{
		"id": remediationID,
	}

	results, err := store.SearchInCollection(ctx, collection, "remediation", 1, filters)
	if err != nil {
		return nil, fmt.Errorf("searching for remediation: %w", err)
	}
	if len(results) > 0 {
		rem := s.resultToRemediation(results[0])
		if rem != nil {
			return rem, nil
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
		s.recordError(ctx, "feedback", "get_remediation_failed")
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

	// Get store for the remediation's scope
	store, collection, err := s.getStore(ctx, req.TenantID, rem.Scope, rem.TeamID, rem.ProjectPath)
	if err != nil {
		span.RecordError(err)
		s.recordError(ctx, "feedback", "get_store_failed")
		return err
	}

	// Delete old version
	if err := store.DeleteDocumentsFromCollection(ctx, collection, []string{rem.ID}); err != nil {
		span.RecordError(err)
		s.recordError(ctx, "feedback", "delete_old_failed")
		return fmt.Errorf("failed to delete old remediation: %w", err)
	}

	// Re-add with updated confidence
	doc := s.remediationToDocument(rem, collection)
	if _, err := store.AddDocuments(ctx, []vectorstore.Document{doc}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.recordError(ctx, "feedback", "update_failed")
		return fmt.Errorf("failed to update remediation: %w", err)
	}

	// Record metrics
	if s.feedbackCounter != nil {
		s.feedbackCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("rating", string(req.Rating)),
			attribute.String("project_id", rem.ProjectPath),
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
//
// Note: This method searches org-level scope only. For project or team-scoped
// remediations, use DeleteByScope which accepts full scope parameters.
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

	// With StoreProvider, we can only search org scope without teamID/projectPath
	if s.stores != nil {
		store, collection, err := s.getStore(ctx, tenantID, ScopeOrg, "", "")
		if err != nil {
			return err
		}

		exists, err := store.CollectionExists(ctx, collection)
		if err != nil || !exists {
			return fmt.Errorf("remediation not found: %s", remediationID)
		}

		if err := store.DeleteDocumentsFromCollection(ctx, collection, []string{remediationID}); err == nil {
			s.logger.Info("deleted remediation", zap.String("id", remediationID))
			return nil
		}
		return fmt.Errorf("remediation not found: %s", remediationID)
	}

	// Legacy mode: try each scope level
	scopes := []Scope{ScopeProject, ScopeTeam, ScopeOrg}
	for _, scope := range scopes {
		collection := s.collectionName(tenantID, scope, "", "")
		exists, err := s.store.CollectionExists(ctx, collection)
		if err != nil || !exists {
			continue
		}

		if err := s.store.DeleteDocumentsFromCollection(ctx, collection, []string{remediationID}); err == nil {
			s.logger.Info("deleted remediation", zap.String("id", remediationID))
			return nil
		}
	}

	return fmt.Errorf("remediation not found: %s", remediationID)
}

// DeleteByScope removes a remediation from a specific scope.
// This method works with both legacy Store and StoreProvider modes.
func (s *service) DeleteByScope(ctx context.Context, tenantID, remediationID string, scope Scope, teamID, projectPath string) error {
	ctx, span := s.tracer.Start(ctx, "remediation.delete_by_scope")
	defer span.End()

	span.SetAttributes(
		attribute.String("tenant_id", tenantID),
		attribute.String("remediation_id", remediationID),
		attribute.String("scope", string(scope)),
	)

	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return errors.New("service is closed")
	}
	s.mu.RUnlock()

	store, collection, err := s.getStore(ctx, tenantID, scope, teamID, projectPath)
	if err != nil {
		return err
	}

	exists, err := store.CollectionExists(ctx, collection)
	if err != nil || !exists {
		return fmt.Errorf("remediation not found: %s", remediationID)
	}

	if err := store.DeleteDocumentsFromCollection(ctx, collection, []string{remediationID}); err != nil {
		return fmt.Errorf("failed to delete remediation: %w", err)
	}

	s.logger.Info("deleted remediation", zap.String("id", remediationID), zap.String("scope", string(scope)))
	return nil
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

// remediationToDocument converts a Remediation to a vectorstore Document.
func (s *service) remediationToDocument(r *Remediation, collectionName string) vectorstore.Document {
	// Combine title, problem, symptoms, and root_cause for embedding
	content := fmt.Sprintf("%s\n\n%s\n\n%s", r.Title, r.Problem, r.RootCause)
	if len(r.Symptoms) > 0 {
		content += "\n\nSymptoms: " + joinStrings(r.Symptoms, ", ")
	}

	metadata := map[string]interface{}{
		"id":           r.ID,
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
		metadata["code_diff"] = r.CodeDiff
	}

	if len(r.Symptoms) > 0 {
		metadata["symptoms"] = joinStrings(r.Symptoms, "||")
	}

	if len(r.AffectedFiles) > 0 {
		metadata["affected_files"] = joinStrings(r.AffectedFiles, "||")
	}

	if len(r.Tags) > 0 {
		metadata["tags"] = joinStrings(r.Tags, "||")
	}

	return vectorstore.Document{
		ID:         r.ID,
		Content:    content,
		Metadata:   metadata,
		Collection: collectionName,
	}
}

// resultToRemediation converts a vectorstore SearchResult to a Remediation.
func (s *service) resultToRemediation(result vectorstore.SearchResult) *Remediation {
	if result.Metadata == nil {
		return nil
	}

	r := &Remediation{}

	// Extract ID
	if id, ok := result.Metadata["id"].(string); ok {
		r.ID = id
	} else {
		r.ID = result.ID
	}

	if v, ok := result.Metadata["title"].(string); ok {
		r.Title = v
	}
	if v, ok := result.Metadata["problem"].(string); ok {
		r.Problem = v
	}
	if v, ok := result.Metadata["root_cause"].(string); ok {
		r.RootCause = v
	}
	if v, ok := result.Metadata["solution"].(string); ok {
		r.Solution = v
	}
	if v, ok := result.Metadata["code_diff"].(string); ok {
		r.CodeDiff = v
	}
	if v, ok := result.Metadata["category"].(string); ok {
		r.Category = ErrorCategory(v)
	}
	if v, ok := result.Metadata["confidence"].(float64); ok {
		r.Confidence = v
	}
	if v, ok := result.Metadata["usage_count"].(int64); ok {
		r.UsageCount = v
	} else if v, ok := result.Metadata["usage_count"].(float64); ok {
		r.UsageCount = int64(v)
	}
	if v, ok := result.Metadata["scope"].(string); ok {
		r.Scope = Scope(v)
	}
	if v, ok := result.Metadata["tenant_id"].(string); ok {
		r.TenantID = v
	}
	if v, ok := result.Metadata["team_id"].(string); ok {
		r.TeamID = v
	}
	if v, ok := result.Metadata["project_path"].(string); ok {
		r.ProjectPath = v
	}
	if v, ok := result.Metadata["session_id"].(string); ok {
		r.SessionID = v
	}
	if v, ok := result.Metadata["created_at"].(int64); ok {
		r.CreatedAt = time.Unix(v, 0)
	} else if v, ok := result.Metadata["created_at"].(float64); ok {
		r.CreatedAt = time.Unix(int64(v), 0)
	}
	if v, ok := result.Metadata["updated_at"].(int64); ok {
		r.UpdatedAt = time.Unix(v, 0)
	} else if v, ok := result.Metadata["updated_at"].(float64); ok {
		r.UpdatedAt = time.Unix(int64(v), 0)
	}

	// Parse symptoms
	if v, ok := result.Metadata["symptoms"].(string); ok && v != "" {
		r.Symptoms = splitByDelimiter(v, "||")
	}

	// Parse affected_files
	if v, ok := result.Metadata["affected_files"].(string); ok && v != "" {
		r.AffectedFiles = splitByDelimiter(v, "||")
	}

	// Parse tags
	if v, ok := result.Metadata["tags"].(string); ok && v != "" {
		r.Tags = splitByDelimiter(v, "||")
	}

	return r
}

// joinStrings joins strings with a delimiter.
func joinStrings(strs []string, delimiter string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += delimiter
		}
		result += s
	}
	return result
}

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

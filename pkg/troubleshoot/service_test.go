package troubleshoot

import (
	"context"
	"errors"
	"testing"

	"github.com/fyrsmithlabs/contextd/pkg/vectorstore"
	"go.uber.org/zap"
)

// mockVectorStore is a mock implementation of VectorStore for testing
type mockVectorStore struct {
	addDocumentsFunc      func(ctx context.Context, docs []vectorstore.Document) error
	searchWithFiltersFunc func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error)
}

func (m *mockVectorStore) AddDocuments(ctx context.Context, docs []vectorstore.Document) error {
	if m.addDocumentsFunc != nil {
		return m.addDocumentsFunc(ctx, docs)
	}
	return nil
}

func (m *mockVectorStore) SearchWithFilters(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	if m.searchWithFiltersFunc != nil {
		return m.searchWithFiltersFunc(ctx, query, k, filters)
	}
	return nil, nil
}

// mockAIClient is a mock OpenAI client for testing
type mockAIClient struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *mockAIClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	return "", nil
}

func TestNewService(t *testing.T) {
	store := &mockVectorStore{}
	logger := zap.NewNop()

	svc := NewService(store, logger, nil)

	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
	if svc.store != store {
		t.Error("Service store not set correctly")
	}
	if svc.logger != logger {
		t.Error("Service logger not set correctly")
	}
}

func TestNewService_NilLogger(t *testing.T) {
	store := &mockVectorStore{}

	svc := NewService(store, nil, nil)

	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
	if svc.logger == nil {
		t.Error("Service should have created a no-op logger")
	}
}

func TestSavePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern *Pattern
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pattern",
			pattern: &Pattern{
				ErrorType:   "ConnectionError",
				Description: "Connection refused to Qdrant",
				Solution:    "Start Qdrant with docker-compose up -d",
				Confidence:  0.95,
			},
			wantErr: false,
		},
		{
			name:    "nil pattern",
			pattern: nil,
			wantErr: true,
			errMsg:  "pattern cannot be nil",
		},
		{
			name: "empty error type",
			pattern: &Pattern{
				Description: "Some description",
				Solution:    "Some solution",
			},
			wantErr: true,
			errMsg:  "error type is required",
		},
		{
			name: "empty description",
			pattern: &Pattern{
				ErrorType: "ConnectionError",
				Solution:  "Some solution",
			},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name: "empty solution",
			pattern: &Pattern{
				ErrorType:   "ConnectionError",
				Description: "Some description",
			},
			wantErr: true,
			errMsg:  "solution is required",
		},
		{
			name: "invalid confidence (too low)",
			pattern: &Pattern{
				ErrorType:   "ConnectionError",
				Description: "Some description",
				Solution:    "Some solution",
				Confidence:  -0.1,
			},
			wantErr: true,
			errMsg:  "confidence must be between 0.0 and 1.0",
		},
		{
			name: "invalid confidence (too high)",
			pattern: &Pattern{
				ErrorType:   "ConnectionError",
				Description: "Some description",
				Solution:    "Some solution",
				Confidence:  1.1,
			},
			wantErr: true,
			errMsg:  "confidence must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockVectorStore{}
			svc := NewService(store, zap.NewNop(), nil)

			err := svc.SavePattern(context.Background(), tt.pattern)

			if tt.wantErr && err == nil {
				t.Errorf("SavePattern() error = nil, wantErr %v", tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SavePattern() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if err.Error() != tt.errMsg {
					t.Errorf("SavePattern() error = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestSavePattern_StoreError(t *testing.T) {
	store := &mockVectorStore{
		addDocumentsFunc: func(ctx context.Context, docs []vectorstore.Document) error {
			return errors.New("store error")
		},
	}
	svc := NewService(store, zap.NewNop(), nil)

	pattern := &Pattern{
		ErrorType:   "ConnectionError",
		Description: "Test",
		Solution:    "Test solution",
		Confidence:  0.9,
	}

	err := svc.SavePattern(context.Background(), pattern)
	if err == nil {
		t.Fatal("SavePattern() should return error when store fails")
	}
	if err.Error() != "failed to store pattern: store error" {
		t.Errorf("SavePattern() error = %q, want %q", err.Error(), "failed to store pattern: store error")
	}
}

func TestGetPatterns(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockVectorStore)
		want    int
		wantErr bool
	}{
		{
			name: "successful retrieval",
			setup: func(m *mockVectorStore) {
				m.searchWithFiltersFunc = func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					return []vectorstore.SearchResult{
						{
							ID:      "pattern1",
							Content: "Connection refused",
							Metadata: map[string]interface{}{
								"error_type":  "ConnectionError",
								"description": "Connection refused to Qdrant",
								"solution":    "Start Qdrant",
								"confidence":  0.95,
								"frequency":   10,
								"created_at":  "2025-11-19T10:00:00Z",
							},
						},
					}, nil
				}
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "store error",
			setup: func(m *mockVectorStore) {
				m.searchWithFiltersFunc = func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					return nil, errors.New("search failed")
				}
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "empty results",
			setup: func(m *mockVectorStore) {
				m.searchWithFiltersFunc = func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					return []vectorstore.SearchResult{}, nil
				}
			},
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockVectorStore{}
			tt.setup(store)
			svc := NewService(store, zap.NewNop(), nil)

			patterns, err := svc.GetPatterns(context.Background())

			if tt.wantErr && err == nil {
				t.Error("GetPatterns() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("GetPatterns() error = %v, want nil", err)
			}
			if len(patterns) != tt.want {
				t.Errorf("GetPatterns() returned %d patterns, want %d", len(patterns), tt.want)
			}
		})
	}
}

func TestDiagnose(t *testing.T) {
	tests := []struct {
		name         string
		errorMsg     string
		errorContext string
		setupStore   func(*mockVectorStore)
		setupAI      func(*mockAIClient)
		wantErr      bool
		checkResult  func(*testing.T, *Diagnosis)
	}{
		{
			name:         "empty error message",
			errorMsg:     "",
			errorContext: "some context",
			setupStore:   func(m *mockVectorStore) {},
			setupAI:      func(m *mockAIClient) {},
			wantErr:      true,
		},
		{
			name:         "high confidence pattern match",
			errorMsg:     "connection refused port 6333",
			errorContext: "",
			setupStore: func(m *mockVectorStore) {
				m.searchWithFiltersFunc = func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					return []vectorstore.SearchResult{
						{
							ID:      "pattern1",
							Content: "connection refused",
							Score:   0.95,
							Metadata: map[string]interface{}{
								"error_type":  "ConnectionError",
								"description": "Connection refused to Qdrant",
								"solution":    "Start Qdrant with docker-compose up -d",
								"confidence":  0.95,
								"frequency":   10,
								"created_at":  "2025-11-19T10:00:00Z",
							},
						},
					}, nil
				}
			},
			setupAI: func(m *mockAIClient) {
				// Should not be called for high-confidence match
			},
			wantErr: false,
			checkResult: func(t *testing.T, d *Diagnosis) {
				if len(d.RelatedPatterns) == 0 {
					t.Error("Diagnosis should have related patterns")
				}
				if d.RootCause == "" {
					t.Error("Diagnosis should have root cause from pattern")
				}
			},
		},
		{
			name:         "low confidence requires AI",
			errorMsg:     "unexpected error xyz",
			errorContext: "during startup",
			setupStore: func(m *mockVectorStore) {
				m.searchWithFiltersFunc = func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					return []vectorstore.SearchResult{
						{
							ID:    "pattern1",
							Score: 0.4, // Low confidence
							Metadata: map[string]interface{}{
								"error_type":  "GenericError",
								"description": "Generic error",
								"solution":    "Check logs",
								"confidence":  0.4,
								"frequency":   1,
								"created_at":  "2025-11-19T10:00:00Z",
							},
						},
					}, nil
				}
			},
			setupAI: func(m *mockAIClient) {
				m.generateFunc = func(ctx context.Context, prompt string) (string, error) {
					return `{
						"root_cause": "Configuration issue",
						"hypotheses": [
							{
								"description": "Missing environment variable",
								"likelihood": 0.8,
								"evidence": "Startup context suggests config problem"
							}
						],
						"recommendations": ["Check environment variables", "Review config file"]
					}`, nil
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, d *Diagnosis) {
				if len(d.Hypotheses) == 0 {
					t.Error("Diagnosis should have AI-generated hypotheses")
				}
				if d.RootCause == "" {
					t.Error("Diagnosis should have root cause from AI")
				}
			},
		},
		{
			name:         "AI failure with pattern fallback",
			errorMsg:     "some error",
			errorContext: "",
			setupStore: func(m *mockVectorStore) {
				m.searchWithFiltersFunc = func(ctx context.Context, query string, k int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
					return []vectorstore.SearchResult{
						{
							ID:    "pattern1",
							Score: 0.6,
							Metadata: map[string]interface{}{
								"error_type":  "GenericError",
								"description": "Generic error",
								"solution":    "Check logs",
								"confidence":  0.6,
								"frequency":   1,
								"created_at":  "2025-11-19T10:00:00Z",
							},
						},
					}, nil
				}
			},
			setupAI: func(m *mockAIClient) {
				m.generateFunc = func(ctx context.Context, prompt string) (string, error) {
					return "", errors.New("AI service unavailable")
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, d *Diagnosis) {
				if len(d.RelatedPatterns) == 0 {
					t.Error("Diagnosis should fall back to pattern-based diagnosis")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockVectorStore{}
			ai := &mockAIClient{}
			tt.setupStore(store)
			tt.setupAI(ai)
			svc := NewService(store, zap.NewNop(), ai)

			diagnosis, err := svc.Diagnose(context.Background(), tt.errorMsg, tt.errorContext)

			if tt.wantErr && err == nil {
				t.Error("Diagnose() error = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Diagnose() error = %v, want nil", err)
			}
			if !tt.wantErr && diagnosis != nil && tt.checkResult != nil {
				tt.checkResult(t, diagnosis)
			}
		})
	}
}

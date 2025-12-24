package compression

import (
	"context"
	"testing"
	"time"
)

// TestNewExperiment tests experiment creation
func TestNewExperiment(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		algos   []Algorithm
		wantErr bool
	}{
		{
			name:    "valid two-variant experiment",
			id:      "exp-001",
			algos:   []Algorithm{AlgorithmExtractive, AlgorithmAbstractive},
			wantErr: false,
		},
		{
			name:    "valid three-variant experiment",
			id:      "exp-002",
			algos:   []Algorithm{AlgorithmExtractive, AlgorithmAbstractive, AlgorithmHybrid},
			wantErr: false,
		},
		{
			name:    "empty experiment ID",
			id:      "",
			algos:   []Algorithm{AlgorithmExtractive, AlgorithmAbstractive},
			wantErr: true,
		},
		{
			name:    "single variant (invalid)",
			id:      "exp-003",
			algos:   []Algorithm{AlgorithmExtractive},
			wantErr: true,
		},
		{
			name:    "no variants",
			id:      "exp-004",
			algos:   []Algorithm{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp, err := NewExperiment(tt.id, tt.algos)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewExperiment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if exp.ID != tt.id {
					t.Errorf("Experiment.ID = %v, want %v", exp.ID, tt.id)
				}
				if len(exp.Variants) != len(tt.algos) {
					t.Errorf("len(Experiment.Variants) = %v, want %v", len(exp.Variants), len(tt.algos))
				}
			}
		})
	}
}

// TestExperimentAssignVariant tests variant assignment
func TestExperimentAssignVariant(t *testing.T) {
	exp, err := NewExperiment("test-exp", []Algorithm{AlgorithmExtractive, AlgorithmAbstractive})
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	tests := []struct {
		name      string
		sessionID string
		wantErr   bool
	}{
		{
			name:      "valid session ID",
			sessionID: "session-123",
			wantErr:   false,
		},
		{
			name:      "empty session ID",
			sessionID: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variant, err := exp.AssignVariant(tt.sessionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssignVariant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify variant is one of the experiment variants
				found := false
				for _, v := range exp.Variants {
					if v.Algorithm == variant {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("AssignVariant() returned %v, which is not in experiment variants", variant)
				}
			}
		})
	}
}

// TestExperimentAssignVariant_Consistency tests that same session gets same variant
func TestExperimentAssignVariant_Consistency(t *testing.T) {
	exp, err := NewExperiment("test-exp", []Algorithm{AlgorithmExtractive, AlgorithmAbstractive})
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	sessionID := "consistent-session"
	variant1, err := exp.AssignVariant(sessionID)
	if err != nil {
		t.Fatalf("First assignment failed: %v", err)
	}

	// Assign again - should get same variant
	for i := 0; i < 10; i++ {
		variant, err := exp.AssignVariant(sessionID)
		if err != nil {
			t.Fatalf("Assignment %d failed: %v", i, err)
		}
		if variant != variant1 {
			t.Errorf("Assignment %d = %v, want %v (inconsistent)", i, variant, variant1)
		}
	}
}

// TestExperimentRecordOutcome tests recording compression outcomes
func TestExperimentRecordOutcome(t *testing.T) {
	exp, err := NewExperiment("test-exp", []Algorithm{AlgorithmExtractive, AlgorithmAbstractive})
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	tests := []struct {
		name    string
		outcome CompressionOutcome
		wantErr bool
	}{
		{
			name: "valid outcome",
			outcome: CompressionOutcome{
				SessionID:        "session-1",
				Algorithm:        AlgorithmExtractive,
				CompressionRatio: 2.5,
				QualityScore:     0.85,
				ProcessingTimeMs: 50,
				Success:          true,
				Timestamp:        time.Now(),
			},
			wantErr: false,
		},
		{
			name: "outcome with user feedback",
			outcome: CompressionOutcome{
				SessionID:        "session-2",
				Algorithm:        AlgorithmAbstractive,
				CompressionRatio: 3.0,
				QualityScore:     0.90,
				ProcessingTimeMs: 500,
				Success:          true,
				UserAccepted:     true,
				Timestamp:        time.Now(),
			},
			wantErr: false,
		},
		{
			name: "failed compression",
			outcome: CompressionOutcome{
				SessionID:    "session-3",
				Algorithm:    AlgorithmExtractive,
				Success:      false,
				ErrorMessage: "content too large",
				Timestamp:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty session ID",
			outcome: CompressionOutcome{
				Algorithm: AlgorithmExtractive,
				Success:   true,
			},
			wantErr: true,
		},
		{
			name: "algorithm not in experiment",
			outcome: CompressionOutcome{
				SessionID: "session-4",
				Algorithm: Algorithm("unknown"),
				Success:   true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exp.RecordOutcome(tt.outcome)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordOutcome() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExperimentGetMetrics tests metrics retrieval
func TestExperimentGetMetrics(t *testing.T) {
	exp, err := NewExperiment("test-exp", []Algorithm{AlgorithmExtractive, AlgorithmAbstractive})
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	// Record some outcomes
	outcomes := []CompressionOutcome{
		{SessionID: "s1", Algorithm: AlgorithmExtractive, CompressionRatio: 2.0, QualityScore: 0.80, ProcessingTimeMs: 50, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s2", Algorithm: AlgorithmExtractive, CompressionRatio: 2.5, QualityScore: 0.85, ProcessingTimeMs: 55, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s3", Algorithm: AlgorithmExtractive, CompressionRatio: 0, QualityScore: 0, ProcessingTimeMs: 0, Success: false, Timestamp: time.Now()},
		{SessionID: "s4", Algorithm: AlgorithmAbstractive, CompressionRatio: 3.0, QualityScore: 0.90, ProcessingTimeMs: 500, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s5", Algorithm: AlgorithmAbstractive, CompressionRatio: 3.5, QualityScore: 0.92, ProcessingTimeMs: 550, Success: true, UserAccepted: false, Timestamp: time.Now()},
	}

	for _, outcome := range outcomes {
		if err := exp.RecordOutcome(outcome); err != nil {
			t.Fatalf("Failed to record outcome: %v", err)
		}
	}

	metrics := exp.GetMetrics()

	// Verify metrics structure
	if len(metrics) != 2 {
		t.Errorf("GetMetrics() returned %d variants, want 2", len(metrics))
	}

	// Check extractive metrics
	extractiveMetrics, ok := metrics[AlgorithmExtractive]
	if !ok {
		t.Fatal("Missing extractive metrics")
	}
	if extractiveMetrics.TotalAttempts != 3 {
		t.Errorf("Extractive TotalAttempts = %d, want 3", extractiveMetrics.TotalAttempts)
	}
	if extractiveMetrics.SuccessCount != 2 {
		t.Errorf("Extractive SuccessCount = %d, want 2", extractiveMetrics.SuccessCount)
	}
	expectedSuccessRate := 2.0 / 3.0
	if extractiveMetrics.SuccessRate < expectedSuccessRate-0.01 || extractiveMetrics.SuccessRate > expectedSuccessRate+0.01 {
		t.Errorf("Extractive SuccessRate = %f, want ~%f", extractiveMetrics.SuccessRate, expectedSuccessRate)
	}

	// Check abstractive metrics
	abstractiveMetrics, ok := metrics[AlgorithmAbstractive]
	if !ok {
		t.Fatal("Missing abstractive metrics")
	}
	if abstractiveMetrics.TotalAttempts != 2 {
		t.Errorf("Abstractive TotalAttempts = %d, want 2", abstractiveMetrics.TotalAttempts)
	}
	if abstractiveMetrics.SuccessCount != 2 {
		t.Errorf("Abstractive SuccessCount = %d, want 2", abstractiveMetrics.SuccessCount)
	}
	if abstractiveMetrics.UserAcceptanceRate != 0.5 {
		t.Errorf("Abstractive UserAcceptanceRate = %f, want 0.5", abstractiveMetrics.UserAcceptanceRate)
	}
}

// TestGenerateComparisonReport tests report generation
func TestGenerateComparisonReport(t *testing.T) {
	exp, err := NewExperiment("test-exp", []Algorithm{AlgorithmExtractive, AlgorithmAbstractive})
	if err != nil {
		t.Fatalf("Failed to create experiment: %v", err)
	}

	// Record outcomes (need at least 5 per variant for winner determination)
	outcomes := []CompressionOutcome{
		// Extractive variant
		{SessionID: "s1", Algorithm: AlgorithmExtractive, CompressionRatio: 2.0, QualityScore: 0.80, ProcessingTimeMs: 50, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s2", Algorithm: AlgorithmExtractive, CompressionRatio: 2.2, QualityScore: 0.82, ProcessingTimeMs: 52, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s3", Algorithm: AlgorithmExtractive, CompressionRatio: 2.1, QualityScore: 0.81, ProcessingTimeMs: 51, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s4", Algorithm: AlgorithmExtractive, CompressionRatio: 2.3, QualityScore: 0.83, ProcessingTimeMs: 53, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s5", Algorithm: AlgorithmExtractive, CompressionRatio: 2.4, QualityScore: 0.84, ProcessingTimeMs: 54, Success: true, UserAccepted: true, Timestamp: time.Now()},
		// Abstractive variant
		{SessionID: "s6", Algorithm: AlgorithmAbstractive, CompressionRatio: 3.0, QualityScore: 0.90, ProcessingTimeMs: 500, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s7", Algorithm: AlgorithmAbstractive, CompressionRatio: 3.2, QualityScore: 0.91, ProcessingTimeMs: 520, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s8", Algorithm: AlgorithmAbstractive, CompressionRatio: 3.1, QualityScore: 0.89, ProcessingTimeMs: 510, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s9", Algorithm: AlgorithmAbstractive, CompressionRatio: 3.3, QualityScore: 0.92, ProcessingTimeMs: 530, Success: true, UserAccepted: true, Timestamp: time.Now()},
		{SessionID: "s10", Algorithm: AlgorithmAbstractive, CompressionRatio: 3.4, QualityScore: 0.93, ProcessingTimeMs: 540, Success: true, UserAccepted: true, Timestamp: time.Now()},
	}
	for _, outcome := range outcomes {
		exp.RecordOutcome(outcome)
	}

	report := exp.GenerateComparisonReport()

	if report.ExperimentID != exp.ID {
		t.Errorf("Report ExperimentID = %v, want %v", report.ExperimentID, exp.ID)
	}
	if len(report.VariantMetrics) != 2 {
		t.Errorf("Report has %d variants, want 2", len(report.VariantMetrics))
	}
	if report.Winner == nil {
		t.Error("Report Winner is nil, expected a winner")
	}
	if report.Recommendation == "" {
		t.Error("Report Recommendation is empty")
	}
}

// TestABTestManager tests the manager for multiple experiments
func TestABTestManager(t *testing.T) {
	ctx := context.Background()
	manager := NewABTestManager()

	// Create experiment
	exp, err := manager.CreateExperiment(ctx, "exp-1", []Algorithm{AlgorithmExtractive, AlgorithmAbstractive})
	if err != nil {
		t.Fatalf("CreateExperiment() failed: %v", err)
	}
	if exp.ID != "exp-1" {
		t.Errorf("Experiment ID = %v, want exp-1", exp.ID)
	}

	// Get experiment
	retrieved, err := manager.GetExperiment(ctx, "exp-1")
	if err != nil {
		t.Fatalf("GetExperiment() failed: %v", err)
	}
	if retrieved.ID != exp.ID {
		t.Errorf("Retrieved experiment ID = %v, want %v", retrieved.ID, exp.ID)
	}

	// List experiments
	experiments := manager.ListExperiments(ctx)
	if len(experiments) != 1 {
		t.Errorf("ListExperiments() returned %d experiments, want 1", len(experiments))
	}

	// Non-existent experiment
	_, err = manager.GetExperiment(ctx, "non-existent")
	if err == nil {
		t.Error("GetExperiment() should fail for non-existent experiment")
	}
}

// TestABTestManager_ExportToAnalytics tests analytics integration
func TestABTestManager_ExportToAnalytics(t *testing.T) {
	ctx := context.Background()
	manager := NewABTestManager()

	exp, err := manager.CreateExperiment(ctx, "exp-1", []Algorithm{AlgorithmExtractive, AlgorithmAbstractive})
	if err != nil {
		t.Fatalf("CreateExperiment() failed: %v", err)
	}

	// Record outcome
	outcome := CompressionOutcome{
		SessionID:        "session-1",
		Algorithm:        AlgorithmExtractive,
		CompressionRatio: 2.5,
		QualityScore:     0.85,
		ProcessingTimeMs: 50,
		Success:          true,
		Timestamp:        time.Now(),
	}
	if err := exp.RecordOutcome(outcome); err != nil {
		t.Fatalf("RecordOutcome() failed: %v", err)
	}

	// Export to analytics (without actual analytics service for now)
	metrics := manager.ExportMetrics(ctx, "exp-1")
	if metrics == nil {
		t.Error("ExportMetrics() returned nil")
	}
	if len(metrics) == 0 {
		t.Error("ExportMetrics() returned empty metrics")
	}
}

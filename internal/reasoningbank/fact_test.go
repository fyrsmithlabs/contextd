package reasoningbank

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFact(t *testing.T) {
	now := time.Now()
	projectID := "test-project"

	tests := []struct {
		name          string
		subject       string
		predicate     string
		object        string
		timestamp     time.Time
		confidence    float64
		provenance    string
		sourceID      string
		projectID     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid fact",
			subject:     "I",
			predicate:   "attended",
			object:      "meeting",
			timestamp:   now,
			confidence:  0.9,
			provenance:  "I attended the meeting yesterday",
			sourceID:    "mem-123",
			projectID:   projectID,
			expectError: false,
		},
		{
			name:          "empty subject",
			subject:       "",
			predicate:     "attended",
			object:        "meeting",
			timestamp:     now,
			confidence:    0.9,
			projectID:     projectID,
			expectError:   true,
			errorContains: "subject cannot be empty",
		},
		{
			name:          "empty predicate",
			subject:       "I",
			predicate:     "",
			object:        "meeting",
			timestamp:     now,
			confidence:    0.9,
			projectID:     projectID,
			expectError:   true,
			errorContains: "predicate cannot be empty",
		},
		{
			name:          "empty object",
			subject:       "I",
			predicate:     "attended",
			object:        "",
			timestamp:     now,
			confidence:    0.9,
			projectID:     projectID,
			expectError:   true,
			errorContains: "object cannot be empty",
		},
		{
			name:          "invalid confidence too high",
			subject:       "I",
			predicate:     "attended",
			object:        "meeting",
			timestamp:     now,
			confidence:    1.5,
			projectID:     projectID,
			expectError:   true,
			errorContains: "confidence must be between",
		},
		{
			name:          "invalid confidence too low",
			subject:       "I",
			predicate:     "attended",
			object:        "meeting",
			timestamp:     now,
			confidence:    -0.1,
			projectID:     projectID,
			expectError:   true,
			errorContains: "confidence must be between",
		},
		{
			name:          "empty project ID",
			subject:       "I",
			predicate:     "attended",
			object:        "meeting",
			timestamp:     now,
			confidence:    0.9,
			projectID:     "",
			expectError:   true,
			errorContains: "project ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fact, err := NewFact(tt.subject, tt.predicate, tt.object, tt.timestamp, tt.confidence, tt.provenance, tt.sourceID, tt.projectID)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, fact)
			} else {
				require.NoError(t, err)
				require.NotNil(t, fact)
				assert.NotEmpty(t, fact.ID)
				assert.Equal(t, tt.subject, fact.Subject)
				assert.Equal(t, tt.predicate, fact.Predicate)
				assert.Equal(t, tt.object, fact.Object)
				assert.Equal(t, tt.confidence, fact.Confidence)
				assert.Equal(t, tt.projectID, fact.ProjectID)
				assert.False(t, fact.CreatedAt.IsZero())

				// Verify UUID format
				_, err := uuid.Parse(fact.ID)
				assert.NoError(t, err)
			}
		})
	}
}

func TestFactValidate(t *testing.T) {
	now := time.Now()
	projectID := "test-project"

	validFact, err := NewFact("I", "attended", "meeting", now, 0.9, "provenance", "source-1", projectID)
	require.NoError(t, err)

	tests := []struct {
		name          string
		fact          *Fact
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid fact",
			fact:        validFact,
			expectError: false,
		},
		{
			name: "empty ID",
			fact: &Fact{
				ID:         "",
				Subject:    "I",
				Predicate:  "attended",
				Object:     "meeting",
				Timestamp:  now,
				Confidence: 0.9,
				ProjectID:  projectID,
			},
			expectError:   true,
			errorContains: "ID cannot be empty",
		},
		{
			name: "invalid UUID",
			fact: &Fact{
				ID:         "not-a-uuid",
				Subject:    "I",
				Predicate:  "attended",
				Object:     "meeting",
				Timestamp:  now,
				Confidence: 0.9,
				ProjectID:  projectID,
			},
			expectError:   true,
			errorContains: "invalid fact ID format",
		},
		{
			name: "empty subject",
			fact: &Fact{
				ID:         uuid.New().String(),
				Subject:    "",
				Predicate:  "attended",
				Object:     "meeting",
				Timestamp:  now,
				Confidence: 0.9,
				ProjectID:  projectID,
			},
			expectError:   true,
			errorContains: "subject cannot be empty",
		},
		{
			name: "empty predicate",
			fact: &Fact{
				ID:         uuid.New().String(),
				Subject:    "I",
				Predicate:  "",
				Object:     "meeting",
				Timestamp:  now,
				Confidence: 0.9,
				ProjectID:  projectID,
			},
			expectError:   true,
			errorContains: "predicate cannot be empty",
		},
		{
			name: "empty object",
			fact: &Fact{
				ID:         uuid.New().String(),
				Subject:    "I",
				Predicate:  "attended",
				Object:     "",
				Timestamp:  now,
				Confidence: 0.9,
				ProjectID:  projectID,
			},
			expectError:   true,
			errorContains: "object cannot be empty",
		},
		{
			name: "invalid confidence",
			fact: &Fact{
				ID:         uuid.New().String(),
				Subject:    "I",
				Predicate:  "attended",
				Object:     "meeting",
				Timestamp:  now,
				Confidence: 1.5,
				ProjectID:  projectID,
			},
			expectError:   true,
			errorContains: "confidence must be between",
		},
		{
			name: "empty project ID",
			fact: &Fact{
				ID:         uuid.New().String(),
				Subject:    "I",
				Predicate:  "attended",
				Object:     "meeting",
				Timestamp:  now,
				Confidence: 0.9,
				ProjectID:  "",
			},
			expectError:   true,
			errorContains: "project ID cannot be empty",
		},
		{
			name: "zero timestamp",
			fact: &Fact{
				ID:         uuid.New().String(),
				Subject:    "I",
				Predicate:  "attended",
				Object:     "meeting",
				Timestamp:  time.Time{},
				Confidence: 0.9,
				ProjectID:  projectID,
			},
			expectError:   true,
			errorContains: "timestamp cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fact.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

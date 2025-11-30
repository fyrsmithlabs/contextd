package qdrant

import (
	"context"
	"testing"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/qdrant/go-client/qdrant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClientConfig_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config *ClientConfig
		check  func(t *testing.T, cfg *ClientConfig)
	}{
		{
			name:   "empty config gets all defaults",
			config: &ClientConfig{},
			check: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, "localhost", cfg.Host)
				assert.Equal(t, 6334, cfg.Port)
				assert.Equal(t, false, cfg.UseTLS)
				assert.Equal(t, 50*1024*1024, cfg.MaxMessageSize)
				assert.Equal(t, 5*time.Second, cfg.DialTimeout)
				assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
				assert.Equal(t, 3, cfg.RetryAttempts)
			},
		},
		{
			name: "partial config preserves set values",
			config: &ClientConfig{
				Host: "qdrant.example.com",
				Port: 6335,
			},
			check: func(t *testing.T, cfg *ClientConfig) {
				assert.Equal(t, "qdrant.example.com", cfg.Host)
				assert.Equal(t, 6335, cfg.Port)
				assert.Equal(t, 50*1024*1024, cfg.MaxMessageSize)
				assert.Equal(t, 5*time.Second, cfg.DialTimeout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.ApplyDefaults()
			tt.check(t, tt.config)
		})
	}
}

func TestClientConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ClientConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &ClientConfig{
				Host:           "localhost",
				Port:           6334,
				MaxMessageSize: 1024,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: &ClientConfig{
				Port:           6334,
				MaxMessageSize: 1024,
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "invalid port - zero",
			config: &ClientConfig{
				Host:           "localhost",
				Port:           0,
				MaxMessageSize: 1024,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid port - negative",
			config: &ClientConfig{
				Host:           "localhost",
				Port:           -1,
				MaxMessageSize: 1024,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid port - too large",
			config: &ClientConfig{
				Host:           "localhost",
				Port:           65536,
				MaxMessageSize: 1024,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid max message size",
			config: &ClientConfig{
				Host:           "localhost",
				Port:           6334,
				MaxMessageSize: 0,
			},
			wantErr: true,
			errMsg:  "invalid max message size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConvertToQdrantPoint(t *testing.T) {
	tests := []struct {
		name  string
		point *Point
		check func(t *testing.T, qp *qdrant.PointStruct)
	}{
		{
			name: "point with mixed payload types",
			point: &Point{
				ID:     "test-id-123",
				Vector: []float32{0.1, 0.2, 0.3},
				Payload: map[string]interface{}{
					"string_field":  "test",
					"int_field":     42,
					"int64_field":   int64(100),
					"float_field":   3.14,
					"bool_field":    true,
					"unknown_field": struct{}{}, // Should convert to string
				},
			},
			check: func(t *testing.T, qp *qdrant.PointStruct) {
				assert.NotNil(t, qp)
				assert.NotNil(t, qp.Id)
				assert.NotNil(t, qp.Vectors)
				assert.Len(t, qp.Payload, 6)

				// Check string field
				assert.Equal(t, "test", qp.Payload["string_field"].GetStringValue())

				// Check int fields
				assert.Equal(t, int64(42), qp.Payload["int_field"].GetIntegerValue())
				assert.Equal(t, int64(100), qp.Payload["int64_field"].GetIntegerValue())

				// Check float field
				assert.Equal(t, 3.14, qp.Payload["float_field"].GetDoubleValue())

				// Check bool field
				assert.Equal(t, true, qp.Payload["bool_field"].GetBoolValue())

				// Check unknown field converted to string
				assert.Contains(t, qp.Payload["unknown_field"].GetStringValue(), "{}")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToQdrantPoint(tt.point)
			tt.check(t, result)
		})
	}
}

func TestConvertToQdrantFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		check  func(t *testing.T, qf *qdrant.Filter)
	}{
		{
			name:   "nil filter",
			filter: nil,
			check: func(t *testing.T, qf *qdrant.Filter) {
				assert.Nil(t, qf)
			},
		},
		{
			name: "filter with Must conditions",
			filter: &Filter{
				Must: []Condition{
					{
						Field: "category",
						Match: "test",
					},
					{
						Field: "confidence",
						Range: &RangeCondition{
							Gte: ptrFloat64(0.5),
							Lte: ptrFloat64(1.0),
						},
					},
				},
			},
			check: func(t *testing.T, qf *qdrant.Filter) {
				assert.NotNil(t, qf)
				assert.Len(t, qf.Must, 2)

				// Check first condition (Match)
				fieldCond := qf.Must[0].GetField()
				assert.NotNil(t, fieldCond)
				assert.Equal(t, "category", fieldCond.Key)
				assert.Equal(t, "test", fieldCond.Match.GetKeyword())

				// Check second condition (Range)
				rangeCond := qf.Must[1].GetField()
				assert.NotNil(t, rangeCond)
				assert.Equal(t, "confidence", rangeCond.Key)
				assert.NotNil(t, rangeCond.Range)
				assert.Equal(t, 0.5, *rangeCond.Range.Gte)
				assert.Equal(t, 1.0, *rangeCond.Range.Lte)
			},
		},
		{
			name: "filter with Should and MustNot conditions",
			filter: &Filter{
				Should: []Condition{
					{
						Field: "tag",
						Match: "urgent",
					},
				},
				MustNot: []Condition{
					{
						Field: "status",
						Match: "archived",
					},
				},
			},
			check: func(t *testing.T, qf *qdrant.Filter) {
				assert.NotNil(t, qf)
				assert.Len(t, qf.Should, 1)
				assert.Len(t, qf.MustNot, 1)

				// Check Should condition
				shouldCond := qf.Should[0].GetField()
				assert.Equal(t, "tag", shouldCond.Key)
				assert.Equal(t, "urgent", shouldCond.Match.GetKeyword())

				// Check MustNot condition
				mustNotCond := qf.MustNot[0].GetField()
				assert.Equal(t, "status", mustNotCond.Key)
				assert.Equal(t, "archived", mustNotCond.Match.GetKeyword())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToQdrantFilter(tt.filter)
			tt.check(t, result)
		})
	}
}

func TestExtractPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]*qdrant.Value
		want    map[string]interface{}
	}{
		{
			name:    "nil payload",
			payload: nil,
			want:    nil,
		},
		{
			name: "mixed value types",
			payload: map[string]*qdrant.Value{
				"string": {Kind: &qdrant.Value_StringValue{StringValue: "test"}},
				"int":    {Kind: &qdrant.Value_IntegerValue{IntegerValue: 42}},
				"float":  {Kind: &qdrant.Value_DoubleValue{DoubleValue: 3.14}},
				"bool":   {Kind: &qdrant.Value_BoolValue{BoolValue: true}},
			},
			want: map[string]interface{}{
				"string": "test",
				"int":    int64(42),
				"float":  3.14,
				"bool":   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPayload(tt.payload)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "unavailable error",
			err:  status.Error(codes.Unavailable, "service unavailable"),
			want: true,
		},
		{
			name: "deadline exceeded error",
			err:  status.Error(codes.DeadlineExceeded, "timeout"),
			want: true,
		},
		{
			name: "aborted error",
			err:  status.Error(codes.Aborted, "aborted"),
			want: true,
		},
		{
			name: "resource exhausted error",
			err:  status.Error(codes.ResourceExhausted, "too many requests"),
			want: true,
		},
		{
			name: "not found error - not transient",
			err:  status.Error(codes.NotFound, "not found"),
			want: false,
		},
		{
			name: "invalid argument error - not transient",
			err:  status.Error(codes.InvalidArgument, "bad request"),
			want: false,
		},
		{
			name: "permission denied error - not transient",
			err:  status.Error(codes.PermissionDenied, "forbidden"),
			want: false,
		},
		{
			name: "already exists error - not transient",
			err:  status.Error(codes.AlreadyExists, "already exists"),
			want: false,
		},
		{
			name: "non-grpc error - not transient",
			err:  assert.AnError,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTransientError(tt.err)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestExtractPointID(t *testing.T) {
	tests := []struct {
		name string
		id   *qdrant.PointId
		want string
	}{
		{
			name: "nil id",
			id:   nil,
			want: "",
		},
		{
			name: "uuid id",
			id:   qdrant.NewIDUUID("550e8400-e29b-41d4-a716-446655440000"),
			want: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "numeric id",
			id:   qdrant.NewIDNum(12345),
			want: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPointID(tt.id)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 6334, cfg.Port)
	assert.False(t, cfg.UseTLS)
	assert.Equal(t, 50*1024*1024, cfg.MaxMessageSize)
	assert.Equal(t, 5*time.Second, cfg.DialTimeout)
	assert.Equal(t, 30*time.Second, cfg.RequestTimeout)
	assert.Equal(t, 3, cfg.RetryAttempts)
}

func TestConvertToQdrantRange(t *testing.T) {
	tests := []struct {
		name  string
		input *RangeCondition
		want  *qdrant.Range
	}{
		{
			name:  "nil range",
			input: nil,
			want:  nil,
		},
		{
			name: "full range with all fields",
			input: &RangeCondition{
				Gte: ptrFloat64(0.5),
				Lte: ptrFloat64(1.0),
				Gt:  ptrFloat64(0.4),
				Lt:  ptrFloat64(1.1),
			},
			want: &qdrant.Range{
				Gte: ptrFloat64(0.5),
				Lte: ptrFloat64(1.0),
				Gt:  ptrFloat64(0.4),
				Lt:  ptrFloat64(1.1),
			},
		},
		{
			name: "partial range with only Gte",
			input: &RangeCondition{
				Gte: ptrFloat64(0.5),
			},
			want: &qdrant.Range{
				Gte: ptrFloat64(0.5),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToQdrantRange(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestConvertToQdrantValue(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		check func(t *testing.T, val *qdrant.Value)
	}{
		{
			name:  "string value",
			input: "test string",
			check: func(t *testing.T, val *qdrant.Value) {
				assert.Equal(t, "test string", val.GetStringValue())
			},
		},
		{
			name:  "int value",
			input: 42,
			check: func(t *testing.T, val *qdrant.Value) {
				assert.Equal(t, int64(42), val.GetIntegerValue())
			},
		},
		{
			name:  "int64 value",
			input: int64(100),
			check: func(t *testing.T, val *qdrant.Value) {
				assert.Equal(t, int64(100), val.GetIntegerValue())
			},
		},
		{
			name:  "float64 value",
			input: 3.14,
			check: func(t *testing.T, val *qdrant.Value) {
				assert.Equal(t, 3.14, val.GetDoubleValue())
			},
		},
		{
			name:  "bool value - true",
			input: true,
			check: func(t *testing.T, val *qdrant.Value) {
				assert.Equal(t, true, val.GetBoolValue())
			},
		},
		{
			name:  "bool value - false",
			input: false,
			check: func(t *testing.T, val *qdrant.Value) {
				assert.Equal(t, false, val.GetBoolValue())
			},
		},
		{
			name:  "unknown type - converts to string",
			input: struct{ Field string }{Field: "test"},
			check: func(t *testing.T, val *qdrant.Value) {
				assert.Contains(t, val.GetStringValue(), "test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToQdrantValue(tt.input)
			tt.check(t, result)
		})
	}
}

func TestExtractVectorOutput(t *testing.T) {
	t.Run("nil vectors", func(t *testing.T) {
		result := extractVectorOutput(nil)
		assert.Nil(t, result)
	})

	// Note: Testing with actual VectorsOutput structure would require
	// integration tests or mocking the entire Qdrant response structure.
	// The extraction logic is covered by integration tests in vectorstore package.
}

// Helper functions for tests

func ptrFloat64(v float64) *float64 {
	return &v
}

func TestRetryOperation_Logging(t *testing.T) {
	tests := []struct {
		name          string
		operation     func() error
		retryAttempts int
		expectedLogs  []struct {
			level   zapcore.Level
			message string
		}
	}{
		{
			name: "successful operation - no retries - no logs",
			operation: func() error {
				return nil
			},
			retryAttempts: 3,
			expectedLogs: []struct {
				level   zapcore.Level
				message string
			}{},
		},
		{
			name: "transient error then success - logs retry and recovery",
			operation: func() func() error {
				attempt := 0
				return func() error {
					attempt++
					if attempt == 1 {
						return status.Error(codes.Unavailable, "service unavailable")
					}
					return nil
				}
			}(),
			retryAttempts: 3,
			expectedLogs: []struct {
				level   zapcore.Level
				message string
			}{
				{level: zapcore.DebugLevel, message: "retrying operation after transient error"},
				{level: zapcore.InfoLevel, message: "operation recovered after retries"},
			},
		},
		{
			name: "all retries exhausted - logs all attempts and final failure",
			operation: func() error {
				return status.Error(codes.Unavailable, "service unavailable")
			},
			retryAttempts: 2,
			expectedLogs: []struct {
				level   zapcore.Level
				message string
			}{
				{level: zapcore.DebugLevel, message: "retrying operation after transient error"},
				{level: zapcore.DebugLevel, message: "retrying operation after transient error"},
				{level: zapcore.WarnLevel, message: "operation failed after all retries exhausted"},
			},
		},
		{
			name: "non-transient error - no retry logs",
			operation: func() error {
				return status.Error(codes.InvalidArgument, "bad request")
			},
			retryAttempts: 3,
			expectedLogs: []struct {
				level   zapcore.Level
				message string
			}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger
			testLogger := logging.NewTestLogger()

			// Create client with test logger
			client := &GRPCClient{
				config: &ClientConfig{
					RetryAttempts: tt.retryAttempts,
				},
				logger: testLogger.Logger,
			}

			// Execute operation
			ctx := context.Background()
			_ = client.retryOperation(ctx, tt.operation)

			// Verify expected logs
			for _, expectedLog := range tt.expectedLogs {
				testLogger.AssertLogged(t, expectedLog.level, expectedLog.message)
			}
		})
	}
}

func TestNewGRPCClient_RequiresLogger(t *testing.T) {
	config := DefaultClientConfig()

	// Should fail without logger
	_, err := NewGRPCClient(config, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "logger is required")
}

package qdrant

import (
	"context"
	"fmt"
	"time"

	"github.com/fyrsmithlabs/contextd/internal/logging"
	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// GRPCClient implements the Client interface using Qdrant's official Go client.
type GRPCClient struct {
	client *qdrant.Client
	config *ClientConfig
	logger *logging.Logger
}

// ClientConfig configures the Qdrant gRPC client.
type ClientConfig struct {
	// Host is the Qdrant server hostname or IP address.
	// Default: "localhost"
	Host string

	// Port is the Qdrant gRPC port (NOT HTTP REST port).
	// Default: 6334 (gRPC), not 6333 (HTTP)
	Port int

	// UseTLS enables TLS encryption for gRPC connection.
	// Default: false (for local development)
	UseTLS bool

	// APIKey is the optional API key for authentication.
	// Leave empty for local development.
	APIKey string

	// MaxMessageSize is the maximum gRPC message size in bytes.
	// Default: 50MB (to handle large documents)
	MaxMessageSize int

	// DialTimeout is the timeout for establishing connection.
	// Default: 5 seconds
	DialTimeout time.Duration

	// RequestTimeout is the default timeout for individual requests.
	// Default: 30 seconds
	RequestTimeout time.Duration

	// RetryAttempts is the number of retry attempts for transient failures.
	// Default: 3
	RetryAttempts int

	// Distance is the default distance metric for new collections.
	// Default: Cosine
	Distance qdrant.Distance
}

// DefaultClientConfig returns sensible defaults for local development.
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Host:           "localhost",
		Port:           6334,
		UseTLS:         false,
		MaxMessageSize: 50 * 1024 * 1024, // 50MB
		DialTimeout:    5 * time.Second,
		RequestTimeout: 30 * time.Second,
		RetryAttempts:  3,
		Distance:       qdrant.Distance_Cosine,
	}
}

// ApplyDefaults sets default values for unset fields.
func (c *ClientConfig) ApplyDefaults() {
	defaults := DefaultClientConfig()

	if c.Host == "" {
		c.Host = defaults.Host
	}
	if c.Port == 0 {
		c.Port = defaults.Port
	}
	if c.MaxMessageSize == 0 {
		c.MaxMessageSize = defaults.MaxMessageSize
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = defaults.DialTimeout
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = defaults.RequestTimeout
	}
	if c.RetryAttempts == 0 {
		c.RetryAttempts = defaults.RetryAttempts
	}
	if c.Distance == 0 {
		c.Distance = defaults.Distance
	}
}

// Validate validates the client configuration.
func (c *ClientConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Port)
	}
	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("invalid max message size: %d (must be > 0)", c.MaxMessageSize)
	}
	return nil
}

// NewGRPCClient creates a new Qdrant gRPC client.
func NewGRPCClient(config *ClientConfig, logger *logging.Logger) (*GRPCClient, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Apply defaults
	config.ApplyDefaults()

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Build Qdrant client config
	qdrantConfig := &qdrant.Config{
		Host:   config.Host,
		Port:   config.Port,
		UseTLS: config.UseTLS,
		APIKey: config.APIKey,
		GrpcOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(config.MaxMessageSize),
				grpc.MaxCallSendMsgSize(config.MaxMessageSize),
			),
		},
	}

	// For non-TLS connections, explicitly set insecure credentials
	if !config.UseTLS {
		qdrantConfig.GrpcOptions = append(qdrantConfig.GrpcOptions,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	}

	// Create Qdrant client
	client, err := qdrant.NewClient(qdrantConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	grpcClient := &GRPCClient{
		client: client,
		config: config,
		logger: logger,
	}

	// Perform health check
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	logger.Info(ctx, "connecting to qdrant",
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
	)

	if err := grpcClient.Health(ctx); err != nil {
		_ = client.Close()
		logger.Error(ctx, "qdrant health check failed",
			zap.String("host", config.Host),
			zap.Int("port", config.Port),
			zap.Error(err),
		)
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	logger.Info(ctx, "qdrant connection established",
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
	)

	return grpcClient, nil
}

// Health performs a health check on the Qdrant connection.
func (c *GRPCClient) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	_, err := c.client.HealthCheck(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// CreateCollection creates a new collection with the specified configuration.
func (c *GRPCClient) CreateCollection(ctx context.Context, name string, vectorSize uint64) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	return c.retryOperation(ctx, func() error {
		return c.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: name,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     vectorSize,
				Distance: c.config.Distance,
			}),
		})
	})
}

// DeleteCollection deletes a collection and all its documents.
func (c *GRPCClient) DeleteCollection(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	return c.retryOperation(ctx, func() error {
		return c.client.DeleteCollection(ctx, name)
	})
}

// CollectionExists checks if a collection exists.
func (c *GRPCClient) CollectionExists(ctx context.Context, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	var exists bool
	err := c.retryOperation(ctx, func() error {
		info, err := c.client.GetCollectionInfo(ctx, name)
		if err != nil {
			// Check if it's a not found error
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.NotFound {
				exists = false
				return nil // Not an error, just doesn't exist
			}
			return err
		}
		exists = info != nil
		return nil
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}

// ListCollections returns a list of all collection names.
func (c *GRPCClient) ListCollections(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	var collections []string
	err := c.retryOperation(ctx, func() error {
		result, err := c.client.ListCollections(ctx)
		if err != nil {
			return err
		}
		collections = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	return collections, nil
}

// Upsert inserts or updates points in a collection.
func (c *GRPCClient) Upsert(ctx context.Context, collection string, points []*Point) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	// Convert to Qdrant points
	qdrantPoints := make([]*qdrant.PointStruct, len(points))
	for i, point := range points {
		qdrantPoints[i] = convertToQdrantPoint(point)
	}

	return c.retryOperation(ctx, func() error {
		_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: collection,
			Points:         qdrantPoints,
		})
		return err
	})
}

// Search performs similarity search in a collection.
func (c *GRPCClient) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter *Filter) ([]*ScoredPoint, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	var results []*qdrant.ScoredPoint
	err := c.retryOperation(ctx, func() error {
		// Convert filter if provided
		var qdrantFilter *qdrant.Filter
		if filter != nil {
			qdrantFilter = convertToQdrantFilter(filter)
		}

		res, err := c.client.Query(ctx, &qdrant.QueryPoints{
			CollectionName: collection,
			Query:          qdrant.NewQuery(vector...),
			Limit:          qdrant.PtrOf(limit),
			WithPayload:    qdrant.NewWithPayload(true),
			Filter:         qdrantFilter,
		})
		if err != nil {
			return err
		}
		results = res
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Convert results
	scoredPoints := make([]*ScoredPoint, len(results))
	for i, result := range results {
		scoredPoints[i] = convertFromQdrantScoredPoint(result)
	}
	return scoredPoints, nil
}

// Get retrieves points by their IDs.
func (c *GRPCClient) Get(ctx context.Context, collection string, ids []string) ([]*Point, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	var points []*qdrant.RetrievedPoint
	err := c.retryOperation(ctx, func() error {
		// Convert string IDs to PointId
		pointIDs := make([]*qdrant.PointId, len(ids))
		for i, id := range ids {
			pointIDs[i] = qdrant.NewIDUUID(id)
		}

		result, err := c.client.Get(ctx, &qdrant.GetPoints{
			CollectionName: collection,
			Ids:            pointIDs,
			WithPayload:    qdrant.NewWithPayload(true),
		})
		if err != nil {
			return err
		}
		points = result
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Convert results
	result := make([]*Point, len(points))
	for i, p := range points {
		result[i] = convertFromQdrantRetrievedPoint(p)
	}
	return result, nil
}

// Delete removes points from a collection.
func (c *GRPCClient) Delete(ctx context.Context, collection string, ids []string) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()

	return c.retryOperation(ctx, func() error {
		// Convert string IDs to PointId
		pointIDs := make([]*qdrant.PointId, len(ids))
		for i, id := range ids {
			pointIDs[i] = qdrant.NewIDUUID(id)
		}

		_, err := c.client.Delete(ctx, &qdrant.DeletePoints{
			CollectionName: collection,
			Points: &qdrant.PointsSelector{
				PointsSelectorOneOf: &qdrant.PointsSelector_Points{
					Points: &qdrant.PointsIdsList{
						Ids: pointIDs,
					},
				},
			},
		})
		return err
	})
}

// Close closes the client connection.
func (c *GRPCClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// retryOperation retries an operation with exponential backoff.
func (c *GRPCClient) retryOperation(ctx context.Context, operation func() error) error {
	var lastErr error
	backoff := time.Second
	startTime := time.Now()

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		err := operation()
		if err == nil {
			// Log successful recovery after retries
			if attempt > 0 {
				c.logger.Info(ctx, "operation recovered after retries",
					zap.Int("attempts", attempt),
					zap.Duration("total_time", time.Since(startTime)),
				)
			}
			return nil
		}

		lastErr = err

		// Check if error is transient
		if !isTransientError(err) {
			return err
		}

		// Last attempt, return error
		if attempt == c.config.RetryAttempts {
			break
		}

		// Log retry attempt
		c.logger.Debug(ctx, "retrying operation after transient error",
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", c.config.RetryAttempts),
			zap.Error(err),
			zap.Duration("backoff", backoff),
		)

		// Wait before retry (exponential backoff)
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation canceled: %w", ctx.Err())
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	// Log final failure
	c.logger.Warn(ctx, "operation failed after all retries exhausted",
		zap.Int("total_attempts", c.config.RetryAttempts+1),
		zap.Duration("total_time", time.Since(startTime)),
		zap.Error(lastErr),
	)

	return fmt.Errorf("operation failed after %d retries: %w", c.config.RetryAttempts, lastErr)
}

// isTransientError checks if an error is transient and should be retried.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Aborted, codes.ResourceExhausted:
		return true
	case codes.InvalidArgument, codes.NotFound, codes.PermissionDenied, codes.Unauthenticated, codes.AlreadyExists:
		return false
	default:
		return false
	}
}

// Helper conversion functions

func convertToQdrantPoint(p *Point) *qdrant.PointStruct {
	// Convert payload
	payload := make(map[string]*qdrant.Value)
	for k, v := range p.Payload {
		payload[k] = convertToQdrantValue(v)
	}

	return &qdrant.PointStruct{
		Id:      qdrant.NewIDUUID(p.ID),
		Vectors: qdrant.NewVectors(p.Vector...),
		Payload: payload,
	}
}

func convertToQdrantValue(v interface{}) *qdrant.Value {
	switch val := v.(type) {
	case string:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
	case int:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
	case int64:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: val}}
	case float64:
		return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: val}}
	case bool:
		return &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: val}}
	default:
		// Fallback to string representation
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}

func convertFromQdrantScoredPoint(p *qdrant.ScoredPoint) *ScoredPoint {
	return &ScoredPoint{
		Point: Point{
			ID:      extractPointID(p.Id),
			Vector:  extractVectorOutput(p.Vectors),
			Payload: extractPayload(p.Payload),
		},
		Score: p.Score,
	}
}

func convertFromQdrantRetrievedPoint(p *qdrant.RetrievedPoint) *Point {
	return &Point{
		ID:      extractPointID(p.Id),
		Vector:  extractVectorOutput(p.Vectors),
		Payload: extractPayload(p.Payload),
	}
}

func extractPointID(id *qdrant.PointId) string {
	if id == nil {
		return ""
	}
	if uuid := id.GetUuid(); uuid != "" {
		return uuid
	}
	if num := id.GetNum(); num != 0 {
		return fmt.Sprintf("%d", num)
	}
	return ""
}

func extractVectorOutput(vectors *qdrant.VectorsOutput) []float32 {
	if vectors == nil {
		return nil
	}
	if vec := vectors.GetVector(); vec != nil {
		if dense := vec.GetDense(); dense != nil {
			return dense.GetData()
		}
	}
	return nil
}

func extractPayload(payload map[string]*qdrant.Value) map[string]interface{} {
	if payload == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range payload {
		result[k] = extractValue(v)
	}
	return result
}

func extractValue(v *qdrant.Value) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.Kind.(type) {
	case *qdrant.Value_StringValue:
		return val.StringValue
	case *qdrant.Value_IntegerValue:
		return val.IntegerValue
	case *qdrant.Value_DoubleValue:
		return val.DoubleValue
	case *qdrant.Value_BoolValue:
		return val.BoolValue
	default:
		return nil
	}
}

func convertToQdrantFilter(f *Filter) *qdrant.Filter {
	if f == nil {
		return nil
	}

	filter := &qdrant.Filter{}

	// Convert Must conditions
	if len(f.Must) > 0 {
		filter.Must = make([]*qdrant.Condition, len(f.Must))
		for i, cond := range f.Must {
			filter.Must[i] = convertToQdrantCondition(cond)
		}
	}

	// Convert Should conditions
	if len(f.Should) > 0 {
		filter.Should = make([]*qdrant.Condition, len(f.Should))
		for i, cond := range f.Should {
			filter.Should[i] = convertToQdrantCondition(cond)
		}
	}

	// Convert MustNot conditions
	if len(f.MustNot) > 0 {
		filter.MustNot = make([]*qdrant.Condition, len(f.MustNot))
		for i, cond := range f.MustNot {
			filter.MustNot[i] = convertToQdrantCondition(cond)
		}
	}

	return filter
}

func convertToQdrantCondition(c Condition) *qdrant.Condition {
	// Handle Match condition
	if c.Match != nil {
		return &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: c.Field,
					Match: &qdrant.Match{
						MatchValue: convertToQdrantMatch(c.Match),
					},
				},
			},
		}
	}

	// Handle Range condition
	if c.Range != nil {
		return &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   c.Field,
					Range: convertToQdrantRange(c.Range),
				},
			},
		}
	}

	return nil
}

func convertToQdrantMatch(match interface{}) *qdrant.Match_Keyword {
	switch v := match.(type) {
	case string:
		return &qdrant.Match_Keyword{Keyword: v}
	default:
		return &qdrant.Match_Keyword{Keyword: fmt.Sprintf("%v", v)}
	}
}

func convertToQdrantRange(r *RangeCondition) *qdrant.Range {
	if r == nil {
		return nil
	}

	return &qdrant.Range{
		Gte: r.Gte,
		Lte: r.Lte,
		Gt:  r.Gt,
		Lt:  r.Lt,
	}
}

// Ensure GRPCClient implements Client interface
var _ Client = (*GRPCClient)(nil)

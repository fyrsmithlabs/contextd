package vectorstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// mockEmbedder implements Embedder interface for testing.
type mockEmbedder struct {
	dimension int
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = make([]float32, m.dimension)
		// Simple deterministic embedding based on text length
		for j := 0; j < m.dimension; j++ {
			embeddings[i][j] = float32(len(texts[i])%10) / 10.0
		}
	}
	return embeddings, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, m.dimension)
	for i := 0; i < m.dimension; i++ {
		embedding[i] = float32(len(text)%10) / 10.0
	}
	return embedding, nil
}

func TestChromemStoreProvider_GetProjectStore(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	embedder := &mockEmbedder{dimension: 384}

	provider, err := NewChromemStoreProvider(ProviderConfig{
		BasePath:   tmpDir,
		VectorSize: 384,
	}, embedder, nil)
	if err != nil {
		t.Fatalf("NewChromemStoreProvider failed: %v", err)
	}
	defer provider.Close()

	// Get project store (free tier - no team)
	store1, err := provider.GetProjectStore(ctx, "acme", "", "contextd")
	if err != nil {
		t.Fatalf("GetProjectStore failed: %v", err)
	}
	if store1 == nil {
		t.Fatal("GetProjectStore returned nil store")
	}

	// Verify directory created
	projectPath := filepath.Join(tmpDir, "acme", "contextd")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("project directory not created: %s", projectPath)
	}

	// Get same store again (should return cached)
	store2, err := provider.GetProjectStore(ctx, "acme", "", "contextd")
	if err != nil {
		t.Fatalf("GetProjectStore (cached) failed: %v", err)
	}
	if store1 != store2 {
		t.Error("expected cached store to be returned")
	}

	// Get different project store
	store3, err := provider.GetProjectStore(ctx, "acme", "", "another-project")
	if err != nil {
		t.Fatalf("GetProjectStore (different project) failed: %v", err)
	}
	if store1 == store3 {
		t.Error("expected different store for different project")
	}
}

func TestChromemStoreProvider_GetProjectStore_WithTeam(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	embedder := &mockEmbedder{dimension: 384}

	provider, err := NewChromemStoreProvider(ProviderConfig{
		BasePath:   tmpDir,
		VectorSize: 384,
	}, embedder, nil)
	if err != nil {
		t.Fatalf("NewChromemStoreProvider failed: %v", err)
	}
	defer provider.Close()

	// Get project store (paid tier - with team)
	store, err := provider.GetProjectStore(ctx, "acme", "platform", "contextd")
	if err != nil {
		t.Fatalf("GetProjectStore with team failed: %v", err)
	}
	if store == nil {
		t.Fatal("GetProjectStore returned nil store")
	}

	// Verify directory structure created
	teamPath := filepath.Join(tmpDir, "acme", "platform")
	if _, err := os.Stat(teamPath); os.IsNotExist(err) {
		t.Errorf("team directory not created: %s", teamPath)
	}

	projectPath := filepath.Join(tmpDir, "acme", "platform", "contextd")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("project directory not created: %s", projectPath)
	}
}

func TestChromemStoreProvider_GetTeamStore(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	embedder := &mockEmbedder{dimension: 384}

	provider, err := NewChromemStoreProvider(ProviderConfig{
		BasePath:   tmpDir,
		VectorSize: 384,
	}, embedder, nil)
	if err != nil {
		t.Fatalf("NewChromemStoreProvider failed: %v", err)
	}
	defer provider.Close()

	// Get team store
	store, err := provider.GetTeamStore(ctx, "acme", "platform")
	if err != nil {
		t.Fatalf("GetTeamStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("GetTeamStore returned nil store")
	}

	// Verify directory created
	teamPath := filepath.Join(tmpDir, "acme", "platform")
	if _, err := os.Stat(teamPath); os.IsNotExist(err) {
		t.Errorf("team directory not created: %s", teamPath)
	}
}

func TestChromemStoreProvider_GetOrgStore(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	embedder := &mockEmbedder{dimension: 384}

	provider, err := NewChromemStoreProvider(ProviderConfig{
		BasePath:   tmpDir,
		VectorSize: 384,
	}, embedder, nil)
	if err != nil {
		t.Fatalf("NewChromemStoreProvider failed: %v", err)
	}
	defer provider.Close()

	// Get org store
	store, err := provider.GetOrgStore(ctx, "acme")
	if err != nil {
		t.Fatalf("GetOrgStore failed: %v", err)
	}
	if store == nil {
		t.Fatal("GetOrgStore returned nil store")
	}

	// Verify directory created
	orgPath := filepath.Join(tmpDir, "acme")
	if _, err := os.Stat(orgPath); os.IsNotExist(err) {
		t.Errorf("org directory not created: %s", orgPath)
	}
}

func TestChromemStoreProvider_InvalidNames(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	embedder := &mockEmbedder{dimension: 384}

	provider, err := NewChromemStoreProvider(ProviderConfig{
		BasePath:   tmpDir,
		VectorSize: 384,
	}, embedder, nil)
	if err != nil {
		t.Fatalf("NewChromemStoreProvider failed: %v", err)
	}
	defer provider.Close()

	// Test path traversal attempts
	_, err = provider.GetProjectStore(ctx, "../evil", "", "project")
	if err == nil {
		t.Error("expected error for path traversal tenant, got nil")
	}

	_, err = provider.GetProjectStore(ctx, "acme", "", "../evil")
	if err == nil {
		t.Error("expected error for path traversal project, got nil")
	}

	_, err = provider.GetProjectStore(ctx, "acme", "../evil", "project")
	if err == nil {
		t.Error("expected error for path traversal team, got nil")
	}
}

func TestChromemStoreProvider_StoreIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	embedder := &mockEmbedder{dimension: 384}

	provider, err := NewChromemStoreProvider(ProviderConfig{
		BasePath:   tmpDir,
		VectorSize: 384,
	}, embedder, nil)
	if err != nil {
		t.Fatalf("NewChromemStoreProvider failed: %v", err)
	}
	defer provider.Close()

	// Get two different project stores
	store1, err := provider.GetProjectStore(ctx, "tenant1", "", "project1")
	if err != nil {
		t.Fatalf("GetProjectStore 1 failed: %v", err)
	}

	store2, err := provider.GetProjectStore(ctx, "tenant2", "", "project2")
	if err != nil {
		t.Fatalf("GetProjectStore 2 failed: %v", err)
	}

	// Add document to store1
	doc := Document{
		ID:      "doc1",
		Content: "test content for store 1",
		Metadata: map[string]interface{}{
			"source": "test",
		},
		Collection: "test_collection",
	}
	_, err = store1.AddDocuments(ctx, []Document{doc})
	if err != nil {
		t.Fatalf("AddDocuments to store1 failed: %v", err)
	}

	// Verify document exists in store1
	exists, err := store1.CollectionExists(ctx, "test_collection")
	if err != nil {
		t.Fatalf("CollectionExists store1 failed: %v", err)
	}
	if !exists {
		t.Error("expected collection to exist in store1")
	}

	// Verify collection does NOT exist in store2 (isolation)
	exists, err = store2.CollectionExists(ctx, "test_collection")
	if err != nil {
		t.Fatalf("CollectionExists store2 failed: %v", err)
	}
	if exists {
		t.Error("collection should NOT exist in store2 - isolation violated")
	}
}

func TestChromemStoreProvider_Registry(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	embedder := &mockEmbedder{dimension: 384}

	provider, err := NewChromemStoreProvider(ProviderConfig{
		BasePath:   tmpDir,
		VectorSize: 384,
	}, embedder, nil)
	if err != nil {
		t.Fatalf("NewChromemStoreProvider failed: %v", err)
	}
	defer provider.Close()

	// Create some projects
	_, err = provider.GetProjectStore(ctx, "acme", "", "project1")
	if err != nil {
		t.Fatalf("GetProjectStore failed: %v", err)
	}
	_, err = provider.GetProjectStore(ctx, "acme", "", "project2")
	if err != nil {
		t.Fatalf("GetProjectStore failed: %v", err)
	}

	// Verify registry tracks them
	reg := provider.Registry()
	tenants := reg.ListTenants()
	if len(tenants) != 1 || tenants[0] != "acme" {
		t.Errorf("expected [acme], got %v", tenants)
	}

	projects := reg.ListProjects("acme")
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	// Verify UUIDs are assigned
	entry, err := reg.GetProject("acme", "", "project1")
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if entry.UUID == "" {
		t.Error("expected UUID to be assigned")
	}
}

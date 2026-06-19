package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// newResourceHandler returns a trivial resource handler that echoes the
// requested URI back as an empty JSON array. It satisfies the SDK's
// requirement that a resource exist before a subscription is meaningful.
func newResourceHandler() mcp.ResourceHandler {
	return func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: "[]"}},
		}, nil
	}
}

// TestNotifyCollectionUpdated_NoSubscribers asserts the helper is a safe no-op
// (no error, no panic) when no client has subscribed. ResourceUpdated simply
// fans out to an empty set of sessions.
//
// This uses the bare-server construction form: a *mcp.Server created with nil
// options has no SubscribeHandler, so it cannot actually track subscriptions —
// but ResourceUpdated is still callable and must not panic.
func TestNotifyCollectionUpdated_NoSubscribers(t *testing.T) {
	s := &Server{
		mcp: mcp.NewServer(&mcp.Implementation{
			Name:    "contextd",
			Version: "test",
		}, nil),
		logger: zap.NewNop(),
	}
	s.mcp.AddResource(&mcp.Resource{URI: "contextd://proj/memories", Name: "m"}, newResourceHandler())

	// Must not panic and must return cleanly even with zero subscribers.
	s.notifyCollectionUpdated(context.Background(), "proj", "memories")
	s.notifyCollectionUpdated(context.Background(), "proj", "remediations")
	s.notifyCollectionUpdated(context.Background(), "proj", "checkpoints")
}

// TestNotifyCollectionUpdated_NilSafe asserts the helper guards against a nil
// underlying MCP server (and a nil receiver) without panicking.
func TestNotifyCollectionUpdated_NilSafe(t *testing.T) {
	// nil mcp server
	s := &Server{logger: zap.NewNop()}
	s.notifyCollectionUpdated(context.Background(), "proj", "memories")

	// nil receiver
	var ns *Server
	ns.notifyCollectionUpdated(context.Background(), "proj", "memories")
}

// TestNotifyCollectionUpdated_DeliversToSubscriber is the end-to-end swarm test:
// a client subscribes to a collection URI, the server records (via
// notifyCollectionUpdated), and the client must receive a
// notifications/resources/updated for that exact URI.
//
// NOTE on construction: the SDK only tracks subscriptions when the server was
// created with a SubscribeHandler/UnsubscribeHandler pair (see
// (*Server).subscribe in the go-sdk, which returns ErrMethodNotFound when
// SubscribeHandler is nil and never records the subscription). The production
// contextd server therefore must register these handlers to support the swarm
// mechanism. We mirror that here so the subscribe → notify → receive path is
// exercised for real. The notifyCollectionUpdated helper under test is
// agnostic to how the server was constructed.
func TestNotifyCollectionUpdated_DeliversToSubscriber(t *testing.T) {
	const uri = "contextd://proj/memories"

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "contextd",
		Version: "test",
	}, &mcp.ServerOptions{
		SubscribeHandler:   func(_ context.Context, _ *mcp.SubscribeRequest) error { return nil },
		UnsubscribeHandler: func(_ context.Context, _ *mcp.UnsubscribeRequest) error { return nil },
	})
	mcpServer.AddResource(&mcp.Resource{URI: uri, Name: "m"}, newResourceHandler())

	s := &Server{mcp: mcpServer, logger: zap.NewNop()}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Wire an in-memory transport pair: one end for the server session, one
	// for the client session.
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := s.mcp.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer serverSession.Close()

	// Client installs a handler for resources/updated notifications.
	updated := make(chan string, 1)
	client := mcp.NewClient(&mcp.Implementation{Name: "agent", Version: "test"}, &mcp.ClientOptions{
		ResourceUpdatedHandler: func(_ context.Context, req *mcp.ResourceUpdatedNotificationRequest) {
			// Non-blocking send; buffer of 1 is enough for this single event.
			select {
			case updated <- req.Params.URI:
			default:
			}
		},
	})

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer clientSession.Close()

	// Agent subscribes to the collection it cares about.
	if err := clientSession.Subscribe(ctx, &mcp.SubscribeParams{URI: uri}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Another agent records shared knowledge → server broadcasts the update.
	s.notifyCollectionUpdated(ctx, "proj", "memories")

	select {
	case got := <-updated:
		if got != uri {
			t.Fatalf("got updated notification for %q, want %q", got, uri)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for resources/updated notification: %v", ctx.Err())
	}
}

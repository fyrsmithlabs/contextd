package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

// HandleSSE streams operation progress via Server-Sent Events.
//
// This handler subscribes to NATS events for the specified operation
// and streams them to the client using SSE protocol. The connection
// remains open until the operation completes or the client disconnects.
//
// SSE Event Types:
//   - started: Operation began execution
//   - progress: Progress update (percent, message)
//   - log: Informational log message
//   - error: Operation failed
//   - completed: Operation finished successfully
//
// Example:
//
//	GET /mcp/sse/{operation_id}
//
//	event: started
//	data: {"id":"op-123","tool":"checkpoint_save"}
//
//	event: progress
//	data: {"id":"op-123","percent":50,"message":"Processing..."}
//
//	event: completed
//	data: {"id":"op-123","result":{"checkpoint_id":"ckpt-456"}}
func HandleSSE(c echo.Context, operations *OperationRegistry, nc *nats.Conn) error {
	opID := c.Param("operation_id")

	// Validate operation exists
	op, err := operations.Get(opID)
	if err != nil {
		return c.JSON(404, map[string]string{
			"error": "Operation not found",
		})
	}

	// Set SSE headers
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Subscribe to NATS events for this operation
	subject := fmt.Sprintf("operations.%s.%s.*", op.OwnerID, opID)
	msgChan := make(chan *nats.Msg, 10)
	sub, err := nc.ChanSubscribe(subject, msgChan)
	if err != nil {
		return err
	}
	defer func() {
		_ = sub.Unsubscribe()
	}()

	// Heartbeat ticker to prevent proxy timeouts
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Stream events until completion or disconnect
	for {
		select {
		case msg := <-msgChan:
			// Extract event type from subject
			parts := strings.Split(msg.Subject, ".")
			if len(parts) < 4 {
				continue
			}
			eventType := parts[3] // started, progress, log, error, completed

			// Send SSE event
			fmt.Fprintf(c.Response(), "event: %s\n", eventType)
			fmt.Fprintf(c.Response(), "data: %s\n\n", string(msg.Data))
			c.Response().Flush()

			// Close stream on completion or error
			if eventType == "completed" || eventType == "error" {
				return nil
			}

		case <-ticker.C:
			// Send heartbeat to keep connection alive
			fmt.Fprintf(c.Response(), ": heartbeat\n\n")
			c.Response().Flush()

		case <-c.Request().Context().Done():
			// Client disconnected
			return nil
		}
	}
}

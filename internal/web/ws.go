package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/theencryptedafro/appwrap/internal/service"
)

type wsHub struct {
	mu         sync.RWMutex
	operations map[string]*operation
}

type operation struct {
	events  chan service.Event
	buffer  []service.Event // buffered events for late joiners
	done    bool
	clients []*websocket.Conn
	mu      sync.Mutex
}

func newWSHub() *wsHub {
	return &wsHub{
		operations: make(map[string]*operation),
	}
}

// Register creates a new operation entry and starts forwarding events
// from the channel to connected clients.
func (h *wsHub) Register(opID string, events chan service.Event) {
	op := &operation{
		events: events,
	}

	h.mu.Lock()
	h.operations[opID] = op
	h.mu.Unlock()

	// Start a goroutine that reads from the events channel,
	// buffers events, and forwards to connected WS clients.
	go h.forwardEvents(opID, op)
}

// Complete marks an operation as done so late-connecting clients
// receive the full buffer plus a done indicator.
func (h *wsHub) Complete(opID string) {
	h.mu.RLock()
	op, ok := h.operations[opID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	op.mu.Lock()
	op.done = true
	op.mu.Unlock()

	// Schedule cleanup after 5 minutes (give late clients time to connect)
	go func() {
		time.Sleep(5 * time.Minute)
		h.mu.Lock()
		delete(h.operations, opID)
		h.mu.Unlock()
	}()
}

func (h *wsHub) forwardEvents(opID string, op *operation) {
	for evt := range op.events {
		op.mu.Lock()
		op.buffer = append(op.buffer, evt)
		// Send to all connected clients
		for _, conn := range op.clients {
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				log.Printf("ws: write error for client: %v", err)
			}
			cancel()
		}
		op.mu.Unlock()
	}
}

// addClient adds a WebSocket connection to an operation, replaying buffered events.
func (h *wsHub) addClient(opID string, conn *websocket.Conn) bool {
	h.mu.RLock()
	op, ok := h.operations[opID]
	h.mu.RUnlock()
	if !ok {
		return false
	}

	op.mu.Lock()
	defer op.mu.Unlock()

	// Replay buffered events
	for _, evt := range op.buffer {
		data, err := json.Marshal(evt)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			log.Printf("ws: write error for client: %v", err)
		}
		cancel()
	}

	// If operation is already done, send a synthetic done event and close
	if op.done {
		done := service.Event{
			Kind:      service.EventComplete,
			Message:   "__done__",
			Timestamp: time.Now(),
		}
		data, _ := json.Marshal(done)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
			log.Printf("ws: write error for client: %v", err)
		}
		cancel()
		return true
	}

	op.clients = append(op.clients, conn)
	return true
}

// removeClient removes a WebSocket connection from an operation.
func (h *wsHub) removeClient(opID string, conn *websocket.Conn) {
	h.mu.RLock()
	op, ok := h.operations[opID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	op.mu.Lock()
	defer op.mu.Unlock()

	for i, c := range op.clients {
		if c == conn {
			op.clients = append(op.clients[:i], op.clients[i+1:]...)
			break
		}
	}
}

// handleWS upgrades the HTTP connection to a WebSocket and streams events.
func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	opID := r.PathValue("opId")
	if opID == "" {
		http.Error(w, "missing operationId", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow connections from any origin during dev
	})
	if err != nil {
		http.Error(w, "websocket upgrade failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	if !s.hub.addClient(opID, conn) {
		conn.Close(websocket.StatusPolicyViolation, "unknown operation")
		return
	}
	defer s.hub.removeClient(opID, conn)

	// Keep connection alive by reading (and discarding) client messages.
	// The connection stays open until the client disconnects or the server closes it.
	ctx := r.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			return
		}
	}
}

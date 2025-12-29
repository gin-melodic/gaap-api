package ws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gorilla/websocket"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	MessageTypeTaskUpdate MessageType = "TASK_UPDATE"
	MessageTypePing       MessageType = "PING"
	MessageTypePong       MessageType = "PONG"
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// TaskUpdatePayload contains task update data
type TaskUpdatePayload struct {
	TaskId   string      `json:"taskId"`
	Status   string      `json:"status"`
	TaskType string      `json:"taskType"`
	Result   interface{} `json:"result,omitempty"`
}

// Hub manages WebSocket connections per user
type Hub struct {
	// userId -> set of connections
	connections map[string]map[*websocket.Conn]bool
	mu          sync.RWMutex
}

var (
	hubInstance *Hub
	hubOnce     sync.Once
)

// GetHub returns the singleton Hub instance
func GetHub() *Hub {
	hubOnce.Do(func() {
		hubInstance = &Hub{
			connections: make(map[string]map[*websocket.Conn]bool),
		}
	})
	return hubInstance
}

// Register adds a connection for a user
func (h *Hub) Register(userId string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.connections[userId] == nil {
		h.connections[userId] = make(map[*websocket.Conn]bool)
	}
	h.connections[userId][conn] = true
	g.Log().Debugf(context.TODO(), "WebSocket: registered connection for user %s (total: %d)", userId, len(h.connections[userId]))
}

// Unregister removes a connection for a user
func (h *Hub) Unregister(userId string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.connections[userId]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.connections, userId)
		}
		g.Log().Debugf(context.TODO(), "WebSocket: unregistered connection for user %s", userId)
	}
}

// SendToUser sends a message to all connections of a specific user
func (h *Hub) SendToUser(userId string, msg *Message) {
	h.mu.RLock()
	conns := h.connections[userId]
	h.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		g.Log().Errorf(context.TODO(), "WebSocket: failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range conns {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			g.Log().Warningf(context.TODO(), "WebSocket: failed to send message to user %s: %v", userId, err)
			// Connection will be cleaned up by the handler's read loop
		}
	}
	g.Log().Debugf(context.TODO(), "WebSocket: sent %s message to user %s (%d connections)", msg.Type, userId, len(conns))
}

// ConnectionCount returns the number of active connections for a user
func (h *Hub) ConnectionCount(userId string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections[userId])
}

// TotalConnections returns the total number of active connections
func (h *Hub) TotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, conns := range h.connections {
		total += len(conns)
	}
	return total
}

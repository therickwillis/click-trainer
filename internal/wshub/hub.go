package wshub

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/coder/websocket"
)

// ClientMessage is the JSON structure received from clients.
type ClientMessage struct {
	Type     string `json:"t"`
	TargetID int    `json:"id,omitempty"`
	Points   int    `json:"p,omitempty"`
	X        int    `json:"x,omitempty"`
	Y        int    `json:"y,omitempty"`
}

// ServerMessage is the JSON structure sent to clients.
type ServerMessage struct {
	Type     string `json:"t"`
	PlayerID string `json:"id,omitempty"`
	Name     string `json:"n,omitempty"`
	Color    string `json:"c,omitempty"`
	X        int    `json:"x,omitempty"`
	Y        int    `json:"y,omitempty"`
	Points   int    `json:"p,omitempty"`
}

// Client represents a single WebSocket connection in the hub.
type Client struct {
	PlayerID string
	Name     string
	Color    string
	Conn     *websocket.Conn
	Send     chan []byte
}

// WritePump reads from the Send channel and writes to the WebSocket connection.
func (c *Client) WritePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			if err := c.Conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		}
	}
}

// Hub manages per-room WebSocket connections.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.PlayerID] = c
}

// Unregister removes a client and closes its Send channel, then broadcasts a leave message.
func (h *Hub) Unregister(playerID string) {
	h.mu.Lock()
	c, ok := h.clients[playerID]
	if ok {
		close(c.Send)
		delete(h.clients, playerID)
	}
	h.mu.Unlock()

	if ok {
		h.BroadcastExcept(playerID, ServerMessage{
			Type:     "leave",
			PlayerID: playerID,
		})
	}
}

// BroadcastExcept sends a message to all clients except the sender. Non-blocking: drops if channel full.
func (h *Hub) BroadcastExcept(senderID string, msg ServerMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WSHub] Marshal error: %v\n", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for id, c := range h.clients {
		if id == senderID {
			continue
		}
		select {
		case c.Send <- data:
		default:
			// Drop message if channel full
		}
	}
}

package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for POS LAN & local kiosk hardware
	},
}

type EventMessage struct {
	Topic   string `json:"topic"` // e.g. "kds", "gate", "order", "qris"
	Event   string `json:"event"`
	Payload any    `json:"payload"`
}

type Client struct {
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	topic string
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastEvent(topic, event string, payload any) {
	msg := EventMessage{
		Topic:   topic,
		Event:   event,
		Payload: payload,
	}
	data, err := json.Marshal(msg)
	if err == nil {
		h.broadcast <- data
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	topic := r.URL.Query().Get("topic")
	client := &Client{
		hub:   h,
		conn:  conn,
		send:  make(chan []byte, 64),
		topic: topic,
	}

	h.register <- client

	// Start pump routines
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		// Read message; could handle incoming gate turnstile signals or KDS bump actions
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

package notifier

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"ripple/internal/store"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Client represents a single active WebSocket connection.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	projectID string
	userID    string
}

// Hub maintains the set of active WebSocket clients and broadcasts notifications.
type Hub struct {
	redisStore *store.RedisFeedStore
	clients    map[string]map[*Client]bool // key: "projectID:userID" -> set of *Client
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub(redisStore *store.RedisFeedStore) *Hub {
	return &Hub{
		redisStore: redisStore,
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			userKey := h.clientKey(client.projectID, client.userID)
			h.mu.Lock()
			if h.clients[userKey] == nil {
				h.clients[userKey] = make(map[*Client]bool)
			}
			h.clients[userKey][client] = true
			h.mu.Unlock()
			log.Printf("Hub: Client registered for %s", userKey)

		case client := <-h.unregister:
			userKey := h.clientKey(client.projectID, client.userID)
			h.mu.Lock()
			if userClients, ok := h.clients[userKey]; ok {
				if _, exists := userClients[client]; exists {
					delete(userClients, client)
					close(client.send)
					if len(userClients) == 0 {
						delete(h.clients, userKey)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Hub: Client unregistered for %s", userKey)

		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) clientKey(projectID, userID string) string {
	return projectID + ":" + userID
}

// HandleWebSocket upgrades HTTP connection to WebSocket and attaches it to the hub.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket for user %s: %v", userID, err)
		return
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan []byte, 256),
		projectID: projectID,
		userID:    userID,
	}

	h.register <- client

	ctx, cancel := context.WithCancel(context.Background())
	go h.listenRedisPubSub(ctx, client)

	go client.writePump(cancel)
	go client.readPump(cancel)
}

func (h *Hub) listenRedisPubSub(ctx context.Context, client *Client) {
	if h.redisStore == nil {
		return
	}

	pubsub := h.redisStore.SubscribeUserNotifications(ctx, client.projectID, client.userID)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			select {
			case client.send <- []byte(msg.Payload):
			default:
				log.Printf("Client send buffer full for user %s", client.userID)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) readPump(cancel context.CancelFunc) {
	defer func() {
		cancel()
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump(cancel context.CancelFunc) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		cancel()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

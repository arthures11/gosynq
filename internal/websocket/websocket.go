package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/arthures11/gosynq/internal/models"
	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	clients      map[*Client]bool
	clientsMutex sync.RWMutex
	eventChan    <-chan models.JobEvent
	upgrader     websocket.Upgrader
	shutdownCh   chan struct{}
	shutdownWg   sync.WaitGroup
}

type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	shutdown chan struct{}
}

func NewWebSocketServer(eventChan <-chan models.JobEvent) *WebSocketServer {
	return &WebSocketServer{
		clients:   make(map[*Client]bool),
		eventChan: eventChan,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: Add proper origin checking
			},
		},
		shutdownCh: make(chan struct{}),
	}
}

func (s *WebSocketServer) Start(ctx context.Context) {
	s.shutdownWg.Add(1)
	go func() {
		defer s.shutdownWg.Done()
		s.processEvents(ctx)
	}()
}

func (s *WebSocketServer) processEvents(ctx context.Context) {
	for {
		select {
		case <-s.shutdownCh:
			return
		case event := <-s.eventChan:
			s.broadcastEvent(event)
		}
	}
}

func (s *WebSocketServer) broadcastEvent(event models.JobEvent) {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	for client := range s.clients {
		select {
		case client.send <- []byte(event.ToJSON()):
		default:
			// Client send channel is full, skip this event
			log.Printf("WebSocket client send channel full, skipping event")
		}
	}
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		shutdown: make(chan struct{}),
	}

	s.clientsMutex.Lock()
	s.clients[client] = true
	s.clientsMutex.Unlock()

	s.shutdownWg.Add(2)
	go s.writePump(client)
	go s.readPump(client)
}

func (s *WebSocketServer) writePump(client *Client) {
	defer func() {
		s.shutdownWg.Done()
		s.removeClient(client)
		client.conn.Close()
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-client.shutdown:
			return
		case message, ok := <-client.send:
			if !ok {
				// Channel closed
				return
			}

			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *WebSocketServer) readPump(client *Client) {
	defer func() {
		s.shutdownWg.Done()
		s.removeClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
	}
}

func (s *WebSocketServer) removeClient(client *Client) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if _, ok := s.clients[client]; ok {
		close(client.send)
		delete(s.clients, client)
	}
}

func (s *WebSocketServer) Shutdown() {
	close(s.shutdownCh)

	// Close all client connections
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	for client := range s.clients {
		close(client.send)
		client.conn.Close()
	}

	s.shutdownWg.Wait()
}

func (s *WebSocketServer) GetClientCount() int {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()
	return len(s.clients)
}

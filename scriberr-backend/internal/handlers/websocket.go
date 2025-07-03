package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader configuration
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, you should check the origin
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Model    string
	Language string
	Translate bool
	WhisperConn *websocket.Conn // Connection to WhisperLive server
}

// Message types for WebSocket communication
type WSMessage struct {
	Type      string      `json:"type"`
	ClientID  string      `json:"client_id,omitempty"`
	ModelSize string      `json:"model_size,omitempty"`
	Language  string      `json:"language,omitempty"`
	Translate bool        `json:"translate,omitempty"`
	Audio     string      `json:"audio,omitempty"` // Base64 encoded audio data
	Format    string      `json:"format,omitempty"` // Audio format (e.g., "audio/mp4", "audio/webm")
	Text      string      `json:"text,omitempty"`
	Timestamp float64     `json:"timestamp,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// Hub manages all WebSocket connections
type Hub struct {
	clients    map[string]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// NewHub creates a new hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client.ID] = client
			h.mutex.Unlock()
			log.Printf("Client %s registered", client.ID)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("Client %s unregistered", client.ID)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for id, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, id)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// Global hub instance
var hub = NewHub()

// HandleWebSocket handles WebSocket connections for live transcription
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		ID:   fmt.Sprintf("client_%d", time.Now().UnixNano()),
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	// Register client with hub
	hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		c.Conn.Close()
		if c.WhisperConn != nil {
			c.WhisperConn.Close()
		}
	}()

	c.Conn.SetReadLimit(1024 * 1024) // 1MB max message size for audio data
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			log.Printf("Client %s WebSocket connection closed: %v", c.ID, err)
			break
		}

		// Parse the message
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			continue
		}

		log.Printf("Client %s received message type: %s", c.ID, wsMsg.Type)

		// Handle different message types
		switch wsMsg.Type {
		case "init":
			// Initialize client with model settings
			c.Model = wsMsg.ModelSize
			c.Language = wsMsg.Language
			c.Translate = wsMsg.Translate
			
			// Connect to FastAPI WhisperLive server
			whisperURL := "ws://localhost:9090/ws/transcribe"
			whisperConn, _, err := websocket.DefaultDialer.Dial(whisperURL, nil)
			if err != nil {
				log.Printf("Failed to connect to WhisperLive server: %v", err)
				response := WSMessage{
					Type:  "error",
					Error: "Failed to connect to transcription service",
				}
				responseBytes, _ := json.Marshal(response)
				c.Send <- responseBytes
				return
			}
			
			c.WhisperConn = whisperConn
			
			// Forward init message to WhisperLive
			initMsg := map[string]interface{}{
				"type":       "init",
				"client_id":  wsMsg.ClientID,
				"model_size": wsMsg.ModelSize,
				"language":   wsMsg.Language,
				"translate":  wsMsg.Translate,
			}
			initBytes, _ := json.Marshal(initMsg)
			if err := whisperConn.WriteMessage(websocket.TextMessage, initBytes); err != nil {
				log.Printf("Failed to send init to WhisperLive: %v", err)
				return
			}
			
			// Start goroutine to forward messages from WhisperLive to client
			go c.forwardFromWhisperLive()
			
			// Send ready message back to client with client's actual ID
			response := WSMessage{
				Type:      "ready",
				ClientID:  wsMsg.ClientID, // Use the client's provided ID
				ModelSize: wsMsg.ModelSize,
				Language:  wsMsg.Language,
				Translate: wsMsg.Translate,
			}
			responseBytes, _ := json.Marshal(response)
			c.Send <- responseBytes

		case "audio_data":
			// Forward audio data to WhisperLive service
			if c.WhisperConn != nil {
				log.Printf("Client %s sending audio data of length %d", c.ID, len(wsMsg.Audio))
				audioMsg := map[string]interface{}{
					"type":  "audio_data",
					"audio": wsMsg.Audio,
				}
				// Forward the format if provided
				if wsMsg.Format != "" {
					audioMsg["format"] = wsMsg.Format
					log.Printf("Client %s forwarding audio format: %s", c.ID, wsMsg.Format)
				}
				audioBytes, _ := json.Marshal(audioMsg)
				if err := c.WhisperConn.WriteMessage(websocket.TextMessage, audioBytes); err != nil {
					log.Printf("Failed to send audio to WhisperLive: %v", err)
					// Don't return here, just log the error and continue
					// The connection might recover or the client might retry
				} else {
					log.Printf("Client %s successfully sent audio data to WhisperLive", c.ID)
				}
			} else {
				log.Printf("No WhisperLive connection for client %s", c.ID)
				// Send error back to client but don't close the connection
				response := WSMessage{
					Type:  "error",
					Error: "No transcription service connection",
				}
				responseBytes, _ := json.Marshal(response)
				c.Send <- responseBytes
			}

		case "stop":
			// Client is stopping transcription
			log.Printf("Client %s stopping transcription", c.ID)
			// Forward stop message to WhisperLive
			if c.WhisperConn != nil {
				stopMsg := map[string]interface{}{
					"type": "stop",
				}
				stopBytes, _ := json.Marshal(stopMsg)
				if err := c.WhisperConn.WriteMessage(websocket.TextMessage, stopBytes); err != nil {
					log.Printf("Failed to send stop message to WhisperLive: %v", err)
				}
				// Close the WhisperLive connection gracefully
				c.WhisperConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				c.WhisperConn.Close()
				c.WhisperConn = nil // Set to nil to prevent double-close in defer
			}
			// Send confirmation back to client
			response := WSMessage{
				Type: "stopped",
			}
			responseBytes, _ := json.Marshal(response)
			c.Send <- responseBytes
			return

		default:
			log.Printf("Unknown message type: %s", wsMsg.Type)
		}
	}
}

// forwardFromWhisperLive forwards messages from WhisperLive server to the client
func (c *Client) forwardFromWhisperLive() {
	defer func() {
		if c.WhisperConn != nil {
			c.WhisperConn.Close()
			c.WhisperConn = nil
		}
	}()

	for {
		_, message, err := c.WhisperConn.ReadMessage()
		if err != nil {
			log.Printf("Error reading from WhisperLive: %v", err)
			// Send error to client but don't break the main WebSocket connection
			response := WSMessage{
				Type:  "error",
				Error: "Transcription service connection lost",
			}
			responseBytes, _ := json.Marshal(response)
			c.Send <- responseBytes
			break
		}

		log.Printf("Client %s received message from WhisperLive: %s", c.ID, string(message))
		
		// Forward the message directly to the client
		c.Send <- message
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// StartHub starts the WebSocket hub
func StartHub() {
	go hub.Run()
	log.Println("WebSocket hub started")
} 
package api

import (
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ProxyRealtimeTranscription proxies WebSocket connections to the WhisperLiveKit service
func (h *Handler) ProxyRealtimeTranscription(c *gin.Context) {
	// Get WhisperLiveKit URL from environment or default
	targetHost := os.Getenv("WHISPERLIVEKIT_HOST")
	if targetHost == "" {
		targetHost = "localhost:8000"
	}

	targetURL := url.URL{Scheme: "ws", Host: targetHost, Path: "/asr"}

	// Upgrade client connection to WebSocket
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// Use package level logger if available, or just log to stdout/stderr
		// Assuming scriberr/pkg/logger has a global logger or similar
		// For now, let's just return as Upgrade handles the error response
		return
	}
	defer clientConn.Close()

	// Connect to backend
	backendConn, _, err := websocket.DefaultDialer.Dial(targetURL.String(), nil)
	if err != nil {
		clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Backend unavailable"))
		return
	}
	defer backendConn.Close()

	// Proxy loop
	errChan := make(chan error, 2)

	// Client -> Backend
	go func() {
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := backendConn.WriteMessage(msgType, msg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Backend -> Client
	go func() {
		for {
			msgType, msg, err := backendConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Wait for error (disconnect)
	<-errChan
}

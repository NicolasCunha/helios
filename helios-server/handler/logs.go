// Package handler provides HTTP handlers for the Helios API.
package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"nfcunha/helios/core/service"
)

// LogHandler handles log-related HTTP requests.
type LogHandler struct {
	logService *service.LogService
	upgrader   websocket.Upgrader
}

// NewLogHandler creates a new log handler.
func NewLogHandler(logService *service.LogService) *LogHandler {
	return &LogHandler{
		logService: logService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}
}

// StreamLogs handles GET /helios/containers/:id/logs (WebSocket)
// Query parameters:
//   - follow: boolean (follow log output)
//   - tail: string (number of lines from end, default "all")
//   - timestamps: boolean (show timestamps)
//   - since: string (show logs since timestamp)
//   - until: string (show logs before timestamp)
func (h *LogHandler) StreamLogs(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Container ID is required",
		})
		return
	}

	// Parse query parameters
	opts := service.LogStreamOptions{
		Follow:     c.Query("follow") == "true",
		Tail:       c.DefaultQuery("tail", "100"),
		Timestamps: c.Query("timestamps") == "true",
		Since:      c.Query("since"),
		Until:      c.Query("until"),
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Set write deadline for initial message
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := conn.WriteMessage(websocket.TextMessage, []byte("Connected to log stream\n")); err != nil {
		log.Printf("Failed to write welcome message: %v", err)
		return
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Handle WebSocket close messages
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	// Create a custom writer that sends to WebSocket
	writer := &websocketWriter{
		conn: conn,
	}

	// Start streaming logs
	errChan, err := h.logService.StreamLogs(ctx, containerID, opts, writer)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v\n", err)))
		return
	}

	// Wait for completion or error
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			log.Printf("Log streaming error: %v", err)
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\nError: %v\n", err)))
		}
	case <-ctx.Done():
		log.Println("Log streaming cancelled")
	}
}

// DownloadLogs handles GET /helios/containers/:id/logs/download
// Downloads container logs as a ZIP file.
// Query parameters:
//   - tail: string (number of lines from end, default "all")
//   - timestamps: boolean (include timestamps)
func (h *LogHandler) DownloadLogs(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Container ID is required",
		})
		return
	}

	// Set headers for download
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=container-%s-logs.zip", containerID[:12]))

	// Create archive and stream to response
	if err := h.logService.CreateLogArchive(c.Request.Context(), containerID, c.Writer); err != nil {
		log.Printf("Failed to create log archive: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to create log archive",
			"detail": err.Error(),
		})
		return
	}
}

// websocketWriter implements io.Writer for WebSocket text messages.
type websocketWriter struct {
	conn *websocket.Conn
}

func (w *websocketWriter) Write(p []byte) (n int, err error) {
	// Set write deadline
	w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	// Send as text message
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *websocketWriter) Flush() error {
	// WebSocket messages are flushed immediately
	return nil
}

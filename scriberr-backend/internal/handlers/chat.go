package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"scriberr-backend/internal/database"
	"scriberr-backend/internal/models"
	"scriberr-backend/internal/summary_tasks"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// CreateChatSession creates a new chat session for an audio transcript
func CreateChatSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateChatSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AudioID == "" || req.Title == "" || req.Model == "" {
		writeJSONError(w, "Missing required fields: audio_id, title, model", http.StatusBadRequest)
		return
	}

	// Verify the audio record exists and has a transcript
	db := database.GetDB()
	var transcript sql.NullString
	err := db.QueryRow("SELECT transcript FROM audio_records WHERE id = ?", req.AudioID).Scan(&transcript)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, "Audio record not found", http.StatusNotFound)
		} else {
			log.Printf("Error checking audio record: %v", err)
			writeJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	if !transcript.Valid || transcript.String == "" || transcript.String == "{}" {
		writeJSONError(w, "Audio record has no transcript. Please transcribe the audio first.", http.StatusBadRequest)
		return
	}

	// Create the chat session
	sessionID := uuid.New().String()
	now := time.Now().UTC()

	query := `INSERT INTO chat_sessions (id, audio_id, title, model, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing chat session insert: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(sessionID, req.AudioID, req.Title, req.Model, now, now)
	if err != nil {
		log.Printf("Error creating chat session: %v", err)
		writeJSONError(w, "Failed to create chat session", http.StatusInternalServerError)
		return
	}

	// Create initial system message with transcript context
	transcriptText := extractTranscriptText(transcript.String)
	systemPrompt := fmt.Sprintf(`You are a helpful assistant that can answer questions about the following transcript. 
	
TRANSCRIPT:
%s

Please answer questions about this transcript accurately and helpfully. If asked about something not in the transcript, say so.`, transcriptText)

	// Insert system message
	messageID := uuid.New().String()
	messageQuery := `INSERT INTO chat_messages (id, session_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)`
	messageStmt, err := db.Prepare(messageQuery)
	if err != nil {
		log.Printf("Error preparing message insert: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer messageStmt.Close()

	_, err = messageStmt.Exec(messageID, sessionID, "system", systemPrompt, now)
	if err != nil {
		log.Printf("Error creating system message: %v", err)
		writeJSONError(w, "Failed to create chat session", http.StatusInternalServerError)
		return
	}

	session := models.ChatSession{
		ID:        sessionID,
		AudioID:   req.AudioID,
		Title:     req.Title,
		Model:     req.Model,
		CreatedAt: now,
		UpdatedAt: now,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// GetChatSessions retrieves all chat sessions for a specific audio record
func GetChatSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	audioID := r.URL.Query().Get("audio_id")
	if audioID == "" {
		writeJSONError(w, "Missing audio_id parameter", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	query := `SELECT id, audio_id, title, model, created_at, updated_at FROM chat_sessions WHERE audio_id = ? ORDER BY updated_at DESC`
	rows, err := db.Query(query, audioID)
	if err != nil {
		log.Printf("Error querying chat sessions: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sessions []models.ChatSession
	for rows.Next() {
		var session models.ChatSession
		err := rows.Scan(&session.ID, &session.AudioID, &session.Title, &session.Model, &session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			log.Printf("Error scanning chat session: %v", err)
			continue
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating chat sessions: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sessions)
}

// GetChatMessages retrieves all messages for a specific chat session
func GetChatMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeJSONError(w, "Missing session_id parameter", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	query := `SELECT id, session_id, role, content, created_at FROM chat_messages WHERE session_id = ? ORDER BY created_at ASC`
	rows, err := db.Query(query, sessionID)
	if err != nil {
		log.Printf("Error querying chat messages: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var message models.ChatMessage
		err := rows.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &message.CreatedAt)
		if err != nil {
			log.Printf("Error scanning chat message: %v", err)
			continue
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating chat messages: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(messages)
}

// SendChatMessage sends a message in a chat session and gets a response
func SendChatMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" || req.Message == "" {
		writeJSONError(w, "Missing required fields: session_id, message", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// Get session details and model
	var session models.ChatSession
	err := db.QueryRow("SELECT id, audio_id, title, model, created_at, updated_at FROM chat_sessions WHERE id = ?", req.SessionID).Scan(
		&session.ID, &session.AudioID, &session.Title, &session.Model, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, "Chat session not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting chat session: %v", err)
			writeJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Use provided model or fall back to session model
	model := req.Model
	if model == "" {
		model = session.Model
	}

	// Get all messages for context
	messages, err := getChatMessagesForSession(db, req.SessionID)
	if err != nil {
		log.Printf("Error getting chat messages: %v", err)
		writeJSONError(w, "Failed to get chat history", http.StatusInternalServerError)
		return
	}

	// Add the new user message
	userMessageID := uuid.New().String()
	now := time.Now().UTC()
	
	messageQuery := `INSERT INTO chat_messages (id, session_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)`
	messageStmt, err := db.Prepare(messageQuery)
	if err != nil {
		log.Printf("Error preparing message insert: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer messageStmt.Close()

	_, err = messageStmt.Exec(userMessageID, req.SessionID, "user", req.Message, now)
	if err != nil {
		log.Printf("Error inserting user message: %v", err)
		writeJSONError(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Generate AI response
	response, err := generateChatResponse(messages, req.Message, model)
	if err != nil {
		log.Printf("Error generating chat response: %v", err)
		writeJSONError(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	// Save AI response
	assistantMessageID := uuid.New().String()
	_, err = messageStmt.Exec(assistantMessageID, req.SessionID, "assistant", response, now)
	if err != nil {
		log.Printf("Error inserting assistant message: %v", err)
		writeJSONError(w, "Failed to save response", http.StatusInternalServerError)
		return
	}

	// Update session timestamp
	updateQuery := `UPDATE chat_sessions SET updated_at = ? WHERE id = ?`
	_, err = db.Exec(updateQuery, now, req.SessionID)
	if err != nil {
		log.Printf("Error updating session timestamp: %v", err)
		// Don't fail the request for this error
	}

	chatResponse := models.ChatResponse{
		MessageID: assistantMessageID,
		Content:   response,
		Model:     model,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(chatResponse)
}

// DeleteChatSession deletes a chat session and all its messages
func DeleteChatSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeJSONError(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	db := database.GetDB()

	// Delete the session (messages will be deleted via CASCADE)
	query := "DELETE FROM chat_sessions WHERE id = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing delete statement: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(sessionID)
	if err != nil {
		log.Printf("Error deleting chat session: %v", err)
		writeJSONError(w, "Failed to delete chat session", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		writeJSONError(w, "Failed to verify deletion", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		writeJSONError(w, "Chat session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper function to get all messages for a session
func getChatMessagesForSession(db *sql.DB, sessionID string) ([]models.ChatMessage, error) {
	query := `SELECT id, session_id, role, content, created_at FROM chat_messages WHERE session_id = ? ORDER BY created_at ASC`
	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var message models.ChatMessage
		err := rows.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &message.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

// Helper function to extract transcript text from JSON
func extractTranscriptText(transcriptJSON string) string {
	var transcript models.JSONTranscript
	if err := json.Unmarshal([]byte(transcriptJSON), &transcript); err != nil {
		return "Error parsing transcript"
	}

	var text strings.Builder
	for _, segment := range transcript.Segments {
		if segment.Speaker != "" {
			text.WriteString(fmt.Sprintf("[Speaker %s]: ", segment.Speaker))
		}
		text.WriteString(segment.Text)
		text.WriteString(" ")
	}

	return strings.TrimSpace(text.String())
}

// Helper function to generate chat response using OpenAI or Ollama
func generateChatResponse(messages []models.ChatMessage, userMessage, model string) (string, error) {
	// Determine which service to use
	useOllama := strings.HasPrefix(model, "ollama:")
	useOpenAI := !useOllama

	if useOpenAI {
		// Use OpenAI
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return "", fmt.Errorf("OpenAI API key not configured")
		}

		client := openai.NewClient(apiKey)
		
		// Convert messages to OpenAI format
		var openaiMessages []openai.ChatCompletionMessage
		for _, msg := range messages {
			role := openai.ChatMessageRoleUser
			if msg.Role == "assistant" {
				role = openai.ChatMessageRoleAssistant
			} else if msg.Role == "system" {
				role = openai.ChatMessageRoleSystem
			}
			openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
				Role:    role,
				Content: msg.Content,
			})
		}

		// Add the new user message
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userMessage,
		})

		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    model,
				Messages: openaiMessages,
			},
		)

		if err != nil {
			return "", fmt.Errorf("OpenAI API error: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("OpenAI returned no choices")
		}

		return resp.Choices[0].Message.Content, nil

	} else {
		// Use Ollama
		ollamaClient := summary_tasks.NewOllamaClient()
		if !ollamaClient.IsAvailable(context.Background()) {
			return "", fmt.Errorf("Ollama is not available")
		}

		ollamaModel := strings.TrimPrefix(model, "ollama:")
		
		// Build conversation context
		var conversation strings.Builder
		for _, msg := range messages {
			if msg.Role == "system" {
				conversation.WriteString("System: " + msg.Content + "\n\n")
			} else if msg.Role == "user" {
				conversation.WriteString("User: " + msg.Content + "\n")
			} else if msg.Role == "assistant" {
				conversation.WriteString("Assistant: " + msg.Content + "\n")
			}
		}
		conversation.WriteString("User: " + userMessage + "\n")
		conversation.WriteString("Assistant: ")

		options := &summary_tasks.OllamaOptions{
			Temperature: 0.7,
			TopP:        0.9,
		}

		response, err := ollamaClient.GenerateText(context.Background(), ollamaModel, conversation.String(), options)
		if err != nil {
			return "", fmt.Errorf("Ollama API error: %w", err)
		}

		return response, nil
	}
} 
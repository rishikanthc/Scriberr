package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"scriberr-backend/internal/database"
	"scriberr-backend/internal/middleware"

	"golang.org/x/crypto/bcrypt"
)

// UserCredentials defines the structure for login and registration requests.
type UserCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register handles user registration.
// For simplicity, this is an open endpoint. In a real application,
// you might want to restrict this to an initial setup or an admin-only function.
func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if creds.Username == "" || creds.Password == "" {
		writeJSONError(w, "Username and password cannot be empty", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		writeJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	db := database.GetDB()
	query := "INSERT INTO users (username, password_hash) VALUES (?, ?)"
	stmt, err := db.Prepare(query)
	if err != nil {
		log.Printf("Error preparing registration statement: %v", err)
		writeJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(creds.Username, string(hashedPassword))
	if err != nil {
		// This could fail if the username is already taken (assuming a UNIQUE constraint)
		log.Printf("Error executing registration insert: %v", err)
		writeJSONError(w, "Failed to register user. Username might already exist.", http.StatusConflict)
		return
	}

	log.Printf("User '%s' registered successfully.", creds.Username)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

// Login handles the user login request.
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	db := database.GetDB()
	var userID int
	var hashedPassword string
	query := "SELECT id, password_hash FROM users WHERE username = ?"
	err := db.QueryRow(query, creds.Username).Scan(&userID, &hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONError(w, "Invalid username or password", http.StatusUnauthorized)
		} else {
			log.Printf("Error querying user for login: %v", err)
			writeJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password)); err != nil {
		writeJSONError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	session, _ := middleware.Store.Get(r, middleware.SessionName)
	session.Values["authenticated"] = true
	session.Values["user_id"] = userID
	session.Options.HttpOnly = true
	session.Options.MaxAge = 86400 * 7 // 7 days
	session.Options.Path = "/"
	session.Options.SameSite = http.SameSiteLaxMode
	// For development over HTTP, Secure must be false.
	session.Options.Secure = false

	if err := session.Save(r, w); err != nil {
		log.Printf("Error saving session: %v", err)
		writeJSONError(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	log.Printf("User ID %d logged in successfully.", userID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged in successfully"})
}

// Logout handles the user logout request.
func Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, _ := middleware.Store.Get(r, middleware.SessionName)

	// Clear the session by setting authenticated to false and max age to -1
	session.Values["authenticated"] = false
	session.Options.MaxAge = -1
	// To properly clear a cookie, its attributes must match.
	session.Options.Path = "/"
	session.Options.SameSite = http.SameSiteLaxMode
	session.Options.Secure = false

	if err := session.Save(r, w); err != nil {
		log.Printf("Error saving session on logout: %v", err)
		writeJSONError(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

// CheckAuthStatus checks if the user is currently authenticated.
// This endpoint is protected by the AuthFunc middleware, so if this handler
// is reached, the user is considered authenticated.
func CheckAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If this handler is reached, the AuthFunc middleware has already confirmed authentication.
	// We just need to return a success status.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": true})
}

// CheckAuthRedirect is a simple endpoint that returns a redirect response
// when the user is not authenticated. This can be used by the frontend
// to handle authentication redirects more reliably.
func CheckAuthRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := middleware.Store.Get(r, middleware.SessionName)
	if err != nil {
		// Session error, redirect to login
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"redirect": "/login"})
		return
	}

	// Check if the 'authenticated' flag is present and set to true in the session.
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		// Not authenticated, redirect to login
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"redirect": "/login"})
		return
	}

	// Authenticated, return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": true})
}

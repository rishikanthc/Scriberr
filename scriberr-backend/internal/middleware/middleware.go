package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

// Store is the global session store. It is initialized in the init function.
var Store *sessions.CookieStore

// SessionName is the name used for the session cookie.
const SessionName = "scriberr-session"

func init() {
	// For production, this key must be loaded from a secure source (e.g., environment variable).
	// A new random key can be generated with a command like: `openssl rand -base64 32`
	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		log.Println("WARNING: SESSION_KEY environment variable not set. Using a temporary, insecure key. Please set a strong, random key for production.")
		// This key is for development convenience only. It should not be used in production.
		sessionKey = "a-very-insecure-temporary-key-for-development"
	}
	Store = sessions.NewCookieStore([]byte(sessionKey))
}

// writeJSONError sends a JSON formatted error message. This is a helper function
// used by the middleware to provide consistent error responses.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// AuthFunc is a middleware that wraps a single http.HandlerFunc to protect it.
// It provides a clear and explicit way to protect a specific endpoint.
func AuthFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := Store.Get(r, SessionName)
		if err != nil {
			// This can happen if the cookie is invalid or decoding fails.
			log.Printf("Session retrieval error: %v. This might be due to a malformed cookie.", err)
			writeJSONError(w, "Invalid session. Please log in again.", http.StatusUnauthorized)
			return
		}

		// Check if the 'authenticated' flag is present and set to true in the session.
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			writeJSONError(w, "Authentication required. Please log in.", http.StatusUnauthorized)
			return
		}

		// If authenticated, call the original handler function.
		next.ServeHTTP(w, r)
	}
}

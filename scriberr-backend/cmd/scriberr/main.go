package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"scriberr-backend/internal/database"
	"scriberr-backend/internal/handlers"
	"scriberr-backend/internal/middleware"
	"scriberr-backend/internal/summary_tasks"
	"scriberr-backend/internal/tasks"

	"github.com/rs/cors"
)

//go:embed all:embedded_assets/*
var rawFSFromEmbed embed.FS // This FS will contain the embedded_assets directory.

// spaFileSystem wraps an http.FileSystem to implement SPA routing.
// If a requested file is not found, it serves 'index.html' instead.
type spaFileSystem struct {
	contentRoot http.FileSystem
}

// Open implements the http.FileSystem interface.
func (sfs spaFileSystem) Open(name string) (http.File, error) {
	f, err := sfs.contentRoot.Open(name)
	// If the file exists, serve it.
	if err == nil {
		return f, nil
	}
	// If the file does not exist, this is the SPA fallback case.
	// Serve the index.html from the root of the content filesystem.
	if os.IsNotExist(err) {
		log.Printf("SPA Fallback: Requested path '%s' not found. Serving 'index.html'.", name)
		return sfs.contentRoot.Open("index.html")
	}
	// For any other errors, return them.
	return nil, err
}

func main() {
	log.Println("Starting Scriberr server...")

	// Initialize the database connection and run migrations.
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize the background job worker.
	tasks.Init()

	// Initialize the summarization job worker.
	summary_tasks.Init()

	// --- Frontend File Server Setup ---
	// Create an fs.FS that is rooted at the "embedded_assets" directory
	// within the raw embedded filesystem. This makes 'index.html' available at the root.
	contentFS, err := fs.Sub(rawFSFromEmbed, "embedded_assets")
	if err != nil {
		log.Fatalf("Failed to create sub FS for embedded assets: %v", err)
	}

	// Wrap the correctly rooted contentFS with our SPA handler logic.
	spaFS := spaFileSystem{contentRoot: http.FS(contentFS)}
	fileServer := http.FileServer(spaFS)

	// --- HTTP Route Handling ---
	// Create a master router for the application.
	mux := http.NewServeMux()

	// Public authentication routes. These do not have the auth middleware.
	mux.HandleFunc("POST /api/auth/register", handlers.Register)
	mux.HandleFunc("POST /api/auth/login", handlers.Login)
	mux.HandleFunc("POST /api/auth/logout", handlers.Logout)

	// Protected API routes. Each handler is wrapped with the AuthFunc middleware.
	mux.HandleFunc("GET /api/auth/status", handlers.CheckAuthStatus)
	mux.HandleFunc("GET /api/auth/check", handlers.CheckAuthRedirect)
	mux.HandleFunc("POST /api/audio", middleware.AuthFunc(handlers.CreateAudio))
	mux.HandleFunc("POST /api/youtube", middleware.AuthFunc(handlers.DownloadYouTubeAudio))
	mux.HandleFunc("GET /api/audio/all", middleware.AuthFunc(handlers.GetAllAudioRecords))
	mux.HandleFunc("GET /api/audio/file/{id}", middleware.AuthFunc(handlers.GetAudioFile))
	mux.HandleFunc("GET /api/audio/{id}", middleware.AuthFunc(handlers.GetAudioRecord))
	mux.HandleFunc("GET /api/audio/{id}/transcript/download", middleware.AuthFunc(handlers.DownloadTranscript))
	mux.HandleFunc("PUT /api/audio/{id}", middleware.AuthFunc(handlers.UpdateAudioTitle))
	mux.HandleFunc("DELETE /api/audio/{id}", middleware.AuthFunc(handlers.DeleteAudio))
	mux.HandleFunc("POST /api/transcribe", middleware.AuthFunc(handlers.HandleTranscribe))
	mux.HandleFunc("GET /api/transcribe/status/{jobid}", middleware.AuthFunc(handlers.GetTranscriptionStatus))
	mux.HandleFunc("GET /api/transcribe/jobs/active", middleware.AuthFunc(handlers.GetActiveJobs))
	mux.HandleFunc("DELETE /api/transcribe/job/{jobid}", middleware.AuthFunc(handlers.TerminateJob))

	// Summary Template routes
	mux.HandleFunc("POST /api/summary-templates", middleware.AuthFunc(handlers.CreateOrUpdateSummaryTemplate))
	mux.HandleFunc("GET /api/summary-templates", middleware.AuthFunc(handlers.GetAllSummaryTemplates))
	mux.HandleFunc("GET /api/summary-templates/{id}", middleware.AuthFunc(handlers.GetSummaryTemplate))
	mux.HandleFunc("DELETE /api/summary-templates/{id}", middleware.AuthFunc(handlers.DeleteSummaryTemplate))

	// Summarization routes
	mux.HandleFunc("POST /api/summarize", middleware.AuthFunc(handlers.SummarizeAudio))
	mux.HandleFunc("GET /api/summarize/status/job/{jobid}", middleware.AuthFunc(handlers.GetSummarizeStatus))
	mux.HandleFunc("GET /api/summarize/status/audio/{id}", middleware.AuthFunc(handlers.GetSummarizeStatusByAudioID))
	mux.HandleFunc("GET /api/models", middleware.AuthFunc(handlers.GetAvailableModels))

	// The root handler serves the frontend SPA.
	// This must be registered after all other routes to act as a catch-all.
	mux.Handle("/", fileServer)

	// --- CORS and Server Setup ---
	// Setup CORS middleware. For a same-origin SPA this isn't strictly necessary,
	// but it's good practice and essential for local development if the
	// frontend and backend run on different ports.
	c := cors.New(cors.Options{
		// In production, this should be restricted to your frontend's actual domain.
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-Requested-With", "Upgrade", "Connection"},
		AllowCredentials: true,
	})

	// Apply CORS for all routes
	handler := c.Handler(mux)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"scriberr/internal/auth"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var staticFiles embed.FS

// GetStaticHandler returns a handler for serving embedded static files
func GetStaticHandler() http.Handler {
	// Get the dist subdirectory from embedded files
	distFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		panic("failed to get dist subdirectory: " + err.Error())
	}

	return http.FileServer(http.FS(distFS))
}

// GetIndexHTML returns the index.html content
func GetIndexHTML() ([]byte, error) {
	return staticFiles.ReadFile("dist/index.html")
}

// SetupStaticRoutes configures static file serving in Gin
func SetupStaticRoutes(router *gin.Engine, authService *auth.AuthService) {

	// Serve static assets (CSS, JS, images) directly from embedded filesystem
	router.GET("/assets/*filepath", func(c *gin.Context) {
		// Extract the file path
		filepath := c.Param("filepath")
		// Remove leading slash if present
		if filepath[0] == '/' {
			filepath = filepath[1:]
		}
		fullPath := "assets/" + filepath
		
		// Try to read the file from embedded filesystem
		fileContent, err := staticFiles.ReadFile("dist/" + fullPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		
		// Set appropriate content type based on file extension
		if strings.Contains(fullPath, ".css") {
			c.Data(http.StatusOK, "text/css", fileContent)
		} else if strings.Contains(fullPath, ".js") {
			c.Data(http.StatusOK, "application/javascript", fileContent)
		} else {
			c.Data(http.StatusOK, "application/octet-stream", fileContent)
		}
	})

	// Serve vite.svg
	router.GET("/vite.svg", func(c *gin.Context) {
		fileContent, err := staticFiles.ReadFile("dist/vite.svg")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/svg+xml", fileContent)
	})


	// Serve index.html for root and any unmatched routes (SPA behavior)
	router.NoRoute(func(c *gin.Context) {
		// For API routes, return 404
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

		// For all other routes, serve the React app
		// The React app will handle authentication client-side
		indexHTML, err := GetIndexHTML()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error loading page")
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}
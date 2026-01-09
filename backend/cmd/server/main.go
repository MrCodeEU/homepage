package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrcodeeu/homepage/internal/config"
	"github.com/mrcodeeu/homepage/internal/storage"
)

//go:embed all:static
var staticFiles embed.FS

// Global data loader (initialized in main)
var (
	dataLoader *storage.DataLoader
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize data loader
	dataLoader = storage.NewDataLoader("")
	log.Printf("Data loader initialized")

	// Create HTTP server
	mux := http.NewServeMux()

	// Serve embedded static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal("Failed to load static files:", err)
	}

	// Debug: List embedded files
	log.Println("Embedded static files:")
	if err := fs.WalkDir(staticFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			log.Printf("  - %s", path)
		}
		return nil
	}); err != nil {
		log.Printf("Warning: Failed to list static files: %v", err)
	}

	// Serve static files directly
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// API endpoints
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/cv", handleCV)
	mux.HandleFunc("/api/projects", handleProjects)
	mux.HandleFunc("/api/strava", handleStrava)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      corsMiddleware(loggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// Middleware for CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Allow localhost for development and same-origin for production
		if origin == "http://localhost:5173" || origin == "" {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware for logging
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// CV endpoint - loads LinkedIn data
func handleCV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to load LinkedIn data
	linkedInData, err := dataLoader.LoadLinkedIn()
	if err != nil {
		log.Printf("Error loading LinkedIn data: %v", err)
		http.Error(w, "Failed to load CV data", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(linkedInData); err != nil {
		http.Error(w, "Failed to encode CV data", http.StatusInternalServerError)
		log.Printf("Error encoding CV response: %v", err)
	}
}

// Projects endpoint - loads GitHub data
func handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Load GitHub projects data
	projects, err := dataLoader.LoadGitHub()
	if err != nil {
		log.Printf("Error loading GitHub data: %v", err)
		http.Error(w, "Failed to load projects", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(projects); err != nil {
		http.Error(w, "Failed to encode projects data", http.StatusInternalServerError)
		log.Printf("Error encoding projects response: %v", err)
	}
}

// Strava endpoint - loads Strava data
func handleStrava(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Load Strava data
	stravaData, err := dataLoader.LoadStrava()
	if err != nil {
		log.Printf("Error loading Strava data: %v", err)
		http.Error(w, "Failed to load Strava data", http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(stravaData); err != nil {
		http.Error(w, "Failed to encode Strava data", http.StatusInternalServerError)
		log.Printf("Error encoding Strava response: %v", err)
	}
}

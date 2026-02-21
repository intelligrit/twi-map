package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/robertmeta/twi-map/internal/store"
)

//go:embed all:static
var staticFS embed.FS

// Server serves the interactive map web app and API.
type Server struct {
	Store *store.Store
	Addr  string
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/chapters", s.handleChapters)
	mux.HandleFunc("/api/locations", s.handleLocations)
	mux.HandleFunc("/api/relationships", s.handleRelationships)
	mux.HandleFunc("/api/coordinates", s.handleCoordinates)
	mux.HandleFunc("/api/containment", s.handleContainment)

	// Static files
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("creating sub filesystem: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticSub)))

	fmt.Printf("Serving at http://%s\n", s.Addr)
	return http.ListenAndServe(s.Addr, mux)
}

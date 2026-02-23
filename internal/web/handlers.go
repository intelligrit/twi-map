package web

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *Server) handleChapters(w http.ResponseWriter, r *http.Request) {
	toc, err := s.Store.ReadTOC()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	volume := r.URL.Query().Get("volume")
	if volume != "" {
		var filtered []any
		for _, ch := range toc.Chapters {
			if ch.Volume == volume {
				filtered = append(filtered, ch)
			}
		}
		writeJSON(w, filtered)
		return
	}

	writeJSON(w, toc.Chapters)
}

func (s *Server) handleLocations(w http.ResponseWriter, r *http.Request) {
	data, err := s.Store.ReadAggregated()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by through parameter (spoiler-free: only show locations first mentioned at or before this chapter)
	throughStr := r.URL.Query().Get("through")
	if throughStr != "" {
		through, err := strconv.Atoi(throughStr)
		if err != nil {
			http.Error(w, "invalid 'through' parameter", http.StatusBadRequest)
			return
		}

		var filtered []any
		for _, loc := range data.Locations {
			if loc.FirstChapterIndex <= through {
				filtered = append(filtered, loc)
			}
		}
		writeJSON(w, filtered)
		return
	}

	writeJSON(w, data.Locations)
}

func (s *Server) handleRelationships(w http.ResponseWriter, r *http.Request) {
	data, err := s.Store.ReadAggregated()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	throughStr := r.URL.Query().Get("through")
	if throughStr != "" {
		through, err := strconv.Atoi(throughStr)
		if err != nil {
			http.Error(w, "invalid 'through' parameter", http.StatusBadRequest)
			return
		}

		var filtered []any
		for _, rel := range data.Relationships {
			if rel.FirstChapterIndex <= through {
				filtered = append(filtered, rel)
			}
		}
		writeJSON(w, filtered)
		return
	}

	writeJSON(w, data.Relationships)
}

func (s *Server) handleCoordinates(w http.ResponseWriter, r *http.Request) {
	coords, err := s.Store.ReadCoordinates()
	if err != nil {
		writeJSON(w, []any{})
		return
	}
	writeJSON(w, coords)
}

func (s *Server) handleContainment(w http.ResponseWriter, r *http.Request) {
	data, err := s.Store.ReadAggregated()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, data.Containment)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	// Wildcard CORS — this is a local development tool, not a public API.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if v == nil {
		_, _ = w.Write([]byte("[]"))
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

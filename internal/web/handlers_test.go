package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/intelligrit/twi-map/internal/model"
	"github.com/intelligrit/twi-map/internal/store"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "twi-map-web-test-"+t.Name())
	os.RemoveAll(dir)
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	return &Server{Store: s, Addr: "localhost:0"}
}

func TestHandleChapters(t *testing.T) {
	srv := testServer(t)

	toc := &model.TOC{
		Chapters: []model.Chapter{
			{Index: 0, WebTitle: "1.00", URL: "https://example.com/1-00", Volume: "vol-1", Slug: "1-00"},
			{Index: 1, WebTitle: "1.01", URL: "https://example.com/1-01", Volume: "vol-1", Slug: "1-01"},
		},
	}
	if err := srv.Store.WriteTOC(toc); err != nil {
		t.Fatalf("writing TOC: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/chapters", nil)
	w := httptest.NewRecorder()
	srv.handleChapters(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var chapters []model.Chapter
	if err := json.NewDecoder(w.Body).Decode(&chapters); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(chapters))
	}
}

func TestHandleLocationsWithThrough(t *testing.T) {
	srv := testServer(t)

	data := &model.AggregatedData{
		AggregatedAt: "2025-01-01",
		Locations: []model.AggregatedLocation{
			{ID: "liscor", Name: "Liscor", Type: "city", FirstChapterIndex: 0, MentionCount: 50},
			{ID: "pallass", Name: "Pallass", Type: "city", FirstChapterIndex: 100, MentionCount: 30},
		},
	}
	if err := srv.Store.WriteAggregated(data); err != nil {
		t.Fatalf("writing aggregated: %v", err)
	}

	// Request with through=50 should only return Liscor
	req := httptest.NewRequest("GET", "/api/locations?through=50", nil)
	w := httptest.NewRecorder()
	srv.handleLocations(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var locs []model.AggregatedLocation
	if err := json.NewDecoder(w.Body).Decode(&locs); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(locs) != 1 {
		t.Fatalf("expected 1 filtered location, got %d", len(locs))
	}
	if locs[0].Name != "Liscor" {
		t.Errorf("expected Liscor, got %q", locs[0].Name)
	}
}

func TestHandleLocationsInvalidThrough(t *testing.T) {
	srv := testServer(t)

	req := httptest.NewRequest("GET", "/api/locations?through=abc", nil)
	w := httptest.NewRecorder()
	srv.handleLocations(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleCoordinates(t *testing.T) {
	srv := testServer(t)

	coord := model.Coordinate{LocationID: "liscor", X: 240, Y: -20, Confidence: "estimated"}
	if err := srv.Store.WriteCoordinate(coord); err != nil {
		t.Fatalf("writing coordinate: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/coordinates", nil)
	w := httptest.NewRecorder()
	srv.handleCoordinates(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var coords []model.Coordinate
	if err := json.NewDecoder(w.Body).Decode(&coords); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(coords) != 1 {
		t.Errorf("expected 1 coordinate, got %d", len(coords))
	}
}

func TestWriteJSONNil(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, nil)

	if w.Body.String() != "[]" {
		t.Errorf("expected '[]' for nil, got %q", w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

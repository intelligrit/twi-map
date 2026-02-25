package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intelligrit/twi-map/internal/model"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "twi-map-store-test-"+t.Name())
	os.RemoveAll(dir)
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, err := New(dir)
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestTOCRoundTrip(t *testing.T) {
	s := testStore(t)

	toc := &model.TOC{
		ScrapedAt: "2025-01-01T00:00:00Z",
		Chapters: []model.Chapter{
			{Index: 0, WebTitle: "1.00", URL: "https://example.com/1-00", Volume: "vol-1", Slug: "1-00"},
			{Index: 1, WebTitle: "1.01", URL: "https://example.com/1-01", Volume: "vol-1", Slug: "1-01"},
		},
	}

	if err := s.WriteTOC(toc); err != nil {
		t.Fatalf("writing TOC: %v", err)
	}

	got, err := s.ReadTOC()
	if err != nil {
		t.Fatalf("reading TOC: %v", err)
	}

	if len(got.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(got.Chapters))
	}
	if got.Chapters[0].WebTitle != "1.00" {
		t.Errorf("expected title '1.00', got %q", got.Chapters[0].WebTitle)
	}
	if got.ScrapedAt != "2025-01-01T00:00:00Z" {
		t.Errorf("expected scraped_at preserved, got %q", got.ScrapedAt)
	}
}

func TestChapterTextRoundTrip(t *testing.T) {
	s := testStore(t)

	// Need a TOC entry first (foreign key)
	toc := &model.TOC{
		Chapters: []model.Chapter{
			{Index: 0, WebTitle: "1.00", URL: "https://example.com", Volume: "vol-1", Slug: "1-00"},
		},
	}
	if err := s.WriteTOC(toc); err != nil {
		t.Fatalf("writing TOC: %v", err)
	}

	text := "Erin Solstice looked at the inn."
	if err := s.WriteChapterText(0, text); err != nil {
		t.Fatalf("writing chapter text: %v", err)
	}

	got, err := s.ReadChapterText(0)
	if err != nil {
		t.Fatalf("reading chapter text: %v", err)
	}
	if got != text {
		t.Errorf("text mismatch: got %q", got)
	}

	if !s.ChapterTextExists(0) {
		t.Error("expected ChapterTextExists(0) = true")
	}
	if s.ChapterTextExists(999) {
		t.Error("expected ChapterTextExists(999) = false")
	}
}

func TestExtractionRoundTrip(t *testing.T) {
	s := testStore(t)

	toc := &model.TOC{
		Chapters: []model.Chapter{
			{Index: 0, WebTitle: "1.00", URL: "https://example.com", Volume: "vol-1", Slug: "1-00"},
		},
	}
	if err := s.WriteTOC(toc); err != nil {
		t.Fatalf("writing TOC: %v", err)
	}

	ext := &model.ChapterExtraction{
		ChapterIndex: 0,
		Model:        "test-model",
		ExtractedAt:  "2025-01-01T00:00:00Z",
		Locations: []model.ExtractedLocation{
			{Name: "Liscor", Type: "city", Description: "A walled city"},
		},
		Relationships: []model.ExtractedRelationship{
			{From: "Liscor", To: "Izril", Type: "containment"},
		},
		Containment: []model.Containment{
			{Child: "Liscor", Parent: "Izril"},
		},
	}

	if err := s.WriteExtraction(ext); err != nil {
		t.Fatalf("writing extraction: %v", err)
	}

	if !s.ExtractionExists(0) {
		t.Error("expected ExtractionExists(0) = true")
	}

	got, err := s.ReadExtraction(0)
	if err != nil {
		t.Fatalf("reading extraction: %v", err)
	}

	if len(got.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(got.Locations))
	}
	if got.Locations[0].Name != "Liscor" {
		t.Errorf("expected location name 'Liscor', got %q", got.Locations[0].Name)
	}
	if len(got.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(got.Relationships))
	}
	if len(got.Containment) != 1 {
		t.Errorf("expected 1 containment, got %d", len(got.Containment))
	}
}

func TestAggregatedRoundTrip(t *testing.T) {
	s := testStore(t)

	data := &model.AggregatedData{
		AggregatedAt: "2025-01-01T00:00:00Z",
		Locations: []model.AggregatedLocation{
			{ID: "liscor", Name: "Liscor", Type: "city", Description: "A walled city", MentionCount: 50, FirstChapterIndex: 0},
		},
		Relationships: []model.AggregatedRelationship{
			{From: "liscor", To: "izril", Type: "containment", FirstChapterIndex: 0},
		},
		Containment: []model.Containment{
			{Child: "liscor", Parent: "izril"},
		},
	}

	if err := s.WriteAggregated(data); err != nil {
		t.Fatalf("writing aggregated: %v", err)
	}

	got, err := s.ReadAggregated()
	if err != nil {
		t.Fatalf("reading aggregated: %v", err)
	}

	if len(got.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(got.Locations))
	}
	if got.Locations[0].Name != "Liscor" {
		t.Errorf("expected 'Liscor', got %q", got.Locations[0].Name)
	}
	if len(got.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(got.Relationships))
	}
	if len(got.Containment) != 1 {
		t.Errorf("expected 1 containment, got %d", len(got.Containment))
	}
}

func TestCoordinateRoundTrip(t *testing.T) {
	s := testStore(t)

	coord := model.Coordinate{LocationID: "liscor", X: 240, Y: -20, Confidence: "estimated"}
	if err := s.WriteCoordinate(coord); err != nil {
		t.Fatalf("writing coordinate: %v", err)
	}

	coords, err := s.ReadCoordinates()
	if err != nil {
		t.Fatalf("reading coordinates: %v", err)
	}

	if len(coords) != 1 {
		t.Fatalf("expected 1 coordinate, got %d", len(coords))
	}
	if coords[0].LocationID != "liscor" || coords[0].X != 240 || coords[0].Y != -20 {
		t.Errorf("coordinate mismatch: %+v", coords[0])
	}
}

func TestCountMethods(t *testing.T) {
	s := testStore(t)

	if s.ChapterCount() != 0 {
		t.Errorf("expected 0 chapters, got %d", s.ChapterCount())
	}

	toc := &model.TOC{
		Chapters: []model.Chapter{
			{Index: 0, WebTitle: "1.00", URL: "https://example.com", Volume: "vol-1", Slug: "1-00"},
		},
	}
	if err := s.WriteTOC(toc); err != nil {
		t.Fatalf("writing TOC: %v", err)
	}

	if s.ChapterCount() != 1 {
		t.Errorf("expected 1 chapter, got %d", s.ChapterCount())
	}
}

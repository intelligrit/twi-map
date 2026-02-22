package aggregator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robertmeta/twi-map/internal/model"
	"github.com/robertmeta/twi-map/internal/store"
)

func TestAggregate(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "twi-map-test-agg")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("creating store: %v", err)
	}
	defer s.Close()

	// Insert TOC (need enough chapters so locations meet minMentions=3 filter)
	toc := &model.TOC{
		Chapters: []model.Chapter{
			{Index: 0, WebTitle: "1.00", Slug: "1-00", Volume: "vol-1"},
			{Index: 1, WebTitle: "1.01", Slug: "1-01", Volume: "vol-1"},
			{Index: 2, WebTitle: "1.02", Slug: "1-02", Volume: "vol-1"},
		},
	}
	if err := s.WriteTOC(toc); err != nil {
		t.Fatalf("writing TOC: %v", err)
	}

	// Insert extractions — each location needs >=3 mentions across chapters
	ext1 := &model.ChapterExtraction{
		ChapterIndex: 0,
		ChapterTitle: "1.00",
		Model:        "test",
		ExtractedAt:  "2024-01-01",
		Locations: []model.ExtractedLocation{
			{Name: "Liscor", Type: "city", Description: "A walled city"},
			{Name: "The Wandering Inn", Type: "building", Description: "An old inn"},
			{Name: "Izril", Type: "continent", Description: "A continent"},
		},
		Relationships: []model.ExtractedRelationship{
			{From: "The Wandering Inn", To: "Liscor", Type: "adjacency", Detail: "near Liscor"},
		},
		Containment: []model.Containment{
			{Child: "Liscor", Parent: "Izril"},
		},
	}
	ext2 := &model.ChapterExtraction{
		ChapterIndex: 1,
		ChapterTitle: "1.01",
		Model:        "test",
		ExtractedAt:  "2024-01-01",
		Locations: []model.ExtractedLocation{
			{Name: "Liscor", Type: "city", Description: "A walled city in the south of Izril"},
			{Name: "The Wandering Inn", Type: "building", Description: "An inn outside Liscor"},
			{Name: "Izril", Type: "continent", Description: "Main continent"},
		},
	}
	ext3 := &model.ChapterExtraction{
		ChapterIndex: 2,
		ChapterTitle: "1.02",
		Model:        "test",
		ExtractedAt:  "2024-01-01",
		Locations: []model.ExtractedLocation{
			{Name: "Liscor", Type: "city", Description: "Liscor again"},
			{Name: "The Wandering Inn", Type: "building", Description: "The old inn"},
			{Name: "Izril", Type: "continent", Description: "Izril continent"},
		},
	}

	if err := s.WriteExtraction(ext1); err != nil {
		t.Fatalf("writing extraction 1: %v", err)
	}
	if err := s.WriteExtraction(ext2); err != nil {
		t.Fatalf("writing extraction 2: %v", err)
	}
	if err := s.WriteExtraction(ext3); err != nil {
		t.Fatalf("writing extraction 3: %v", err)
	}

	data, err := Aggregate(s)
	if err != nil {
		t.Fatalf("aggregation failed: %v", err)
	}

	if len(data.Locations) < 3 {
		t.Errorf("expected at least 3 locations, got %d", len(data.Locations))
	}

	// Find Liscor
	var liscor *model.AggregatedLocation
	for i, loc := range data.Locations {
		if loc.Name == "Liscor" {
			liscor = &data.Locations[i]
			break
		}
	}
	if liscor == nil {
		t.Fatal("Liscor not found in aggregated data")
	}
	if liscor.MentionCount != 3 {
		t.Errorf("expected Liscor mention count 3, got %d", liscor.MentionCount)
	}
	if liscor.FirstChapterIndex != 0 {
		t.Errorf("expected Liscor first chapter 0, got %d", liscor.FirstChapterIndex)
	}
	// Should have the longer description
	if liscor.Description != "A walled city in the south of Izril" {
		t.Errorf("expected longer description, got %q", liscor.Description)
	}

	// Check relationships preserved
	if len(data.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(data.Relationships))
	}

	// Check containment preserved
	if len(data.Containment) != 1 {
		t.Errorf("expected 1 containment, got %d", len(data.Containment))
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Liscor", "liscor"},
		{"  The Wandering Inn  ", "the wandering inn"},
		{"IZRIL", "izril"},
	}
	for _, tt := range tests {
		got := normalizeName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
